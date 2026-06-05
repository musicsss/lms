package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalDriver struct {
	root string
}

func NewLocalDriver(root string) (*LocalDriver, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &LocalDriver{root: absRoot}, nil
}

// ResolvePath returns the absolute filesystem path for a storage key.
func (d *LocalDriver) ResolvePath(key string) string {
	return d.resolvePath(key)
}

func (d *LocalDriver) resolvePath(key string) string {
	return filepath.Join(d.root, key)
}

func (d *LocalDriver) Put(ctx context.Context, key string, reader io.Reader, _ int64) error {
	path := d.resolvePath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (d *LocalDriver) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return os.Open(d.resolvePath(key))
}

func (d *LocalDriver) Delete(ctx context.Context, key string) error {
	return os.Remove(d.resolvePath(key))
}

func (d *LocalDriver) Range(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	f, err := os.Open(d.resolvePath(key))
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	return &sectionReaderCloser{f: f, limit: length}, nil
}

type sectionReaderCloser struct {
	f     *os.File
	limit int64
	read  int64
}

func (s *sectionReaderCloser) Read(p []byte) (int, error) {
	if s.limit > 0 && s.read >= s.limit {
		return 0, io.EOF
	}
	maxRead := len(p)
	if s.limit > 0 {
		remaining := s.limit - s.read
		if int64(maxRead) > remaining {
			maxRead = int(remaining)
		}
	}
	n, err := s.f.Read(p[:maxRead])
	s.read += int64(n)
	return n, err
}

func (s *sectionReaderCloser) Close() error {
	return s.f.Close()
}
