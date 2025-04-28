package fs

import (
	"context"
	"io"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

var _ content.ManifestService = &manifestService{}

type manifestService struct {
	blobStore *linkedBlobStore
}

func (m *manifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	return m.blobStore.Remove(ctx, dgst)
}

func (m *manifestService) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	return m.blobStore.Info(ctx, dgst)
}

func (m *manifestService) Get(ctx context.Context, dgst digest.Digest) (manifestv1.Manifest, error) {
	info, err := m.Info(ctx, dgst)
	if err != nil {
		return nil, err
	}

	blob, err := m.blobStore.Open(ctx, info.Digest)
	if err != nil {
		return nil, err
	}
	defer blob.Close()

	raw, err := io.ReadAll(blob)
	if err != nil {
		return nil, err
	}

	payload := &manifestv1.Payload{}
	if err := payload.UnmarshalJSON(raw); err != nil {
		return nil, err
	}
	return payload, nil
}

func (m *manifestService) Put(ctx context.Context, manifest manifestv1.Manifest) (digest.Digest, error) {
	payload, err := manifestv1.From(manifest)
	if err != nil {
		return "", nil
	}

	raw, dgst, err := payload.Payload()
	if err != nil {
		return "", nil
	}

	w, err := m.blobStore.Writer(ctx)
	if err != nil {
		return "", err
	}
	defer w.Close()

	if _, err := w.Write(raw); err != nil {
		return "", err
	}

	d, err := w.Commit(ctx, manifestv1.Descriptor{
		Digest: dgst,
	})
	if err != nil {
		return "", err
	}

	return d.Digest, nil
}
