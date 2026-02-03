package internal

import (
	"context"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/oci"
)

func ToManifest(ctx context.Context, m oci.Manifest) (manifestv1.Manifest, error) {
	p := &manifestv1.Payload{}

	desc, err := m.Descriptor(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := m.Raw(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.InitFromRaw(raw, desc); err != nil {
		return nil, err
	}
	return p, nil
}
