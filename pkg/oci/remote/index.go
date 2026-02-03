package remote

import (
	"context"
	"fmt"
	"io"
	"iter"
	"sync"

	syncx "github.com/octohelm/x/sync"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
)

func pullAsIndex(ctx context.Context, repo content.Repository, desc ocispecv1.Descriptor, open func(ctx context.Context) (io.ReadCloser, error)) (oci.Index, error) {
	idx := &index{
		repo: repo,
	}

	r, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := idx.InitFromReader(r, desc); err != nil {
		return nil, fmt.Errorf("init index %s failed: %w", desc.Digest, err)
	}

	return idx, nil
}

type index struct {
	internal.Index
	repo content.Repository

	cached syncx.Map[digest.Digest, func() (oci.Manifest, error)]
}

func (i *index) Manifests(ctx context.Context) iter.Seq2[oci.Manifest, error] {
	return func(yield func(oci.Manifest, error) bool) {
		idx, err := i.Value(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, md := range idx.Manifests {
			call, _ := i.cached.LoadOrStore(md.Digest, sync.OnceValues(func() (oci.Manifest, error) {
				return manifest(ctx, i.repo, md)
			}))

			if !yield(call()) {
				return
			}
		}
	}
}
