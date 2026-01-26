package remote_test

import (
	"fmt"
	randv2 "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/distribution/reference"
	"github.com/go-json-experiment/json"

	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/content/remote/authn"
	contenttestutil "github.com/octohelm/crkit/pkg/content/testutil"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

func FuzzRemoteNamespace(f *testing.F) {
	manifests := []oci.Manifest{
		bdd.Must(random.Image(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5)),
		bdd.Must(random.Index(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5, 2)),
	}

	for i := range manifests {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		src := manifests[idx]

		remoteRegistry := bdd.MustDo(func() (remoteRegistry *httptest.Server, err error) {
			rh := contenttestutil.NewRegistry(t)

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

		registryServer := bdd.MustDo(func() (*httptest.Server, error) {
			h, err := httprouter.New(apis.R, "registry")
			if err != nil {
				return nil, err
			}

			return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if strings.HasSuffix(req.URL.Path, "/") {
					req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
				}
				h.ServeHTTP(w, req.WithContext(injector.InjectContext(ctx)))
			})), nil
		})
		t.Cleanup(registryServer.Close)

		reg := bdd.Must(contentremote.New(ctx, contentremote.Registry{
			Endpoint: registryServer.URL,
		}))

		t.Run("GIVEN an artifact", bdd.GivenT(func(b bdd.T) {
			ns, _ := content.NamespaceFromContext(ctx)

			named := bdd.Must(reference.WithName("test/manifest"))
			repo := bdd.Must(reg.Repository(ctx, named))

			b.When("push this image to container registry", func(b bdd.T) {
				b.Then("success pushed",
					bdd.NoError(remote.Push(ctx, src, repo, "latest")),
				)

				b.When("pull and push as v1", func(b bdd.T) {
					imagePushed := bdd.Must(remote.Manifest(ctx, repo, "latest"))

					b.Then("success",
						bdd.NoError(remote.Push(ctx, imagePushed, repo, "v1")),
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
