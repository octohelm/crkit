package random

import (
	"fmt"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func Image(byteSize int64, layerCount int) (img oci.Image, err error) {
	img = empty.Image

	for range layerCount {
		img, err = mutate.AppendLayers(img, partial.BlobFromBytes(randomBytes(byteSize), ocispecv1.Descriptor{MediaType: ocispecv1.MediaTypeImageLayer}))
		if err != nil {
			return
		}
	}

	img, err = mutate.WithImageConfig(img, &ocispecv1.ImageConfig{
		Env: []string{
			fmt.Sprintf("SEED=%x", randomBytes(256)),
		},
	})

	return
}
