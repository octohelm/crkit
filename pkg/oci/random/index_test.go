package random

import (
	randv2 "math/rand/v2"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/unifs/pkg/units"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func FuzzIndex(f *testing.F) {
	for range 10 {
		f.Add(randv2.IntN(10))
	}

	f.Fuzz(func(t *testing.T, layersN int) {
		t.Run("index", func(t *testing.T) {
			count := 2

			i := MustValue(t, func() (oci.Index, error) {
				return Index(int64(1*units.MiB), layersN, 2)
			})

			indexes := make(map[digest.Digest]ocispecv1.Descriptor)
			images := make(map[digest.Digest]ocispecv1.Descriptor)
			layers := make(map[digest.Digest]ocispecv1.Descriptor)
			configs := make(map[digest.Digest]ocispecv1.Descriptor)

			t.Run("WHEN iterating all child descriptors", func(t *testing.T) {
				for d := range partial.AllChildDescriptors(t.Context(), i) {
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

				Then(t, "got expect layers",
					Expect(len(layers), Equal(int(layersN)*count)),
				)

				Then(t, "got expect configs",
					Expect(len(configs), Equal(count)),
				)

				Then(t, "got expect images",
					Expect(len(images), Equal(count)),
				)

				Then(t, "got expect indexes",
					Expect(len(indexes), Equal(0)),
				)
			})
		})
	})
}
