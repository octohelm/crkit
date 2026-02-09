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
	. "github.com/octohelm/x/testing/v2"

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
		MustValue(f, func() (oci.Manifest, error) {
			return random.Image(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5)
		}),
		MustValue(f, func() (oci.Manifest, error) {
			return random.Index(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5, 2)
		}),
	}

	for i := range manifests {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		src := manifests[idx]

		remoteRegistry := MustValue(t, func() (remoteRegistry *httptest.Server, err error) {
			rh := contenttestutil.NewRegistry(t)

			s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
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

			return s, nil
		})
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

			d.Content.Backend = *MustValue(t, func() (*strfmt.Endpoint, error) {
				return strfmt.ParseEndpoint("file://" + tmp)
			})

			d.NoCache = true
			d.Remote.Endpoint = remoteRegistry.URL
			d.Remote.Username = "test"
			d.Remote.Password = "test"
		})

		injector := configuration.ContextInjectorFromContext(ctx)

		registryServer := MustValue(t, func() (*httptest.Server, error) {
			h, err := httprouter.New(apis.R, "registry")
			if err != nil {
				return nil, err
			}

			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if strings.HasSuffix(req.URL.Path, "/") {
					req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
				}
				h.ServeHTTP(w, req.WithContext(injector.InjectContext(ctx)))
			}))

			return s, nil
		})
		t.Cleanup(registryServer.Close)

		reg := MustValue(t, func() (content.Namespace, error) {
			return contentremote.New(ctx, contentremote.Registry{
				Endpoint: registryServer.URL,
			})
		})

		t.Run("OCI artifact 操作测试", func(t *testing.T) {
			ns, _ := content.NamespaceFromContext(ctx)

			named := MustValue(t, func() (reference.Named, error) {
				return reference.WithName("test/manifest")
			})

			repo := MustValue(t, func() (content.Repository, error) {
				return reg.Repository(ctx, named)
			})

			Then(t, "推送镜像到容器注册表",
				ExpectDo(
					func() error {
						return remote.Push(ctx, src, repo, "latest")
					},
				),
			)

			t.Run("拉取并重新推送为v1标签", func(t *testing.T) {
				imagePushed := MustValue(t, func() (oci.Manifest, error) {
					return remote.Manifest(ctx, repo, "latest")
				})

				Then(t, "成功推送 v1 标签",
					ExpectDo(
						func() error {
							return remote.Push(ctx, imagePushed, repo, "v1")
						},
					),
				)

				repository := MustValue(t, func() (content.Repository, error) {
					return ns.Repository(ctx, content.Name("test/manifest"))
				})

				tags := MustValue(t, func() (content.TagService, error) {
					return repository.Tags(ctx)
				})

				Then(t, "验证存在两个标签",
					ExpectMustValue(
						func() ([]string, error) {
							return tags.All(ctx)
						},
						Equal([]string{"latest", "v1"}),
					),
				)

				t.Run("删除 latest 标签", func(t *testing.T) {
					Then(t, "成功删除标签",
						ExpectDo(
							func() error {
								return tags.Untag(ctx, "latest")
							},
						),
					)

					Then(t, "只剩 v1 标签",
						ExpectMustValue(
							func() ([]string, error) {
								return tags.All(ctx)
							},
							Equal([]string{"v1"}),
						),
					)
				})
			})
		})
	})
}
