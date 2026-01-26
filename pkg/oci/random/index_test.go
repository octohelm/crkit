package random

import (
	randv2 "math/rand/v2"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/oci/partial"
)

func FuzzIndex(f *testing.F) {
	for range 10 {
		f.Add(randv2.IntN(10))
	}

	f.Fuzz(func(t *testing.T, layersN int) {
		b := bdd.FromT(t)

		b.Given("index", func(b bdd.T) {
			count := 2

			i := bdd.Must(Index(int64(1*units.MiB), layersN, 2))

			indexes := make(map[digest.Digest]ocispecv1.Descriptor)
			images := make(map[digest.Digest]ocispecv1.Descriptor)
			layers := make(map[digest.Digest]ocispecv1.Descriptor)
			configs := make(map[digest.Digest]ocispecv1.Descriptor)

			for d := range partial.AllChildDescriptors(b.Context(), i) {
				switch d.MediaType {
				case ocispecv1.MediaTypeImageIndex:
					indexes[d.Digest] = d
				case ocispecv1.MediaTypeImageManifest:
					images[d.Digest] = d
				case ocispecv1.MediaTypeImageConfig:
					configs[d.Digest] = d
				default:
					layers[d.Digest] = d
				}
			}

			b.Then("got expect layers", bdd.Equal(len(layers), int(layersN)*count))
			b.Then("got expect configs", bdd.Equal(len(configs), count))
			b.Then("got expect images", bdd.Equal(len(images), count))
			b.Then("got expect indexes", bdd.Equal(len(indexes), 0))
		})
	})
}
