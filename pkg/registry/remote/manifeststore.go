package remote

import (
	"context"

	"github.com/distribution/distribution/v3"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"

	_ "github.com/distribution/distribution/v3/manifest/manifestlist"
	_ "github.com/distribution/distribution/v3/manifest/ocischema"
	_ "github.com/distribution/distribution/v3/manifest/schema2"
)

type manifestService struct {
	*repository
}

var _ distribution.ManifestService = &manifestService{}

func (ms *manifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	return ms.pusher.Delete(ctx, ms.repo.Digest(dgst.String()))
}

func (ms *manifestService) Put(ctx context.Context, m distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	mediaType, raw, err := m.Payload()
	if err != nil {
		return "", err
	}

	dgst := digest.FromBytes(raw)

	var ref name.Reference = ms.repo.Digest(dgst.String())

	for _, o := range options {
		switch x := o.(type) {
		case distribution.WithTagOption:
			ref = ms.repo.Tag(x.Tag)
		}
	}

	if err := ms.pusher.Push(ctx, ref, &manifest{mediaType, raw}); err != nil {
		return "", err
	}

	return dgst, nil
}

func (ms *manifestService) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	_, err := ms.puller.Head(ctx, ms.repo.Digest(dgst.String()))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (ms *manifestService) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	d, err := ms.puller.Get(ctx, ms.repo.Digest(dgst.String()))
	if err != nil {
		return nil, err
	}
	m, _, err := distribution.UnmarshalManifest(string(d.MediaType), d.Manifest)
	if err != nil {
		return nil, err
	}
	return m, nil
}

type manifest struct {
	mediaType string
	raw       []byte
}

func (m *manifest) RawManifest() ([]byte, error) {
	return m.raw, nil
}
