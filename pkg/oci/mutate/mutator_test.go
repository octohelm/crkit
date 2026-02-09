package mutate

import (
	"context"
	randv2 "math/rand/v2"
	"slices"
	"testing"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func FuzzMutator(f *testing.F) {
	options := []ImageMutatorOption{
		func(m *Mutator[oci.Image]) {
			m.Add(func(ctx context.Context, base oci.Image) (oci.Image, error) {
				return WithArtifactType(base, "application/vnd.content+type")
			})
		},
		func(m *Mutator[oci.Image]) {
			m.Add(func(ctx context.Context, base oci.Image) (oci.Image, error) {
				return WithPlatform(base, "linux/amd64")
			})
		},
		func(m *Mutator[oci.Image]) {
			m.Add(func(ctx context.Context, base oci.Image) (oci.Image, error) {
				return WithAnnotations(base, map[string]string{
					ocispecv1.AnnotationBaseImageName: "x/image",
					ocispecv1.AnnotationRefName:       "v1.0.0",
				})
			})
		},
		func(m *Mutator[oci.Image]) {
			m.Add(func(ctx context.Context, base oci.Image) (oci.Image, error) {
				return AppendLayers(
					base,
					partial.BlobFromBytes([]byte("123"), ocispecv1.Descriptor{MediaType: "text/plain"}),
				)
			})
		},
	}

	m := &ImageMutator{}
	m.Build(options...)

	baseImg := MustValue(f, func() (oci.Image, error) {
		return m.Apply(f.Context(), empty.Image)
	})

	baseRaw := MustValue(f, func() (string, error) {
		raw, err := baseImg.Raw(f.Context())
		return string(raw), err
	})

	for range 10 {
		f.Add(1)
	}

	f.Fuzz(func(t *testing.T, i int) {
		t.Run("shuffle options", func(t *testing.T) {
			m := &ImageMutator{}
			m.Build(ShuffledSlice(options)...)

			img := MustValue(t, func() (oci.Image, error) {
				return m.Apply(t.Context(), empty.Image)
			})

			imgRaw := MustValue(t, func() (string, error) {
				raw, err := img.Raw(t.Context())
				return string(raw), err
			})

			Then(t, "should produce same result regardless of option order",
				Expect(imgRaw, Equal(baseRaw)),
			)
		})
	})
}

func ShuffledSlice[T any](s []T) []T {
	s = slices.Clone(s)

	randv2.Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})

	return s
}
