package remote

import (
	"context"
	"fmt"
	"io"
	"iter"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
)

func pullAsIndex(ctx context.Context, repo content.Repository, desc ocispecv1.Descriptor, open func(ctx context.Context) (io.ReadCloser, error)) (oci.Index, error) {
	idx := &index{
		repo: repo,
	}

	raw, err := internal.ReadAllFromOpener(ctx, open)
	if err != nil {
		return nil, err
	}

	if err := idx.InitFromRaw(raw, desc); err != nil {
		return nil, fmt.Errorf("init index %s failed: %w", desc.Digest, err)
	}

	return idx, nil
}

type index struct {
	internal.Index
	repo content.Repository
}

func (i *index) Manifests(ctx context.Context) iter.Seq2[oci.Manifest, error] {
	return func(yield func(oci.Manifest, error) bool) {
		idx, err := i.Value(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, md := range idx.Manifests {
			if !yield(manifest(ctx, i.repo, md)) {
				return
			}
		}
	}
}
