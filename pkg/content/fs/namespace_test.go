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
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/oci/remote"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/random"
)

func FuzzNamespace(f *testing.F) {
	for i := range 2 {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, i int) {
		b := bdd.FromT(t)

		b.Given("local fs namespace", func(b bdd.T) {
			manifestN, image := bdd.DoValues(b, func() (int, oci.Manifest, error) {
				switch i {
				case 1:
					img, err := random.Image(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5)
					return 1, img, err
				}
				idx, err := random.Index(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5, 2)
				return 3, idx, err
			})

			ctx, _ := testingutil.BuildContext(t, func(d *struct {
				otel.Otel
				contentapi.NamespaceProvider
			},
			) {
				tmp := b.TempDir()
				t.Cleanup(func() {
					_ = os.RemoveAll(tmp)
				})

				d.Content.Backend.Scheme = "file"
				d.Content.Backend.Path = tmp
			})

			s := bdd.DoValue(t, func() (*httptest.Server, error) {
				injector := configuration.ContextInjectorFromContext(ctx)

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

			reg := bdd.Must(contentremote.New(ctx, contentremote.Registry{
				Endpoint: s.URL,
			}))

			remoteRepo := bdd.DoValue(t, func() (content.Repository, error) {
				named, err := reference.WithName("test/manifest")
				if err != nil {
					return nil, err
				}
				return reg.Repository(ctx, named)
			})

			b.Given("an artifact", func(b bdd.T) {
				ns, _ := content.NamespaceFromContext(ctx)

				b.When("push this image to container registry", func(b bdd.T) {
					b.Then("success pushed",
						bdd.NoError(
							remote.Push(ctx, image, remoteRepo, "latest"),
						),
					)

					b.Then("got catalogs",
						bdd.EqualDoValue(
							[]string{remoteRepo.Named().Name()},
							func() ([]string, error) {
								return collect.Catalogs(ctx, ns)
							}),
					)

					b.Then("got digests same as pushed",
						bdd.EqualDoValue(manifestN, func() (int, error) {
							repo, err := ns.Repository(ctx, remoteRepo.Named())
							if err != nil {
								return -1, err
							}

							manifests, err := repo.Manifests(ctx)
							if err != nil {
								return -1, err
							}

							revisions, err := collect.Manifests(ctx, manifests)
							if err != nil {
								return -1, err
							}
							return len(revisions), nil
						}),
					)

					b.When("pull and push as v1", func(b bdd.T) {
						imagePushed := bdd.DoValue(b, func() (oci.Manifest, error) {
							return remote.Manifest(ctx, remoteRepo, "latest")
						})

						b.Then("success",
							bdd.NoError(
								remote.Push(ctx, imagePushed, remoteRepo, "v1"),
							),
						)

						tags := bdd.DoValue(b, func() (content.TagService, error) {
							return remoteRepo.Tags(ctx)
						})

						b.Then("could got two tags",
							bdd.EqualDoValue(
								[]string{
									"latest",
									"v1",
								},
								func() ([]string, error) {
									return tags.All(ctx)
								},
							),
						)

						b.When("remove tag", func(b bdd.T) {
							b.Then("success",
								bdd.NoError(
									tags.Untag(ctx, "latest"),
								),
							)

							b.Then("could got two tags",
								bdd.EqualDoValue(
									[]string{"v1"},
									func() ([]string, error) {
										return tags.All(ctx)
									},
								),
							)
						})
					})
				})
			})
		})
	})
}
