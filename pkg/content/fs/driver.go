package fs

import (
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/unifs/pkg/filesystem"
)

func newWorkspace(fs filesystem.FileSystem, layout layout.Layout) *workspace {
	return &workspace{
		Driver: driver.FromFileSystem(fs),
		layout: layout,
	}
}

type workspace struct {
	driver.Driver

	layout layout.Layout
}
