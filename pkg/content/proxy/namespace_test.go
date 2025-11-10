package proxy_test

import (
	randv2 "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/configuration/testingutil"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

func FuzzProxyNamespace(f *testing.F) {
	images := []remote.Taggable{
		bdd.Must(random.Image(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), randv2.Int64N(5))),
		bdd.Must(random.Index(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), randv2.Int64N(5), 2)),
	}

	for i := range images {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		img := images[idx]

		remoteRegistry := httptest.NewServer(registry.New())
		t.Cleanup(remoteRegistry.Close)

		ctx, _ := testingutil.BuildContext(t, func(d *struct {
			otel.Otel

			contentapi.NamespaceProvider
		},
		) {
			tmp := t.TempDir()
			t.Cleanup(func() {
				_ = os.RemoveAll(tmp)
			})

			d.Remote.Endpoint = remoteRegistry.URL

			d.Content.Backend = *bdd.Must(strfmt.ParseEndpoint("file://" + tmp))
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

		remoteReg := bdd.Must(name.NewRegistry(strings.TrimPrefix(remoteRegistry.URL, "http://"), name.Insecure))

		proxyReg := bdd.Must(name.NewRegistry(strings.TrimPrefix(registryServer.URL, "http://"), name.Insecure))

		t.Run("GIVEN an artifact", bdd.GivenT(func(b bdd.T) {
			ns, _ := content.NamespaceFromContext(ctx)

			repo := bdd.Must(ns.Repository(ctx, bdd.Must(reference.WithName("test/manifest"))))
			tagService := bdd.Must(repo.Tags(ctx))

			ref := remoteReg.Repo("test", "manifest").Tag("latest")

			b.When("push this image to remote registry", func(b bdd.T) {
				b.Then("success pushed",
					bdd.NoError(remote.Push(ref, img)),
				)

				b.When("pull from proxy registry and push as v1", func(b bdd.T) {
					img1 := bdd.Must(remote.Image(proxyReg.Repo("test", "manifest").Tag("latest")))

					err := remote.Push(ref.Tag("v1"), img1)
					b.Then("success",
						bdd.NoError(err),
					)

					_ = bdd.Must(tagService.Get(ctx, "latest"))
					_ = bdd.Must(tagService.Get(ctx, "v1"))
				})
			})
		}))
	})
}
