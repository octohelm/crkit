package remote_test

import (
	"fmt"
	randv2 "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/remote/authn"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"
)

func FuzzRemoteNamespace(f *testing.F) {
	images := []remote.Taggable{
		bdd.Must(random.Image(int64(units.BinarySize(int64(randv2.IntN(100)))*units.MiB), randv2.Int64N(5))),
		bdd.Must(random.Index(int64(units.BinarySize(int64(randv2.IntN(100)))*units.MiB), randv2.Int64N(5), 2)),
	}

	for i := range images {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		img := images[idx]

		remoteRegistry := bdd.MustDo(func() (remoteRegistry *httptest.Server, err error) {
			rh := registry.New()

			remoteRegistry = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/auth/token" {
					tok := &authn.Token{}
					tok.TokenType = "Bearer"
					tok.AccessToken = "test"
					tok.ExpiresIn = 1800

					rw.WriteHeader(http.StatusOK)
					_ = json.MarshalWrite(rw, tok)

					return
				}

				auth := req.Header.Get("Authorization")
				if auth == "" {
					rw.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=%q,service=%s", remoteRegistry.URL+"/auth/token", "test"))
					rw.WriteHeader(http.StatusUnauthorized)
					_, _ = rw.Write(nil)
					return
				}

				rh.ServeHTTP(rw, req)
			}))

			return remoteRegistry, nil
		})

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

			d.NoCache = true

			d.Remote.Endpoint = remoteRegistry.URL
			d.Remote.Username = "test"
			d.Remote.Password = "test"
		})

		injector := configuration.ContextInjectorFromContext(ctx)

		rr := bdd.MustDo(func() (*httptest.Server, error) {
			h, err := httprouter.New(apis.R, "registry")
			if err != nil {
				return nil, err
			}

			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if strings.HasSuffix(req.URL.Path, "/") {
					req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
				}

				fmt.Println(req.Method, req.URL.String())

				h.ServeHTTP(w, req.WithContext(injector.InjectContext(ctx)))
			})), nil
		})

		reg := bdd.Must(name.NewRegistry(strings.TrimPrefix(rr.URL, "http://"), name.Insecure))

		t.Run("GIVEN an artifact", bdd.GivenT(func(b bdd.T) {
			ns, _ := content.NamespaceFromContext(ctx)

			repo := reg.Repo("test", "manifest")
			ref := repo.Tag("latest")

			b.When("push this image to container registry", func(b bdd.T) {
				b.Then("success pushed",
					bdd.NoError(remote.Push(ref, img)),
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
