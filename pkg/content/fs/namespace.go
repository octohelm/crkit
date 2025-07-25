package fs

import (
	"context"
	"fmt"
	"io/fs"
	"iter"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"

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

var _ content.DigestIterable = (*namespace)(nil)

func (n *namespace) Digests(ctx context.Context) iter.Seq2[digest.Digest, error] {
	return (&blobStore{workspace: n.workspace}).Digests(ctx)
}

var _ content.RepositoryNameIterable = (*namespace)(nil)

func (n *namespace) RepositoryNames(ctx context.Context) iter.Seq2[reference.Named, error] {
	return func(yield func(reference.Named, error) bool) {
		yieldNamed := func(named reference.Named, err error) bool {
			return yield(named, err)
		}

		err := n.workspace.WalkDir(ctx, n.workspace.layout.RepositorysPath(), func(pathname string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if pathname == "." {
				return nil
			}

			if d.IsDir() {
				if base := path.Base(pathname); base == "_manifests" {
					name := strings.TrimSuffix(pathname, "/_manifests")

					named, err := reference.WithName(name)
					if err != nil {
						return fmt.Errorf("failed to parse repository name %q: %w", name, err)
					}

					if !yieldNamed(named, nil) {
						return fs.SkipAll
					}

					return fs.SkipDir
				} else if strings.HasPrefix(base, "_") {
					return fs.SkipDir
				}
			}

			return nil
		})
		if err != nil {
			if !yield(nil, err) {
				return
			}
		}
	}
}
