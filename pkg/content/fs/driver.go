package fs

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/unifs/pkg/filesystem"
)

func newWorkspace(fs filesystem.FileSystem, layout layout.Layout) *workspace {
	return &workspace{
		driver: driver{
			fs: fs,
		},
		layout: layout,
	}
}

type workspace struct {
	driver
	layout layout.Layout
}

type driver struct {
	fs filesystem.FileSystem
}

func (d *driver) Stat(ctx context.Context, path string) (filesystem.FileInfo, error) {
	return d.fs.Stat(ctx, path)
}

func (d *driver) Remove(ctx context.Context, path string) error {
	return d.fs.RemoveAll(ctx, path)
}

func (d *driver) PutContent(ctx context.Context, path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" {
		if err := filesystem.MkdirAll(ctx, d.fs, dir); err != nil {
			return err
		}
	}
	return filesystem.Write(ctx, d.fs, path, data)
}

func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	f, err := d.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	return io.ReadAll(f)
}

func (d *driver) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return filesystem.Open(ctx, d.fs, path)
}

func (d *driver) WalkDir(ctx context.Context, path string, fn fs.WalkDirFunc) error {
	return filesystem.WalkDir(ctx, filesystem.Sub(d.fs, path), ".", fn)
}

func (w *driver) Move(ctx context.Context, oldPath string, newPath string) error {
	if err := filesystem.MkdirAll(ctx, w.fs, filepath.Dir(newPath)); err != nil {
		return err
	}
	return w.fs.Rename(ctx, oldPath, newPath)
}
