package fs

import (
	"context"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"

	"github.com/distribution/reference"
)

func NewNamespace(fs filesystem.FileSystem) content.Namespace {
	return &namespace{
		fs: fs,
	}
}

type namespace struct {
	fs filesystem.FileSystem
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (content.Repository, error) {
	return &repository{
		named: named,
		fs:    n.fs,
	}, nil
}
