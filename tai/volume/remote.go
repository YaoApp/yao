package volume

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pierrec/lz4/v4"
	pb "github.com/yaoapp/yao/tai/volume/pb"
	"google.golang.org/grpc"
)

const (
	grpcReadChunk = 64 * 1024  // 64KB per FS IO message
	grpcSyncChunk = 256 * 1024 // 256KB per sync message
)

type remoteStorage struct {
	conn   *grpc.ClientConn
	client pb.VolumeClient
}

// NewRemote creates a Volume backed by gRPC calls to a Tai server.
func NewRemote(conn *grpc.ClientConn) Volume {
	return &remoteStorage{
		conn:   conn,
		client: pb.NewVolumeClient(conn),
	}
}

func (r *remoteStorage) ReadFile(ctx context.Context, sessionID, path string) ([]byte, os.FileMode, error) {
	stream, err := r.client.ReadFile(ctx, &pb.FSReadRequest{
		SessionId: sessionID,
		Path:      path,
	})
	if err != nil {
		return nil, 0, err
	}

	var buf bytes.Buffer
	var mode os.FileMode
	first := true
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		buf.Write(chunk.Data)
		if first {
			mode = os.FileMode(chunk.Mode)
			first = false
		}
	}
	return buf.Bytes(), mode, nil
}

func (r *remoteStorage) WriteFile(ctx context.Context, sessionID, path string, data []byte, perm os.FileMode) error {
	stream, err := r.client.WriteFile(ctx)
	if err != nil {
		return err
	}

	for offset := 0; offset <= len(data); offset += grpcReadChunk {
		end := offset + grpcReadChunk
		if end > len(data) {
			end = len(data)
		}

		chunk := &pb.FSWriteChunk{Data: data[offset:end]}
		if offset == 0 {
			chunk.SessionId = sessionID
			chunk.Path = path
			chunk.Mode = uint32(perm)
			chunk.CreateDirs = true
		}

		if err := stream.Send(chunk); err != nil {
			return err
		}
		if end == len(data) && offset > 0 {
			break
		}
		if offset == 0 && len(data) == 0 {
			break
		}
	}

	_, err = stream.CloseAndRecv()
	return err
}

func (r *remoteStorage) Stat(ctx context.Context, sessionID, path string) (*FileInfo, error) {
	info, err := r.client.Stat(ctx, &pb.FSRequest{
		SessionId: sessionID,
		Path:      path,
	})
	if err != nil {
		return nil, err
	}
	return pbToFileInfo(info), nil
}

func (r *remoteStorage) ListDir(ctx context.Context, sessionID, path string) ([]FileInfo, error) {
	resp, err := r.client.ListDir(ctx, &pb.FSRequest{
		SessionId: sessionID,
		Path:      path,
	})
	if err != nil {
		return nil, err
	}
	result := make([]FileInfo, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		result = append(result, *pbToFileInfo(e))
	}
	return result, nil
}

func (r *remoteStorage) Remove(ctx context.Context, sessionID, path string, recursive bool) error {
	resp, err := r.client.Remove(ctx, &pb.FSRemoveRequest{
		SessionId: sessionID,
		Path:      path,
		Recursive: recursive,
	})
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("remove: %s", resp.Error)
	}
	return nil
}

func (r *remoteStorage) Rename(ctx context.Context, sessionID, oldPath, newPath string) error {
	resp, err := r.client.Rename(ctx, &pb.FSRenameRequest{
		SessionId: sessionID,
		OldPath:   oldPath,
		NewPath:   newPath,
	})
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("rename: %s", resp.Error)
	}
	return nil
}

func (r *remoteStorage) MkdirAll(ctx context.Context, sessionID, path string) error {
	resp, err := r.client.MkdirAll(ctx, &pb.FSRequest{
		SessionId: sessionID,
		Path:      path,
	})
	if err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("mkdir: %s", resp.Error)
	}
	return nil
}

func (r *remoteStorage) Abs(ctx context.Context, sessionID, path string) (string, error) {
	resp, err := r.client.Abs(ctx, &pb.FSRequest{
		SessionId: sessionID,
		Path:      path,
	})
	if err != nil {
		return "", err
	}
	return resp.Path, nil
}

// SyncPush sends local files to Tai using the manifest-first bidi streaming protocol.
func (r *remoteStorage) SyncPush(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := ApplySyncOpts(opts)

	// Scan local directory
	var manifest []*pb.FileInfo
	err := filepath.WalkDir(localDir, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(localDir, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel, d.IsDir(), cfg.Excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		manifest = append(manifest, &pb.FileInfo{
			Path:  rel,
			Size:  info.Size(),
			Mtime: info.ModTime().UnixNano(),
			Mode:  uint32(info.Mode()),
			IsDir: d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan local: %w", err)
	}

	stream, err := r.client.SyncPush(ctx)
	if err != nil {
		return nil, err
	}

	// Step 1: send manifest
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Manifest{
			Manifest: &pb.SyncManifest{
				SessionId:  sessionID,
				Files:      manifest,
				ForceFull:  cfg.ForceFull,
				RemotePath: cfg.RemotePath,
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("send manifest: %w", err)
	}

	// Step 2: receive diff
	msg, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv diff: %w", err)
	}
	diff := msg.GetDiff()
	if diff == nil {
		return nil, fmt.Errorf("expected SyncDiff, got %T", msg.Payload)
	}

	// Step 3: send needed files
	var bytesTransferred int64
	for _, path := range diff.NeedFiles {
		abs := filepath.Join(localDir, filepath.FromSlash(path))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		compressed, err := compress(data)
		if err != nil {
			continue
		}

		info, _ := os.Stat(abs)
		for offset := 0; offset < len(compressed); offset += grpcSyncChunk {
			end := offset + grpcSyncChunk
			if end > len(compressed) {
				end = len(compressed)
			}
			chunk := &pb.FileChunk{
				Path: path,
				Type: pb.FileChunk_FULL,
				Data: compressed[offset:end],
				Eof:  end == len(compressed),
			}
			if offset == 0 && info != nil {
				chunk.Mode = uint32(info.Mode())
				chunk.Mtime = info.ModTime().UnixNano()
			}
			if err := stream.Send(&pb.SyncMessage{
				Payload: &pb.SyncMessage_Chunk{Chunk: chunk},
			}); err != nil {
				return nil, err
			}
			bytesTransferred += int64(len(chunk.Data))
		}
	}

	// Send deletes
	for _, path := range diff.DeleteFiles {
		_ = stream.Send(&pb.SyncMessage{
			Payload: &pb.SyncMessage_Chunk{
				Chunk: &pb.FileChunk{Path: path, Type: pb.FileChunk_DELETE},
			},
		})
	}

	if err := stream.CloseSend(); err != nil {
		return nil, err
	}

	// Step 4: receive result
	msg, err = stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("recv result: %w", err)
	}
	result := msg.GetResult()
	if result == nil {
		return &SyncResult{
			FilesSynced:      len(diff.NeedFiles),
			BytesTransferred: bytesTransferred,
			Duration:         time.Since(start),
		}, nil
	}

	return &SyncResult{
		FilesSynced:      int(result.FilesSynced),
		BytesTransferred: result.BytesTransferred,
		Duration:         time.Since(start),
	}, nil
}

// SyncPull receives changed files from Tai.
func (r *remoteStorage) SyncPull(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := ApplySyncOpts(opts)

	// Build local manifest
	var manifest []*pb.FileInfo
	_ = filepath.WalkDir(localDir, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(localDir, abs)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel, d.IsDir(), cfg.Excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		manifest = append(manifest, &pb.FileInfo{
			Path:  rel,
			Size:  info.Size(),
			Mtime: info.ModTime().UnixNano(),
			Mode:  uint32(info.Mode()),
			IsDir: d.IsDir(),
		})
		return nil
	})

	stream, err := r.client.SyncPull(ctx, &pb.SyncManifest{
		SessionId:  sessionID,
		Files:      manifest,
		ForceFull:  cfg.ForceFull,
		RemotePath: cfg.RemotePath,
	})
	if err != nil {
		return nil, err
	}

	buffers := make(map[string][]byte)
	modes := make(map[string]os.FileMode)
	mtimes := make(map[string]int64)
	var synced int
	var transferred int64

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if result := msg.GetResult(); result != nil {
			return &SyncResult{
				FilesSynced:      int(result.FilesSynced),
				BytesTransferred: result.BytesTransferred,
				Duration:         time.Since(start),
			}, nil
		}

		chunk := msg.GetChunk()
		if chunk == nil {
			continue
		}

		switch chunk.Type {
		case pb.FileChunk_FULL:
			buffers[chunk.Path] = append(buffers[chunk.Path], chunk.Data...)
			transferred += int64(len(chunk.Data))
			if chunk.Mode != 0 {
				modes[chunk.Path] = os.FileMode(chunk.Mode)
			}
			if chunk.Mtime != 0 {
				mtimes[chunk.Path] = chunk.Mtime
			}

			if chunk.Eof {
				decompressed, err := decompress(buffers[chunk.Path])
				if err != nil {
					delete(buffers, chunk.Path)
					continue
				}
				delete(buffers, chunk.Path)

				target := filepath.Join(localDir, filepath.FromSlash(chunk.Path))
				_ = os.MkdirAll(filepath.Dir(target), 0o755)

				perm := modes[chunk.Path]
				if perm == 0 {
					perm = 0o644
				}
				if err := os.WriteFile(target, decompressed, perm); err != nil {
					continue
				}
				if mt, ok := mtimes[chunk.Path]; ok {
					t := time.Unix(0, mt)
					_ = os.Chtimes(target, t, t)
				}
				synced++
			}

		case pb.FileChunk_DELETE:
			target := filepath.Join(localDir, filepath.FromSlash(chunk.Path))
			_ = os.RemoveAll(target)

		case pb.FileChunk_MKDIR:
			target := filepath.Join(localDir, filepath.FromSlash(chunk.Path))
			_ = os.MkdirAll(target, 0o755)
		}
	}

	return &SyncResult{
		FilesSynced:      synced,
		BytesTransferred: transferred,
		Duration:         time.Since(start),
	}, nil
}

func (r *remoteStorage) Zip(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	resp, err := r.client.Zip(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst, Excludes: excludes,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Unzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	resp, err := r.client.Unzip(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Gzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	resp, err := r.client.Gzip(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Gunzip(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	resp, err := r.client.Gunzip(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Tar(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	resp, err := r.client.Tar(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst, Excludes: excludes,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Untar(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	resp, err := r.client.Untar(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Tgz(ctx context.Context, sessionID, src, dst string, excludes []string) (*ArchiveResult, error) {
	resp, err := r.client.Tgz(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst, Excludes: excludes,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Untgz(ctx context.Context, sessionID, src, dst string) (*ArchiveResult, error) {
	resp, err := r.client.Untgz(ctx, &pb.ArchiveRequest{
		SessionId: sessionID, SrcPath: src, DstPath: dst,
	})
	if err != nil {
		return nil, err
	}
	return &ArchiveResult{SizeBytes: resp.SizeBytes, FilesCount: int(resp.FilesCount)}, nil
}

func (r *remoteStorage) Copy(ctx context.Context, sessionID, src, dst string, opts ...SyncOption) (*SyncResult, error) {
	start := time.Now()
	cfg := ApplySyncOpts(opts)

	resp, err := r.client.Copy(ctx, &pb.FSCopyRequest{
		SessionId: sessionID,
		SrcPath:   src,
		DstPath:   dst,
		Excludes:  cfg.Excludes,
		Force:     cfg.ForceFull,
	})
	if err != nil {
		return nil, err
	}
	return &SyncResult{
		FilesSynced:      int(resp.FilesSynced),
		BytesTransferred: resp.BytesTransferred,
		Duration:         time.Since(start),
	}, nil
}

func (r *remoteStorage) Close() error {
	return nil
}

func pbToFileInfo(p *pb.FileInfo) *FileInfo {
	return &FileInfo{
		Path:  p.Path,
		Size:  p.Size,
		Mtime: time.Unix(0, p.Mtime),
		Mode:  fs.FileMode(p.Mode),
		IsDir: p.IsDir,
	}
}

func compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompress(src []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(src))
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
