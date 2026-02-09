package kubepkg

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/distribution/reference"
	"github.com/go-json-experiment/json"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/exp/xiter"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	"github.com/octohelm/x/logr"
	logrslog "github.com/octohelm/x/logr/slog"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/artifact/kubepkg/renamer"
	"github.com/octohelm/crkit/pkg/content"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	contenttestutil "github.com/octohelm/crkit/pkg/content/testutil"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/random"
	"github.com/octohelm/crkit/pkg/oci/remote"
	ocitar "github.com/octohelm/crkit/pkg/oci/tar"
)

import (
	_ "embed"
)

//go:embed testdata/example.kubepkg.json
var kubepkgExample []byte

func TestPacker(t *testing.T) {
	r := contenttestutil.NewRegistry(t)
	s := httptest.NewServer(r)
	t.Cleanup(s.Close)

	ns := MustValue(t, func() (content.Namespace, error) {
		return contentremote.New(t.Context(), contentremote.RegistryHosts{
			"docker.io": contentremote.RegistryHost{
				Server: s.URL,
			},
			"cr.io": contentremote.RegistryHost{
				Server: s.URL,
			},
		})
	})

	t.Run("KubePkg 打包测试", func(t *testing.T) {
		ctx := logr.LoggerInjectContext(t.Context(), logrslog.Logger(slog.Default()))

		named := MustValue(t, func() (reference.Named, error) {
			return reference.ParseNormalizedNamed("docker.io/library/nginx")
		})

		_ = MustValue(t, func() (content.Repository, error) {
			repo, err := ns.Repository(ctx, named)
			if err != nil {
				return nil, err
			}

			idx, err := mutate.With(empty.Index, func(base oci.Index) (oci.Index, error) {
				amd64, err := mutate.With(empty.Image, func(base oci.Image) (oci.Image, error) {
					img, err := random.Image(10, 2)
					if err != nil {
						return nil, err
					}
					return mutate.WithPlatform(img, "linux/amd64")
				})
				if err != nil {
					return nil, err
				}

				arm64, err := mutate.With(empty.Image, func(base oci.Image) (oci.Image, error) {
					img, err := random.Image(10, 2)
					if err != nil {
						return nil, err
					}
					return mutate.WithPlatform(img, "linux/arm64")
				})
				if err != nil {
					return nil, err
				}

				return mutate.AppendManifests(base, amd64, arm64)
			})
			if err != nil {
				return nil, err
			}

			if err := remote.Push(ctx, idx, repo, "1.25.0"); err != nil {
				return nil, err
			}

			if err := remote.Push(ctx, idx, repo, "1.24.0"); err != nil {
				return nil, err
			}

			return repo, nil
		})

		t.Run("打包单平台 (amd64)", func(t *testing.T) {
			kpkg := MustValue(t, func() (*kubepkgv1alpha1.KubePkg, error) {
				kpkg := &kubepkgv1alpha1.KubePkg{}
				if err := json.Unmarshal(kubepkgExample, kpkg); err != nil {
					return nil, err
				}
				return kpkg, nil
			})

			p := &Packer{
				Namespace: ns,
				Renamer: MustValue(t, func() (renamer.Renamer, error) {
					return renamer.NewTemplate("docker.io/x/{{ .name }}")
				}),
				Platforms: []string{
					"linux/amd64",
				},
			}

			idx := MustValue(t, func() (oci.Index, error) {
				return p.Pack(ctx, kpkg)
			})

			m := MustValue(t, func() (ocispecv1.Index, error) {
				return idx.Value(ctx)
			})

			Then(t, "包含 3 个 manifest",
				Expect(len(m.Manifests), Equal(3)),
			)

			Then(t, "包含 2 个平台化镜像",
				Expect(
					xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
						return e.Platform != nil && e.Platform.Architecture == "amd64"
					})),
					Equal(2),
				),
			)

			Then(t, "包含 1 个 kubepkg artifact",
				Expect(
					xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
						return e.ArtifactType == ArtifactType
					})),
					Equal(1),
				),
			)

			t.Run("解析索引", func(t *testing.T) {
				k := MustValue(t, func() (*kubepkgv1alpha1.KubePkg, error) {
					return KubePkg(ctx, idx)
				})

				Then(t, "容器镜像应被正确解析",
					Expect(k.Spec.Containers["web"].Image.Name, Equal("docker.io/x/nginx")),
				)
			})
		})

		t.Run("打包为 OCI 索引", func(t *testing.T) {
			kpkg := MustValue(t, func() (*kubepkgv1alpha1.KubePkg, error) {
				kpkg := &kubepkgv1alpha1.KubePkg{}
				if err := json.Unmarshal(kubepkgExample, kpkg); err != nil {
					return nil, err
				}
				return kpkg, nil
			})

			p := &Packer{
				Namespace: ns,
				Renamer: MustValue(t, func() (renamer.Renamer, error) {
					return renamer.NewTemplate("cr.io/x/{{ .name }}")
				}),
			}

			idx := MustValue(t, func() (oci.Index, error) {
				return p.PackAsIndex(ctx, kpkg)
			})

			filename := "testdata/.tmp/example.kubepkg.tar"

			Then(t, "写入 OCI tar 文件",
				ExpectDo(
					func() error {
						return ocitar.WriteFile(filename, idx)
					},
				),
			)

			t.Run("推送到注册表", func(t *testing.T) {
				Then(t, "成功推送索引",
					ExpectDo(
						func() error {
							return remote.PushIndex(ctx, idx, ns)
						},
					),
				)

				t.Run("读取 tar 文件", func(t *testing.T) {
					idx2 := MustValue(t, func() (oci.Index, error) {
						return ocitar.Index(func() (io.ReadCloser, error) {
							return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
						})
					})

					i := MustValue(t, func() (ocispecv1.Index, error) {
						return idx2.Value(ctx)
					})

					Then(t, "获取 3 个镜像",
						Expect(len(i.Manifests), Equal(3)),
					)

					for m := range idx2.Manifests(ctx) {
						switch x := m.(type) {
						case oci.Index:
							d := MustValue(t, func() (ocispecv1.Descriptor, error) {
								return x.Descriptor(ctx)
							})

							if d.ArtifactType == "" {
								Then(t, "每个镜像应包含 2 个平台",
									ExpectMustValue(
										func() (int, error) {
											m, err := x.Value(ctx)
											if err != nil {
												return 0, err
											}
											return xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
												return e.Platform != nil
											})), nil
										},
										Equal(2),
									),
								)
							}
						}
					}
				})
			})
		})
	})
}
