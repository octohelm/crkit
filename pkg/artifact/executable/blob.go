package executable

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/platforms"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func Platformed(platform string, open func(ctx context.Context) (io.ReadCloser, error)) (oci.Blob, error) {
	p, err := platforms.Parse(platform)
	if err != nil {
		return nil, fmt.Errorf("invalid platform %q: %w", platform, err)
	}

	return partial.CompressedBlobFromOpener(ocispecv1.Descriptor{
		MediaType: MediaTypeBinaryContent,
		Platform:  &p,
	}, open), nil
}
