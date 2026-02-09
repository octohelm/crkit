package mutate_test

import (
	"context"
	"testing"

	"github.com/go-json-experiment/json/jsontext"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func TestMutate(t *testing.T) {
	t.Run("empty image", func(t *testing.T) {
		img := empty.Image

		Then(t, "could be manifest",
			ExpectMustValue(
				func() (Snapshot, error) {
					raw := MustValue(t, func() ([]byte, error) {
						return formated(t.Context(), img)
					})

					return SnapshotOf(
						SnapshotFileFromRaw("manifest.json", raw),
					), nil
				},
				MatchSnapshot("image-empty"),
			),
		)
	})

	t.Run("normal image", func(t *testing.T) {
		img := MustValue(t, func() (oci.Image, error) {
			return mutate.With(
				empty.Image,
				func(base oci.Image) (oci.Image, error) {
					return mutate.WithImageConfig(
						base,
						&ocispecv1.ImageConfig{
							Env: []string{"X=1"},
						},
					)
				},
				func(base oci.Image) (oci.Image, error) {
					return mutate.AppendLayers(
						base,
						partial.BlobFromBytes([]byte("123"), ocispecv1.Descriptor{MediaType: "text/plain"}),
					)
				},
			)
		})

		Then(t, "could be manifest",
			ExpectMustValue(
				func() (Snapshot, error) {
					raw := MustValue(t, func() ([]byte, error) {
						return formated(t.Context(), img)
					})

					return SnapshotOf(
						SnapshotFileFromRaw("manifest.json", raw),
					), nil
				},
				MatchSnapshot("image-normal"),
			),
		)
	})

	t.Run("artifact", func(t *testing.T) {
		img := MustValue(t, func() (oci.Image, error) {
			return mutate.With(
				empty.Image,
				func(base oci.Image) (oci.Image, error) {
					return mutate.WithArtifactType(base, "application/vnd.content+type")
				},
				func(base oci.Image) (oci.Image, error) {
					return mutate.AppendLayers(
						base,
						partial.BlobFromBytes([]byte("123"), ocispecv1.Descriptor{MediaType: "text/plain"}),
					)
				},
			)
		})

		Then(t, "could be manifest",
			ExpectMustValue(
				func() (Snapshot, error) {
					raw, err := formated(t.Context(), img)
					if err != nil {
						return nil, err
					}

					return SnapshotOf(
						SnapshotFileFromRaw("manifest.json", raw),
					), nil
				},
				MatchSnapshot("artifact"),
			),
		)
	})

	t.Run("empty index", func(t *testing.T) {
		idx := empty.Index

		Then(t, "could be manifest",
			ExpectMustValue(
				func() (Snapshot, error) {
					raw, err := formated(t.Context(), idx)
					if err != nil {
						return nil, err
					}

					return SnapshotOf(
						SnapshotFileFromRaw("manifest.json", raw),
					), nil
				},
				MatchSnapshot("index-empty"),
			),
		)
	})

	t.Run("artifact index", func(t *testing.T) {
		idx := MustValue(t, func() (oci.Index, error) {
			return mutate.With(
				empty.Index,
				func(idx oci.Index) (oci.Index, error) {
					return mutate.WithArtifactType(idx, "application/vnd.content+index")
				},
				func(idx oci.Index) (oci.Index, error) {
					img := MustValue(t, func() (oci.Image, error) {
						return mutate.WithPlatform(empty.Image, "linux/amd64")
					})

					return mutate.AppendManifests(idx, img)
				},
				func(idx oci.Index) (oci.Index, error) {
					return mutate.WithAnnotations(idx, map[string]string{
						ocispecv1.AnnotationBaseImageName: "x/image",
						ocispecv1.AnnotationRefName:       "v1.0.0",
					})
				},
			)
		})

		Then(t, "could be manifest",
			ExpectMustValue(
				func() (Snapshot, error) {
					raw, err := formated(t.Context(), idx)
					if err != nil {
						return nil, err
					}

					return SnapshotOf(
						SnapshotFileFromRaw("manifest.json", raw),
					), nil
				},
				MatchSnapshot("index-artifact"),
			),
		)
	})
}

func formated(ctx context.Context, m oci.Manifest) ([]byte, error) {
	raw, err := m.Raw(ctx)
	if err != nil {
		return nil, err
	}
	return jsontext.AppendFormat(nil, raw, jsontext.WithIndent("  "))
}
