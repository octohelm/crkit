package remote_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alecthomas/units"
	"github.com/go-json-experiment/json"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"
	"github.com/octohelm/crkit/pkg/content"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/content/remote/authn"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
	"github.com/octohelm/crkit/pkg/uploadcache"
	testingx "github.com/octohelm/x/testing"
)

func TestNamespace(t *testing.T) {
	rh := registry.New()

	var remoteRegistry *httptest.Server

	remoteRegistry = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("registry", req.Method, req.URL.String(), req.Header)

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

	ctx := context.Background()

	namespace, err := contentremote.New(ctx, contentremote.Registry{
		Endpoint: remoteRegistry.URL,
		Username: "test",
		Password: "test",
	})
	testingx.Expect(t, err, testingx.BeNil[error]())

	uploadCache := &uploadcache.MemUploadCache{}
	uploadCache.SetDefaults()
	err = uploadCache.Init(ctx)
	testingx.Expect(t, err, testingx.BeNil[error]())

	h, err := httprouter.New(apis.R, "registry")
	testingx.Expect(t, err, testingx.BeNil[error]())

	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
		}

		ctx := content.NamespaceInjectContext(req.Context(), namespace)
		ctx = uploadcache.UploadCacheInjectContext(ctx, uploadCache)

		fmt.Println("proxy", req.Method, req.URL, req.Header)

		h.ServeHTTP(w, req.WithContext(ctx))
	}))

	reg, err := name.NewRegistry(
		strings.TrimPrefix(registryServer.URL, "http://"),
		name.Insecure,
	)
	testingx.Expect(t, err, testingx.BeNil[error]())

	t.Run("push manifest", func(t *testing.T) {
		img, err := random.Image(int64(100*units.MiB+101*units.KiB), 1)
		testingx.Expect(t, err, testingx.BeNil[error]())

		repo := reg.Repo("test", "x")

		ref := repo.Tag("latest")

		t.Run("could push", func(t *testing.T) {
			err = remote.Push(ref, img)
			testingx.Expect(t, err, testingx.BeNil[error]())
		})

		t.Run("then pull and push as v1", func(t *testing.T) {
			img1, err := remote.Image(ref)
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = remote.Push(repo.Tag("v1"), img1)
			testingx.Expect(t, err, testingx.BeNil[error]())

			t.Run("then could do with tags", func(t *testing.T) {
				r, _ := namespace.Repository(ctx, content.Name("test/x"))

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
