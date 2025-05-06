package fs_test

import (
	"fmt"

	randv2 "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"
)

func FuzzNamespace(f *testing.F) {
	images := []remote.Taggable{
		bdd.Must(random.Image(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), 5)),
		bdd.Must(random.Index(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), 5, 2)),
	}

	for i := range images {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		img := images[idx]

		layerN := 1
		if idx > 0 {
			layerN = 3
		}

		ctx, _ := testingutil.BuildContext(t, func(d *struct {
			otel.Otel

			contentapi.NamespaceProvider
		},
		) {
			tmp := t.TempDir()

			t.Cleanup(func() {
				_ = os.RemoveAll(tmp)
			})

			d.Content.Backend = *bdd.Must(strfmt.ParseEndpoint("file://" + tmp))
		})

		injector := configuration.ContextInjectorFromContext(ctx)

		s := bdd.MustDo(func() (*httptest.Server, error) {
			h, err := httprouter.New(apis.R, "registry")
			if err != nil {
				return nil, err
			}

			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if strings.HasSuffix(req.URL.Path, "/") {
					req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
				}
				fmt.Println(req.Method, req.URL.String())
				h.ServeHTTP(w, req.WithContext(injector.InjectContext(req.Context())))
			})), nil
		})
		t.Cleanup(s.Close)

		reg := bdd.Must(name.NewRegistry(strings.TrimPrefix(s.URL, "http://"), name.Insecure))

		t.Run("GIVEN an artifact", bdd.GivenT(func(b bdd.T) {
			ns, _ := content.NamespaceFromContext(ctx)

			repo := reg.Repo("test", "manifest")
			ref := repo.Tag("latest")

			named, _ := reference.WithName("test/manifest")

			b.When("push this image to container registry", func(b bdd.T) {
				b.Then("success pushed",
					bdd.NoError(remote.Push(ref, img)),
				)

				catalogs := bdd.Must(collect.Catalogs(ctx, ns))
				b.Then("got catalogs",
					bdd.Equal([]string{named.Name()}, catalogs),
				)

				manifests := bdd.Must(bdd.Must(ns.Repository(ctx, named)).Manifests(ctx))

				revisions := bdd.Must(collect.Manifests(ctx, manifests))
				b.Then("got digests same as pushed",
					bdd.Equal(layerN, len(revisions)),
				)

				b.When("pull and push as v1", func(b bdd.T) {
					img1 := bdd.Must(remote.Image(ref))

					err := remote.Push(ref.Tag("v1"), img1)
					b.Then("success",
						bdd.NoError(err),
					)

					repository := bdd.Must(ns.Repository(ctx, content.Name("test/manifest")))
					tags := bdd.Must(repository.Tags(ctx))

					b.Then("could got two tags",
						bdd.Equal(
							[]string{
								"latest", "v1",
							},
							bdd.Must(tags.All(ctx)),
						),
					)

					b.When("remove tag", func(b bdd.T) {
						b.Then("success",
							bdd.NoError(tags.Untag(ctx, "latest")),
						)

						b.Then("could got two tags",
							bdd.Equal(
								[]string{
									"v1",
								},
								bdd.Must(tags.All(ctx)),
							),
						)
					})
				})
			})
		}))
	})
}
