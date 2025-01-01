package proxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/innoai-tech/infra/pkg/configuration"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/crkit/internal/testingutil"
	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
	"github.com/octohelm/crkit/pkg/uploadcache"
	"github.com/octohelm/unifs/pkg/strfmt"
	testingx "github.com/octohelm/x/testing"
)

func TestNamespace(t *testing.T) {
	rr := httptest.NewServer(registry.New())

	c := &struct {
		otel.Otel

		MemUploadCache uploadcache.MemUploadCache
		contentapi.NamespaceProvider
	}{}

	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	endpoint, _ := strfmt.ParseEndpoint("file://" + tmp)
	c.Content.Backend = *endpoint
	c.Remote.Endpoint = rr.URL

	ctx := testingutil.NewContext(t, c)
	i := configuration.ContextInjectorFromContext(ctx)

	h, err := httprouter.New(apis.R, "registry")
	testingx.Expect(t, err, testingx.BeNil[error]())

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
		}

		fmt.Println("registry", req.Method, req.URL.String())

		h.ServeHTTP(w, req.WithContext(i.InjectContext(req.Context())))
	}))

	reg, err := name.NewRegistry(strings.TrimPrefix(s.URL, "http://"), name.Insecure)
	testingx.Expect(t, err, testingx.BeNil[error]())

	t.Run("push manifest", func(t *testing.T) {
		img, err := random.Image(2048, 5)
		testingx.Expect(t, err, testingx.BeNil[error]())

		repo := reg.Repo("test", "manifest")
		ref := repo.Tag("latest")

		err = remote.Push(ref, img)
		testingx.Expect(t, err, testingx.BeNil[error]())

		t.Run("then pull and push as v1", func(t *testing.T) {
			img1, err := remote.Image(ref)
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = remote.Push(repo.Tag("v1"), img1)
			testingx.Expect(t, err, testingx.BeNil[error]())

			t.Run("then could do with tags", func(t *testing.T) {
				r, _ := c.Repository(ctx, content.Name("test/manifest"))

				tags, err := r.Tags(ctx)
				testingx.Expect(t, err, testingx.BeNil[error]())

				t.Run("could listed", func(t *testing.T) {
					tagList, err := tags.All(ctx)
					testingx.Expect(t, err, testingx.BeNil[error]())
					testingx.Expect(t, tagList, testingx.Equal([]string{
						"latest", "v1",
					}))
				})

				t.Run("could remove", func(t *testing.T) {
					err := tags.Untag(ctx, "latest")
					testingx.Expect(t, err, testingx.BeNil[error]())

					tagList, err := tags.All(ctx)
					testingx.Expect(t, err, testingx.BeNil[error]())
					testingx.Expect(t, tagList, testingx.Equal([]string{
						"v1",
					}))
				})
			})
		})
	})

	t.Run("push index", func(t *testing.T) {
		index, err := random.Index(2048, 5, 5)
		testingx.Expect(t, err, testingx.BeNil[error]())

		repo := reg.Repo("test", "index")

		ref := repo.Tag("latest")

		err = remote.Push(ref, index)
		testingx.Expect(t, err, testingx.BeNil[error]())

		t.Run("then pull and push as v1", func(t *testing.T) {
			index1, err := remote.Index(ref)
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = remote.Push(repo.Tag("v1"), index1)
			testingx.Expect(t, err, testingx.BeNil[error]())

			t.Run("then could do with tags", func(t *testing.T) {
				r, _ := c.Repository(ctx, content.Name("test/index"))

				tags, err := r.Tags(ctx)
				testingx.Expect(t, err, testingx.BeNil[error]())

				t.Run("could listed", func(t *testing.T) {
					tagList, err := tags.All(ctx)
					testingx.Expect(t, err, testingx.BeNil[error]())
					testingx.Expect(t, tagList, testingx.Equal([]string{
						"latest", "v1",
					}))
				})

				t.Run("could remove", func(t *testing.T) {
					err := tags.Untag(ctx, "latest")
					testingx.Expect(t, err, testingx.BeNil[error]())

					tagList, err := tags.All(ctx)
					testingx.Expect(t, err, testingx.BeNil[error]())
					testingx.Expect(t, tagList, testingx.Equal([]string{
						"v1",
					}))
				})
			})
		})
	})
}
