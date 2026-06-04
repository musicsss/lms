package storage

import (
	"context"
	"io"
)

type FileInfo struct {
	Key  string
	Size int64
}

type Driver interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Range(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error)
}
