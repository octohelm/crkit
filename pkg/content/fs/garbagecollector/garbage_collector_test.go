package garbagecollector_test

import (
	"os"
	"testing"
	"time"

	"github.com/distribution/reference"

	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	"github.com/octohelm/crkit/pkg/content/fs/garbagecollector"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
)

func TestGarbageCollector(t *testing.T) {
	_ = os.RemoveAll(".tmp")
	_ = os.Mkdir(".tmp", os.ModePerm)

	ctx, d := testingutil.BuildContext(t, func(d *struct {
		otel.Otel

		contentapi.NamespaceProvider

		garbagecollector.GarbageCollector
	},
	) {
		d.LogLevel = "debug"
		d.LogFormat = "text"

		d.Content.Backend.Scheme = "file"
		d.Content.Backend.Hostname = "."
		d.Content.Backend.Path = ".tmp"
	})

	ns, _ := content.NamespaceFromContext(ctx)

	repository := bdd.Must(ns.Repository(ctx, bdd.Must(reference.WithName("test/manifest"))))

	tagService := bdd.Must(repository.Tags(ctx))
	manifestService := bdd.Must(repository.Manifests(ctx))
	blobsStore := bdd.Must(repository.Blobs(ctx))

	b := bdd.FromT(t)

	b.Given("an index artifact", func(b bdd.T) {
		idx := bdd.Must(random.Index(int64(1*units.MiB), 5, 2))
		manifestsN := 2 /* manifests */ + 1 /* index */
		layersN := (5 /* layers */ + 1 /* config */) * 2
		blobsN := manifestsN + layersN

		b.When("push with tag latest", func(b bdd.T) {
			b.Then("success pushed",
				bdd.NoError(remote.Push(ctx, idx, repository, "latest")),
			)

			b.Then("tag revisions and manifests/layers and blobs got single size",
				bdd.Equal(1, len(bdd.Must(collect.TagRevisions(ctx, tagService, "latest")))),
				bdd.Equal(manifestsN, len(bdd.Must(collect.Manifests(ctx, manifestService)))),
				bdd.Equal(layersN, len(bdd.Must(collect.Layers(ctx, blobsStore)))),
				bdd.Equal(blobsN, len(bdd.Must(collect.Blobs(ctx, ns)))),
			)

			b.Given("another index artifact", func(b bdd.T) {
				idx2 := bdd.Must(random.Index(int64(1*units.MiB), 5, 2))

				b.When("push another image to container registry", func(b bdd.T) {
					b.Then("success pushed",
						bdd.NoError(remote.Push(ctx, idx2, repository, "latest")),
					)

					b.Then("tag revisions and manifests/layers and blobs got double size",
						bdd.Equal(2, len(bdd.Must(collect.TagRevisions(ctx, tagService, "latest")))),
						bdd.Equal(manifestsN*2, len(bdd.Must(collect.Manifests(ctx, manifestService)))),
						bdd.Equal(layersN*2, len(bdd.Must(collect.Layers(ctx, blobsStore)))),
						bdd.Equal(blobsN*2, len(bdd.Must(collect.Blobs(ctx, ns)))),
					)

					b.When("do mark and sweep exclude modified in 1 hour", func(b bdd.T) {
						b.Then("success",
							bdd.NoError(d.GarbageCollector.MarkAndSweepExcludeModifiedIn(ctx, time.Hour)),
						)

						b.Then("tag revisions and manifests/layers and blobs still got double size",
							bdd.Equal(2, len(bdd.Must(collect.TagRevisions(ctx, tagService, "latest")))),
							bdd.Equal(manifestsN*2, len(bdd.Must(collect.Manifests(ctx, manifestService)))),
							bdd.Equal(layersN*2, len(bdd.Must(collect.Layers(ctx, blobsStore)))),
							bdd.Equal(blobsN*2, len(bdd.Must(collect.Blobs(ctx, ns)))),
						)
					})

					b.When("do mark and sweep all", func(b bdd.T) {
						b.Then("success",
							bdd.NoError(d.GarbageCollector.MarkAndSweepExcludeModifiedIn(ctx, 0)),
						)

						b.Then("tag revisions and manifests/layers and blobs got single size",
							bdd.Equal(1, len(bdd.Must(collect.TagRevisions(ctx, tagService, "latest")))),
							bdd.Equal(manifestsN, len(bdd.Must(collect.Manifests(ctx, manifestService)))),
							bdd.Equal(layersN, len(bdd.Must(collect.Layers(ctx, blobsStore)))),
							bdd.Equal(blobsN, len(bdd.Must(collect.Blobs(ctx, ns)))),
						)
					})
				})
			})
		})
	})
}
