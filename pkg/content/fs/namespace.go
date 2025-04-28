package fs

import (
	"context"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/unifs/pkg/filesystem"
)

func NewNamespace(fs filesystem.FileSystem) content.Namespace {
	return &namespace{workspace: newWorkspace(fs, layout.Default)}
}

type namespace struct {
	workspace *workspace
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (content.Repository, error) {
	return &repository{
		named:     named,
		workspace: n.workspace,
	}, nil
}
