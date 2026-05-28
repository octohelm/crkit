package fs

import (
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/crkit/pkg/driver"
)

func newWorkspace(d driver.Driver, layout layout.Layout) *workspace {
	return &workspace{
		Driver: d,
		layout: layout,
	}
}

type workspace struct {
	driver.Driver

	layout layout.Layout
}
