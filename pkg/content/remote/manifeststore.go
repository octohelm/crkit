package remote

import (
	"context"
	"net/http"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/pkg/errors"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
)

type manifestService struct {
	*repository
}

var _ content.ManifestService = &manifestService{}

func (b *manifestService) normalizeError(dgst digest.Digest, err error) error {
	terr := &transport.Error{}
	if errors.As(err, &terr) {
		if terr.StatusCode == http.StatusNotFound {
			return &content.ErrManifestBlobUnknown{
				Digest: dgst,
			}
		}
	}
	return err
}

func (ms *manifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	err := ms.pusher.Delete(ctx, ms.repo.Digest(dgst.String()))
	return ms.normalizeError(dgst, err)
}

func (ms *manifestService) Put(ctx context.Context, m manifestv1.Manifest) (digest.Digest, error) {
	p, err := manifestv1.From(m)
	if err != nil {
		return "", err
	}

	raw, dgst, err := p.Payload()
	if err != nil {
		return "", err
	}

	var ref name.Reference = ms.repo.Digest(dgst.String())

	if err := ms.pusher.Push(ctx, ref, &manifest{mediaType: m.Type(), raw: raw}); err != nil {
		return "", ms.normalizeError(dgst, err)
	}

	return dgst, nil
}

func (ms *manifestService) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	d, err := ms.puller.Get(ctx, ms.repo.Digest(dgst.String()))
	if err != nil {
		return nil, ms.normalizeError(dgst, err)
	}
	return &manifestv1.Descriptor{
		MediaType:   string(d.MediaType),
		Digest:      digest.NewDigestFromHex(d.Digest.Algorithm, d.Digest.Hex),
		Size:        d.Size,
		Annotations: d.Annotations,
	}, nil
}

func (ms *manifestService) Get(ctx context.Context, dgst digest.Digest) (manifestv1.Manifest, error) {
	d, err := ms.puller.Get(ctx, ms.repo.Digest(dgst.String()))
	if err != nil {
		return nil, ms.normalizeError(dgst, err)
	}

	payload := &manifestv1.Payload{}
	if err := payload.UnmarshalJSON(d.Manifest); err != nil {
		return nil, &content.ErrManifestUnverified{}
	}
	return payload, nil
}

type manifest struct {
	mediaType string
	raw       []byte
}

func (m *manifest) MediaType() (types.MediaType, error) {
	return types.MediaType(m.mediaType), nil
}

func (m *manifest) RawManifest() ([]byte, error) {
	return m.raw, nil
}
