package remote

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/containerd/platforms"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/x/logr"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
)

func Manifest(ctx context.Context, repo content.Repository, reference string) (oci.Manifest, error) {
	d := ocispecv1.Descriptor{}

	if dgst, err := content.Reference(reference).Digest(); err != nil {
		tags, err := repo.Tags(ctx)
		if err != nil {
			return nil, err
		}
		found, err := tags.Get(ctx, reference)
		if err != nil {
			return nil, err
		}
		d.MediaType = found.MediaType
		d.Digest = found.Digest
	} else {
		d.Digest = dgst
	}

	return manifest(ctx, repo, d)
}

func manifest(pctx context.Context, repo content.Repository, d ocispecv1.Descriptor) (finalM oci.Manifest, finalErr error) {
	ctx, l := logr.FromContext(pctx).Start(pctx, "resolve manifest",
		slog.String("repo.name", repo.Named().Name()),
		slog.String("manifest.digest", string(d.Digest)),
	)
	defer l.End()
	defer func() {
		if finalErr != nil {
			l.Error(finalErr)
		} else {
			l.Info("resolved")
		}
	}()

	if d.Platform != nil {
		l = l.WithValues(slog.Any("platform", platforms.Format(*d.Platform)))
	}

	if d.Size > 0 {
		l = l.WithValues(slog.Any("manifest.size", d.Size))
	}

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	m, err := manifests.Get(ctx, d.Digest)
	if err != nil {
		return nil, err
	}

	switch m.Type() {
	case ocispecv1.MediaTypeImageIndex, manifestv1.DockerMediaTypeManifestList:
		return pullAsIndex(ctx, repo, d, func(ctx context.Context) (io.ReadCloser, error) {
			return asReadCloser(ctx, m)
		})
	case ocispecv1.MediaTypeImageManifest, manifestv1.DockerMediaTypeManifest:
		return pullAsImage(ctx, repo, d, func(ctx context.Context) (io.ReadCloser, error) {
			return asReadCloser(ctx, m)
		})
	}

	return nil, &content.ErrManifestBlobUnknown{Digest: d.Digest}
}

func asReadCloser(ctx context.Context, x manifestv1.Manifest) (io.ReadCloser, error) {
	p, err := manifestv1.From(x)
	if err != nil {
		return nil, err
	}
	raw, err := p.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewBuffer(raw)), nil
}
