package proxy_test

import (
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
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/unifs/pkg/units"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	contenttestutil "github.com/octohelm/crkit/pkg/content/testutil"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

func FuzzProxyNamespace(f *testing.F) {
	manifests := []oci.Manifest{
		MustValue(f, func() (oci.Manifest, error) {
			return random.Image(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), 5)
		}),
		MustValue(f, func() (oci.Manifest, error) {
			return random.Index(int64(units.BinarySize(int64(randv2.IntN(10)))*units.MiB), 5, 2)
		}),
	}

	for i := range manifests {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, idx int) {
		manifest := manifests[idx]

		remoteRegistry := httptest.NewServer(contenttestutil.NewRegistry(t))
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

			d.Content.Backend = *MustValue(t, func() (*strfmt.Endpoint, error) {
				return strfmt.ParseEndpoint("file://" + tmp)
			})
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

		remoteReg := MustValue(t, func() (content.Namespace, error) {
			return contentremote.New(ctx, contentremote.Registry{
				Endpoint: remoteRegistry.URL,
			})
		})

		proxyReg := MustValue(t, func() (content.Namespace, error) {
			return contentremote.New(ctx, contentremote.Registry{
				Endpoint: registryServer.URL,
			})
		})

		t.Run("代理注册表功能测试", func(t *testing.T) {
			remoteNamed := MustValue(t, func() (reference.Named, error) {
				return reference.WithName("test/manifest")
			})

			remoteRepo := MustValue(t, func() (content.Repository, error) {
				return remoteReg.Repository(ctx, remoteNamed)
			})

			Then(t, "推送镜像到远程注册表",
				ExpectDo(
					func() error {
						return remote.Push(ctx, manifest, remoteRepo, "latest")
					},
				),
			)

			t.Run("从代理注册表拉取并推送为 v1", func(t *testing.T) {
				proxyNamed := MustValue(t, func() (reference.Named, error) {
					return reference.WithName("test/manifest")
				})

				proxyRepo := MustValue(t, func() (content.Repository, error) {
					return proxyReg.Repository(ctx, proxyNamed)
				})

				imagePushed := MustValue(t, func() (oci.Manifest, error) {
					return remote.Manifest(ctx, proxyRepo, "latest")
				})

				Then(t, "成功推送 v1 标签到代理注册表",
					ExpectDo(
						func() error {
							return remote.Push(ctx, imagePushed, proxyRepo, "v1")
						},
					),
				)

				ns, _ := content.NamespaceFromContext(ctx)

				localNamed := MustValue(t, func() (reference.Named, error) {
					return reference.WithName("test/manifest")
				})

				localRepo := MustValue(t, func() (content.Repository, error) {
					return ns.Repository(ctx, localNamed)
				})

				tagService := MustValue(t, func() (content.TagService, error) {
					return localRepo.Tags(ctx)
				})

				Then(t, "验证 latest 标签存在",
					ExpectDo(
						func() error {
							_, err := tagService.Get(ctx, "latest")
							return err
						},
					),
				)

				Then(t, "验证 v1 标签存在",
					ExpectDo(
						func() error {
							_, err := tagService.Get(ctx, "v1")
							return err
						},
					),
				)
			})
		})
	})
}
