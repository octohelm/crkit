package fs

import (
	"context"
	"path/filepath"

	"github.com/octohelm/unifs/pkg/filesystem"
)

func writeFile(ctx context.Context, fs filesystem.FileSystem, name string, data []byte) error {
	dir := filepath.Dir(name)
	if dir != "" {
		if err := filesystem.MkdirAll(ctx, fs, dir); err != nil {
			return err
		}
	}
	return filesystem.Write(ctx, fs, name, data)
}
