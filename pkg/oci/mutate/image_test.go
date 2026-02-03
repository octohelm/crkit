package mutate_test

import (
	"context"
	"testing"

	"github.com/go-json-experiment/json/jsontext"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/x/testing/bdd"
	"github.com/octohelm/x/testing/snapshot"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func TestMutate(t *testing.T) {
	b := bdd.FromT(t)

	b.Given("empty image", func(b bdd.T) {
		img := empty.Image

		b.Then("could be manifest",
			bdd.EqualDoValue(
				bdd.Snapshot("image-empty"),
				func() (*snapshot.Snapshot, error) {
					raw, err := formated(b.Context(), img)
					if err != nil {
					}

					return snapshot.Files(
						snapshot.FileFromRaw("manifest.json", raw),
					), nil
				},
			),
		)
	})

	b.Given("normal image", func(b bdd.T) {
		img := bdd.DoValue(b, func() (oci.Image, error) {
			return mutate.With(
				empty.Image,
				func(base oci.Image) (oci.Image, error) {
					return mutate.WithImageConfig(
						base,
						&ocispecv1.ImageConfig{
							Env: []string{
								"X=1",
							},
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

		b.Then("could be manifest",
			bdd.EqualDoValue(
				bdd.Snapshot("image-normal"),
				func() (*snapshot.Snapshot, error) {
					raw, err := formated(b.Context(), img)
					if err != nil {
					}

					return snapshot.Files(
						snapshot.FileFromRaw("manifest.json", raw),
					), nil
				},
			),
		)
	})

	b.Given("artifact", func(b bdd.T) {
		img := bdd.DoValue(b, func() (oci.Image, error) {
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

		b.Then("could be manifest",
			bdd.EqualDoValue(
				bdd.Snapshot("artifact"),
				func() (*snapshot.Snapshot, error) {
					raw, err := formated(b.Context(), img)
					if err != nil {
					}

					return snapshot.Files(
						snapshot.FileFromRaw("manifest.json", raw),
					), nil
				},
			))
	})

	b.Given("empty index", func(b bdd.T) {
		idx := empty.Index

		b.Then("could be manifest",
			bdd.EqualDoValue(
				bdd.Snapshot("index-empty"),
				func() (*snapshot.Snapshot, error) {
					raw, err := formated(b.Context(), idx)
					if err != nil {
					}

					return snapshot.Files(
						snapshot.FileFromRaw("manifest.json", raw),
					), nil
				},
			))
	})

	b.Given("artifact index", func(b bdd.T) {
		idx := bdd.DoValue(b, func() (oci.Index, error) {
			return mutate.With(
				empty.Index,
				func(idx oci.Index) (oci.Index, error) {
					return mutate.WithArtifactType(idx, "application/vnd.content+index")
				},
				func(idx oci.Index) (oci.Index, error) {
					img, err := mutate.WithPlatform(empty.Image, "linux/amd64")
					if err != nil {
						return nil, err
					}

					return mutate.AppendManifests(
						idx,
						img,
					)
				},
				func(idx oci.Index) (oci.Index, error) {
					return mutate.WithAnnotations(idx, map[string]string{
						ocispecv1.AnnotationBaseImageName: "x/image",
						ocispecv1.AnnotationRefName:       "v1.0.0",
					})
				},
			)
		})

		b.Then("could be manifest",
			bdd.EqualDoValue(
				bdd.Snapshot("index-artifact"),
				func() (*snapshot.Snapshot, error) {
					raw, err := formated(b.Context(), idx)
					if err != nil {
					}

					return snapshot.Files(
						snapshot.FileFromRaw("manifest.json", raw),
					), nil
				},
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
