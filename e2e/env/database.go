package env

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Database is one isolated test database.
type Database interface {
	Name() string
	Driver() string
	DSN() string
	Close() error
}

func newDatabase(opts Options, tempDir string) (Database, error) {
	switch opts.DB {
	case "postgres":
		return newPostgres(opts)
	case "sqlite3":
		path := filepath.Join(tempDir, "test.db")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		return &sqliteDatabase{path: path}, nil
	default:
		return nil, fmt.Errorf("unsupported database: %s", opts.DB)
	}
}

func platformName() string { return runtime.GOOS }

func randomSuffix() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(buf)
}

type sqliteDatabase struct {
	path string
}

func (d *sqliteDatabase) Name() string   { return filepath.Base(d.path) }
func (d *sqliteDatabase) Driver() string { return "sqlite3" }
func (d *sqliteDatabase) DSN() string    { return d.path }
func (d *sqliteDatabase) Close() error   { return nil }
