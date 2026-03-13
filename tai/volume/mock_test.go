package volume

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	pb "github.com/yaoapp/yao/tai/volume/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockVolumeServer struct {
	pb.UnimplementedVolumeServer
	statErr     error
	removeOK    bool
	removeError string
	renameOK    bool
	renameError string
	mkdirOK     bool
	mkdirError  string
}

func (m *mockVolumeServer) Stat(_ context.Context, req *pb.FSRequest) (*pb.FileInfo, error) {
	if m.statErr != nil {
		return nil, m.statErr
	}
	return &pb.FileInfo{Path: req.Path, Size: 42, IsDir: false}, nil
}

func (m *mockVolumeServer) Remove(_ context.Context, req *pb.FSRemoveRequest) (*pb.FSOpResponse, error) {
	return &pb.FSOpResponse{Ok: m.removeOK, Error: m.removeError}, nil
}

func (m *mockVolumeServer) Rename(_ context.Context, req *pb.FSRenameRequest) (*pb.FSOpResponse, error) {
	return &pb.FSOpResponse{Ok: m.renameOK, Error: m.renameError}, nil
}

func (m *mockVolumeServer) MkdirAll(_ context.Context, req *pb.FSRequest) (*pb.FSOpResponse, error) {
	return &pb.FSOpResponse{Ok: m.mkdirOK, Error: m.mkdirError}, nil
}

func (m *mockVolumeServer) ReadFile(req *pb.FSReadRequest, stream grpc.ServerStreamingServer[pb.FSDataChunk]) error {
	return fmt.Errorf("file not found: %s", req.Path)
}

func (m *mockVolumeServer) WriteFile(stream grpc.ClientStreamingServer[pb.FSWriteChunk, pb.FSWriteResponse]) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.FSWriteResponse{Size: 0})
		}
		if err != nil {
			return err
		}
	}
}

func (m *mockVolumeServer) SyncPush(stream grpc.BidiStreamingServer[pb.SyncMessage, pb.SyncMessage]) error {
	// Receive manifest
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	manifest := msg.GetManifest()
	if manifest == nil {
		return fmt.Errorf("expected manifest")
	}

	// Respond with diff: request all files + a ghost delete
	var needFiles []string
	for _, f := range manifest.Files {
		if !f.IsDir {
			needFiles = append(needFiles, f.Path)
		}
	}
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Diff{
			Diff: &pb.SyncDiff{
				NeedFiles:   needFiles,
				DeleteFiles: []string{"old-deleted.txt"},
			},
		},
	}); err != nil {
		return err
	}

	// Receive file chunks until CloseSend
	var synced int32
	var transferred int64
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if chunk := msg.GetChunk(); chunk != nil && chunk.Eof {
			synced++
			transferred += int64(len(chunk.Data))
		}
	}

	// Send result
	return stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Result{
			Result: &pb.SyncResult{
				FilesSynced:      synced,
				BytesTransferred: transferred,
			},
		},
	})
}

func (m *mockVolumeServer) SyncPull(req *pb.SyncManifest, stream grpc.ServerStreamingServer[pb.SyncMessage]) error {
	// Send MKDIR
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Chunk{
			Chunk: &pb.FileChunk{Path: "newdir", Type: pb.FileChunk_MKDIR},
		},
	}); err != nil {
		return err
	}

	// Send DELETE
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Chunk{
			Chunk: &pb.FileChunk{Path: "old-file.txt", Type: pb.FileChunk_DELETE},
		},
	}); err != nil {
		return err
	}

	// Send a file (FULL, multi-chunk)
	data := []byte("mock pull content")
	compressed, err := compress(data)
	if err != nil {
		return err
	}

	half := len(compressed) / 2
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Chunk{
			Chunk: &pb.FileChunk{
				Path:  "pulled.txt",
				Type:  pb.FileChunk_FULL,
				Data:  compressed[:half],
				Eof:   false,
				Mode:  0o644,
				Mtime: 1234567890000000000,
			},
		},
	}); err != nil {
		return err
	}

	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Chunk{
			Chunk: &pb.FileChunk{
				Path: "pulled.txt",
				Type: pb.FileChunk_FULL,
				Data: compressed[half:],
				Eof:  true,
			},
		},
	}); err != nil {
		return err
	}

	// Send a file with no mode (tests default 0o644)
	data2 := []byte("no mode")
	c2, _ := compress(data2)
	if err := stream.Send(&pb.SyncMessage{
		Payload: &pb.SyncMessage_Chunk{
			Chunk: &pb.FileChunk{
				Path: "nomode.txt",
				Type: pb.FileChunk_FULL,
				Data: c2,
				Eof:  true,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func (m *mockVolumeServer) Abs(_ context.Context, req *pb.FSRequest) (*pb.FSAbsResponse, error) {
	return &pb.FSAbsResponse{Path: "/data/" + req.SessionId + "/" + req.Path}, nil
}

func (m *mockVolumeServer) ListDir(_ context.Context, req *pb.FSRequest) (*pb.FSListResponse, error) {
	return &pb.FSListResponse{Entries: []*pb.FileInfo{
		{Path: "a.txt", Size: 10},
		{Path: "b.txt", Size: 20, IsDir: true},
	}}, nil
}

func (m *mockVolumeServer) Copy(_ context.Context, req *pb.FSCopyRequest) (*pb.SyncResult, error) {
	return &pb.SyncResult{
		FilesSynced:      1,
		BytesTransferred: 42,
	}, nil
}

func startMockServer(t *testing.T, mock *mockVolumeServer) (*grpc.ClientConn, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterVolumeServer(srv, mock)

	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		t.Fatalf("dial: %v", err)
	}

	return conn, func() {
		conn.Close()
		srv.Stop()
	}
}

func TestMockRemoteStat(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	info, err := vol.Stat(context.Background(), "s1", "test.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != 42 {
		t.Errorf("size = %d, want 42", info.Size)
	}
}

func TestMockRemoteStatError(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{statErr: fmt.Errorf("boom")})
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.Stat(context.Background(), "s1", "test.txt")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteRemoveFail(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{removeOK: false, removeError: "no such file"})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Remove(context.Background(), "s1", "bad.txt", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteRemoveOK(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{removeOK: true})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Remove(context.Background(), "s1", "good.txt", false)
	if err != nil {
		t.Errorf("Remove: %v", err)
	}
}

func TestMockRemoteRenameFail(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{renameOK: false, renameError: "bad"})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Rename(context.Background(), "s1", "a", "b")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteRenameOK(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{renameOK: true})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Rename(context.Background(), "s1", "a", "b")
	if err != nil {
		t.Errorf("Rename: %v", err)
	}
}

func TestMockRemoteMkdirFail(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{mkdirOK: false, mkdirError: "perm denied"})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.MkdirAll(context.Background(), "s1", "dir")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteMkdirOK(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{mkdirOK: true})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.MkdirAll(context.Background(), "s1", "dir")
	if err != nil {
		t.Errorf("MkdirAll: %v", err)
	}
}

func TestMockRemoteReadFileError(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	_, _, err := vol.ReadFile(context.Background(), "s1", "missing.txt")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteWriteFile(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.WriteFile(context.Background(), "s1", "test.txt", []byte("hello"), 0o644)
	if err != nil {
		t.Errorf("WriteFile: %v", err)
	}
}

func TestMockRemoteWriteFileLarge(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	data := make([]byte, 200*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	err := vol.WriteFile(context.Background(), "s1", "large.bin", data, 0o644)
	if err != nil {
		t.Errorf("WriteFile large: %v", err)
	}
}

func TestMockRemoteWriteFileEmpty(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.WriteFile(context.Background(), "s1", "empty.txt", []byte{}, 0o644)
	if err != nil {
		t.Errorf("WriteFile empty: %v", err)
	}
}

func TestMockRemoteListDir(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	entries, err := vol.ListDir(context.Background(), "s1", ".")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestMockRemoteClose(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	if err := vol.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestMockRemoteSyncPush(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	srcDir := t.TempDir()
	_ = os.WriteFile(srcDir+"/a.txt", []byte("aaa"), 0o644)
	_ = os.Mkdir(srcDir+"/sub", 0o755)
	_ = os.WriteFile(srcDir+"/sub/b.txt", []byte("bbb"), 0o644)

	result, err := vol.SyncPush(context.Background(), "s1", srcDir)
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced < 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}
}

func TestMockRemoteSyncPushWithExcludes(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	srcDir := t.TempDir()
	_ = os.WriteFile(srcDir+"/keep.txt", []byte("keep"), 0o644)
	_ = os.WriteFile(srcDir+"/skip.log", []byte("skip"), 0o644)

	result, err := vol.SyncPush(context.Background(), "s1", srcDir, WithExcludes("*.log"))
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}
}

func TestMockRemoteSyncPushForceFull(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	srcDir := t.TempDir()
	_ = os.WriteFile(srcDir+"/a.txt", []byte("aaa"), 0o644)

	result, err := vol.SyncPush(context.Background(), "s1", srcDir, WithForceFull())
	if err != nil {
		t.Fatalf("SyncPush: %v", err)
	}
	if result.FilesSynced < 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}
}

func TestMockRemoteSyncPull(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	dstDir := t.TempDir()

	// Create a file that the mock will ask to DELETE
	_ = os.WriteFile(dstDir+"/old-file.txt", []byte("old"), 0o644)

	result, err := vol.SyncPull(context.Background(), "s1", dstDir)
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced < 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}

	// Verify pulled file
	data, err := os.ReadFile(dstDir + "/pulled.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "mock pull content" {
		t.Errorf("content = %q", data)
	}

	// Verify MKDIR was created
	info, err := os.Stat(dstDir + "/newdir")
	if err != nil {
		t.Fatalf("MKDIR dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected dir")
	}

	// Verify DELETE removed the file
	if _, err := os.Stat(dstDir + "/old-file.txt"); err == nil {
		t.Error("DELETE file should be removed")
	}

	// Verify nomode.txt was created
	data, err = os.ReadFile(dstDir + "/nomode.txt")
	if err != nil {
		t.Fatalf("ReadFile nomode: %v", err)
	}
	if string(data) != "no mode" {
		t.Errorf("nomode content = %q", data)
	}
}

func TestMockRemoteSyncPullWithLocalFiles(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	dstDir := t.TempDir()
	_ = os.WriteFile(dstDir+"/existing.txt", []byte("exist"), 0o644)

	result, err := vol.SyncPull(context.Background(), "s1", dstDir)
	if err != nil {
		t.Fatalf("SyncPull: %v", err)
	}
	if result.FilesSynced < 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}
}

// errMockVolumeServer returns errors mid-stream for error-path testing.
type errMockVolumeServer struct {
	pb.UnimplementedVolumeServer
}

func (m *errMockVolumeServer) SyncPush(stream grpc.BidiStreamingServer[pb.SyncMessage, pb.SyncMessage]) error {
	_, _ = stream.Recv()
	return fmt.Errorf("injected push error")
}

func (m *errMockVolumeServer) SyncPull(_ *pb.SyncManifest, stream grpc.ServerStreamingServer[pb.SyncMessage]) error {
	return fmt.Errorf("injected pull error")
}

func (m *errMockVolumeServer) ReadFile(_ *pb.FSReadRequest, _ grpc.ServerStreamingServer[pb.FSDataChunk]) error {
	return fmt.Errorf("injected read error")
}

func (m *errMockVolumeServer) WriteFile(stream grpc.ClientStreamingServer[pb.FSWriteChunk, pb.FSWriteResponse]) error {
	return fmt.Errorf("injected write error")
}

func (m *errMockVolumeServer) Stat(_ context.Context, _ *pb.FSRequest) (*pb.FileInfo, error) {
	return nil, fmt.Errorf("injected stat error")
}

func (m *errMockVolumeServer) ListDir(_ context.Context, _ *pb.FSRequest) (*pb.FSListResponse, error) {
	return nil, fmt.Errorf("injected listdir error")
}

func (m *errMockVolumeServer) Remove(_ context.Context, _ *pb.FSRemoveRequest) (*pb.FSOpResponse, error) {
	return nil, fmt.Errorf("injected remove error")
}

func (m *errMockVolumeServer) Rename(_ context.Context, _ *pb.FSRenameRequest) (*pb.FSOpResponse, error) {
	return nil, fmt.Errorf("injected rename error")
}

func (m *errMockVolumeServer) MkdirAll(_ context.Context, _ *pb.FSRequest) (*pb.FSOpResponse, error) {
	return nil, fmt.Errorf("injected mkdir error")
}

func (m *errMockVolumeServer) Copy(_ context.Context, _ *pb.FSCopyRequest) (*pb.SyncResult, error) {
	return nil, fmt.Errorf("injected copy error")
}

func (m *errMockVolumeServer) Abs(_ context.Context, _ *pb.FSRequest) (*pb.FSAbsResponse, error) {
	return nil, fmt.Errorf("injected abs error")
}

func startErrMockServer(t *testing.T) (*grpc.ClientConn, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterVolumeServer(srv, &errMockVolumeServer{})
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		t.Fatalf("dial: %v", err)
	}
	return conn, func() { conn.Close(); srv.Stop() }
}

func TestErrRemoteSyncPush(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	srcDir := t.TempDir()
	_ = os.WriteFile(srcDir+"/a.txt", []byte("aaa"), 0o644)

	_, err := vol.SyncPush(context.Background(), "s1", srcDir)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteSyncPull(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	dstDir := t.TempDir()

	_, err := vol.SyncPull(context.Background(), "s1", dstDir)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteWriteFile(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.WriteFile(context.Background(), "s1", "test.txt", []byte("x"), 0o644)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteReadFile(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, _, err := vol.ReadFile(context.Background(), "s1", "test.txt")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteStat(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.Stat(context.Background(), "s1", "test.txt")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteListDir(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.ListDir(context.Background(), "s1", ".")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteRemove(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Remove(context.Background(), "s1", "test.txt", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteRename(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.Rename(context.Background(), "s1", "a", "b")
	if err == nil {
		t.Error("expected error")
	}
}

func TestErrRemoteMkdirAll(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	err := vol.MkdirAll(context.Background(), "s1", "dir")
	if err == nil {
		t.Error("expected error")
	}
}

func TestPbToFileInfo(t *testing.T) {
	fi := pbToFileInfo(&pb.FileInfo{
		Path:  "test.txt",
		Size:  100,
		Mtime: 1234567890000000000,
		Mode:  0o644,
		IsDir: false,
	})
	if fi.Path != "test.txt" {
		t.Errorf("path = %q", fi.Path)
	}
	if fi.Size != 100 {
		t.Errorf("size = %d", fi.Size)
	}
	if fi.IsDir {
		t.Error("expected not dir")
	}
	if fi.Mode != os.FileMode(0o644) {
		t.Errorf("mode = %v", fi.Mode)
	}
}

func TestMockRemoteCopy(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.Copy(context.Background(), "s1", "src.txt", "dst.txt")
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d, want 1", result.FilesSynced)
	}
	if result.BytesTransferred != 42 {
		t.Errorf("bytes = %d, want 42", result.BytesTransferred)
	}
}

func TestMockRemoteCopyWithOpts(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	result, err := vol.Copy(context.Background(), "s1", "src", "dst",
		WithExcludes("*.log"), WithForceFull())
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.FilesSynced != 1 {
		t.Errorf("synced = %d", result.FilesSynced)
	}
}

func TestErrRemoteCopy(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.Copy(context.Background(), "s1", "a", "b")
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockRemoteAbs(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	got, err := vol.Abs(context.Background(), "s1", ".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	if got != "/data/s1/." {
		t.Errorf("Abs = %q, want %q", got, "/data/s1/.")
	}
}

func TestMockRemoteAbsRelative(t *testing.T) {
	conn, cleanup := startMockServer(t, &mockVolumeServer{})
	defer cleanup()

	vol := NewRemote(conn)
	got, err := vol.Abs(context.Background(), "s1", "sub/file.txt")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	if got != "/data/s1/sub/file.txt" {
		t.Errorf("Abs = %q, want %q", got, "/data/s1/sub/file.txt")
	}
}

func TestErrRemoteAbs(t *testing.T) {
	conn, cleanup := startErrMockServer(t)
	defer cleanup()

	vol := NewRemote(conn)
	_, err := vol.Abs(context.Background(), "s1", ".")
	if err == nil {
		t.Error("expected error")
	}
}
