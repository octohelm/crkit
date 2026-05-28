package driver

import (
	"context"
	"io"
	"io/fs"

	"github.com/octohelm/unifs/pkg/filesystem"
)

// +gengo:injectable:provider
type Driver interface {
	WalkDir(ctx context.Context, path string, fn fs.WalkDirFunc) error
	Stat(ctx context.Context, path string) (filesystem.FileInfo, error)

	Reader(ctx context.Context, path string) (io.ReadCloser, error)
	Writer(ctx context.Context, path string, append bool) (FileWriter, error)

	Delete(ctx context.Context, path string) error

	Move(ctx context.Context, oldPath string, newPath string) error

	GetContent(ctx context.Context, path string) ([]byte, error)
	PutContent(ctx context.Context, path string, data []byte) error
}

type FileWriter interface {
	io.WriteCloser
	Size() int64
	Cancel(context.Context) error
	Commit(context.Context) error
}
