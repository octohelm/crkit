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
	"github.com/octohelm/unifs/pkg/units"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/content"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/collect"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
	"github.com/octohelm/crkit/pkg/registryhttp/apis"
)

func FuzzNamespace(f *testing.F) {
	for i := range 2 {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, i int) {
		t.Run("随机生成 OCI 镜像或索引", func(t *testing.T) {
			manifestN, image := MustValues(t, func() (int, oci.Manifest, error) {
				switch i {
				case 1:
					img, err := random.Image(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5)
					return 1, img, err
				default:
					idx, err := random.Index(int64(units.BinarySize(int64(randv2.IntN(50)))*units.MiB), 5, 2)
					return 3, idx, err
				}
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

				d.Content.Backend.Scheme = "file"
				d.Content.Backend.Path = tmp
			})

			s := MustValue(t, func() (*httptest.Server, error) {
				injector := configuration.ContextInjectorFromContext(ctx)

				h, err := httprouter.New(apis.R, "registry")
				if err != nil {
					return nil, err
				}

				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if strings.HasSuffix(req.URL.Path, "/") {
						req.URL.Path = req.URL.Path[0 : len(req.URL.Path)-1]
					}

					fmt.Println(req.Method, req.URL.String())

					h.ServeHTTP(w, req.WithContext(injector.InjectContext(req.Context())))
				}))

				return s, nil
			})
			t.Cleanup(s.Close)

			reg := MustValue(t, func() (content.Namespace, error) {
				return contentremote.New(ctx, contentremote.Registry{
					Endpoint: s.URL,
				})
			})

			remoteRepo := MustValue(t, func() (content.Repository, error) {
				named, err := reference.WithName("test/manifest")
				if err != nil {
					return nil, err
				}
				return reg.Repository(ctx, named)
			})

			t.Run("OCI artifact 操作流程", func(t *testing.T) {
				ns, _ := content.NamespaceFromContext(ctx)

				Then(t, "推送镜像到容器注册表",
					ExpectDo(
						func() error {
							return remote.Push(ctx, image, remoteRepo, "latest")
						},
					),
				)

				Then(t, "获取目录列表",
					ExpectMustValue(
						func() ([]string, error) {
							return collect.Catalogs(ctx, ns)
						},
						Equal([]string{remoteRepo.Named().Name()}),
					),
				)

				Then(t, "验证推送的manifest数量",
					ExpectMustValue(
						func() (int, error) {
							repo, err := ns.Repository(ctx, remoteRepo.Named())
							if err != nil {
								return 0, err
							}

							manifests, err := repo.Manifests(ctx)
							if err != nil {
								return 0, err
							}

							revisions, err := collect.Manifests(ctx, manifests)
							return len(revisions), err
						},
						Equal(manifestN),
					),
				)

				t.Run("WHEN 拉取并重新推送为v1标签", func(t *testing.T) {
					imagePushed := MustValue(t, func() (oci.Manifest, error) {
						return remote.Manifest(ctx, remoteRepo, "latest")
					})

					Then(t, "成功推送 v1 标签",
						ExpectDo(
							func() error {
								return remote.Push(ctx, imagePushed, remoteRepo, "v1")
							},
							ErrorNotIs(os.ErrNotExist),
						),
					)

					tags := MustValue(t, func() (content.TagService, error) {
						return remoteRepo.Tags(ctx)
					})

					Then(t, "验证存在两个标签",
						ExpectMustValue(
							func() ([]string, error) {
								return tags.All(ctx)
							},
							Equal([]string{"latest", "v1"}),
						),
					)

					t.Run("WHEN 删除 latest 标签", func(t *testing.T) {
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
	})
}
