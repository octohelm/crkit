package garbagecollector_test

import (
	"os"
	"testing"
	"time"

	"github.com/distribution/reference"

	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/unifs/pkg/units"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	"github.com/octohelm/crkit/pkg/content/fs/garbagecollector"
	"github.com/octohelm/crkit/pkg/oci"
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

	named := MustValue(t, func() (reference.Named, error) {
		return reference.WithName("test/manifest")
	})

	repository := MustValue(t, func() (content.Repository, error) {
		return ns.Repository(ctx, named)
	})

	tagService := MustValue(t, func() (content.TagService, error) {
		return repository.Tags(ctx)
	})

	manifestService := MustValue(t, func() (content.ManifestService, error) {
		return repository.Manifests(ctx)
	})

	blobsStore := MustValue(t, func() (content.BlobStore, error) {
		return repository.Blobs(ctx)
	})

	t.Run("测试垃圾回收器", func(t *testing.T) {
		idx := MustValue(t, func() (oci.Manifest, error) {
			return random.Index(int64(1*units.MiB), 5, 2)
		})

		const manifestsN = 3 // 2 manifests + 1 index
		const layersN = 12   // (5 layers + 1 config) * 2
		const blobsN = 15    // manifestsN + layersN

		t.Run("推送latest标签", func(t *testing.T) {
			Then(t, "成功推送",
				ExpectDo(
					func() error {
						return remote.Push(ctx, idx, repository, "latest")
					},
				),
			)

			Then(t, "标签修订版本数量为1",
				ExpectMustValue(
					func() (int, error) {
						revisions, err := collect.TagRevisions(ctx, tagService, "latest")
						return len(revisions), err
					},
					Equal(1),
				),
			)

			Then(t, "manifest数量正确",
				ExpectMustValue(
					func() (int, error) {
						manifests, err := collect.Manifests(ctx, manifestService)
						return len(manifests), err
					},
					Equal(manifestsN),
				),
			)

			Then(t, "layer数量正确",
				ExpectMustValue(
					func() (int, error) {
						layers, err := collect.Layers(ctx, blobsStore)
						return len(layers), err
					},
					Equal(layersN),
				),
			)

			Then(t, "blob总数正确",
				ExpectMustValue(
					func() (int, error) {
						blobs, err := collect.Blobs(ctx, ns)
						return len(blobs), err
					},
					Equal(blobsN),
				),
			)

			t.Run("推送另一个镜像", func(t *testing.T) {
				idx2 := MustValue(t, func() (oci.Manifest, error) {
					return random.Index(int64(1*units.MiB), 5, 2)
				})

				t.Run("推送第二个镜像", func(t *testing.T) {
					Then(t, "成功推送第二个镜像",
						ExpectDo(
							func() error {
								return remote.Push(ctx, idx2, repository, "latest")
							},
						),
					)

					Then(t, "标签修订版本数量翻倍",
						ExpectMustValue(
							func() (int, error) {
								revisions, err := collect.TagRevisions(ctx, tagService, "latest")
								return len(revisions), err
							},
							Equal(2),
						),
					)

					Then(t, "manifest数量翻倍",
						ExpectMustValue(
							func() (int, error) {
								manifests, err := collect.Manifests(ctx, manifestService)
								return len(manifests), err
							},
							Equal(manifestsN*2),
						),
					)

					Then(t, "layer数量翻倍",
						ExpectMustValue(
							func() (int, error) {
								layers, err := collect.Layers(ctx, blobsStore)
								return len(layers), err
							},
							Equal(layersN*2),
						),
					)

					Then(t, "blob总数翻倍",
						ExpectMustValue(
							func() (int, error) {
								blobs, err := collect.Blobs(ctx, ns)
								return len(blobs), err
							},
							Equal(blobsN*2),
						),
					)

					t.Run("排除1小时内修改的垃圾回收", func(t *testing.T) {
						Then(t, "成功执行垃圾回收",
							ExpectDo(
								func() error {
									return d.GarbageCollector.MarkAndSweepExcludeModifiedIn(ctx, time.Hour)
								},
							),
						)

						Then(t, "保留所有资源（1小时内修改）",
							ExpectMustValue(
								func() (int, error) {
									revisions, err := collect.TagRevisions(ctx, tagService, "latest")
									return len(revisions), err
								},
								Equal(2),
							),
							ExpectMustValue(
								func() (int, error) {
									manifests, err := collect.Manifests(ctx, manifestService)
									return len(manifests), err
								},
								Equal(manifestsN*2),
							),
							ExpectMustValue(
								func() (int, error) {
									layers, err := collect.Layers(ctx, blobsStore)
									return len(layers), err
								},
								Equal(layersN*2),
							),
							ExpectMustValue(
								func() (int, error) {
									blobs, err := collect.Blobs(ctx, ns)
									return len(blobs), err
								},
								Equal(blobsN*2),
							),
						)
					})

					t.Run("清除所有垃圾回收", func(t *testing.T) {
						Then(t, "成功执行全面垃圾回收",
							ExpectDo(
								func() error {
									return d.GarbageCollector.MarkAndSweepExcludeModifiedIn(ctx, 0)
								},
							),
						)

						Then(t, "只保留最新资源",
							ExpectMustValue(
								func() (int, error) {
									revisions, err := collect.TagRevisions(ctx, tagService, "latest")
									return len(revisions), err
								},
								Equal(1),
							),
							ExpectMustValue(
								func() (int, error) {
									manifests, err := collect.Manifests(ctx, manifestService)
									return len(manifests), err
								},
								Equal(manifestsN),
							),
							ExpectMustValue(
								func() (int, error) {
									layers, err := collect.Layers(ctx, blobsStore)
									return len(layers), err
								},
								Equal(layersN),
							),
							ExpectMustValue(
								func() (int, error) {
									blobs, err := collect.Blobs(ctx, ns)
									return len(blobs), err
								},
								Equal(blobsN),
							),
						)
					})
				})
			})
		})
	})
}
