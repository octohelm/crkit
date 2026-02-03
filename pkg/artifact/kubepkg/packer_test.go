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
	"github.com/octohelm/x/testing/bdd"

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
	b := bdd.FromT(t)

	r := contenttestutil.NewRegistry(t)
	s := httptest.NewServer(r)
	t.Cleanup(s.Close)

	ns := bdd.DoValue(t, func() (content.Namespace, error) {
		return contentremote.New(t.Context(), contentremote.RegistryHosts{
			"docker.io": contentremote.RegistryHost{
				Server: s.URL,
			},
			"cr.io": contentremote.RegistryHost{
				Server: s.URL,
			},
		})
	})

	b.Given("kubepkg related images", func(b bdd.T) {
		named := bdd.DoValue(b, func() (reference.Named, error) {
			return reference.ParseNormalizedNamed("docker.io/library/nginx")
		})

		ctx := logr.LoggerInjectContext(b.Context(), logrslog.Logger(slog.Default()))

		_ = bdd.DoValue(b, func() (content.Repository, error) {
			repo, err := ns.Repository(ctx, named)
			if err != nil {
				return nil, err
			}

			idx, err := mutate.With(
				empty.Index,

				func(base oci.Index) (oci.Index, error) {
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
				},
			)
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

		b.When("pack for single amd64", func(b bdd.T) {
			kpkg := bdd.DoValue(b, func() (*kubepkgv1alpha1.KubePkg, error) {
				kpkg := &kubepkgv1alpha1.KubePkg{}
				if err := json.Unmarshal(kubepkgExample, kpkg); err != nil {
					return nil, err
				}
				return kpkg, nil
			})

			p := &Packer{
				Namespace: ns,
				Renamer:   bdd.Must(renamer.NewTemplate("docker.io/x/{{ .name }}")),
				Platforms: []string{
					"linux/amd64",
				},
			}

			idx, err := p.Pack(ctx, kpkg)
			b.Then("success", bdd.NoError(err))

			m := bdd.DoValue(b, func() (ocispecv1.Index, error) {
				return idx.Value(ctx)
			})

			b.Then("contains 3 manifests ", bdd.Equal(3, len(m.Manifests)))
			b.Then("contains 2 platformed", bdd.Equal(2,
				xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
					return e.Platform != nil && e.Platform.Architecture == "amd64"
				})),
			))
			b.Then("contains 1 kubepkg artifact", bdd.Equal(1,
				xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
					return e.ArtifactType == ArtifactType
				})),
			))

			b.When("resolve index", func(b bdd.T) {
				k := bdd.DoValue(b, func() (*kubepkgv1alpha1.KubePkg, error) {
					return KubePkg(ctx, idx)
				})

				b.Then("container image should be resolved",
					bdd.Equal("docker.io/x/nginx", k.Spec.Containers["web"].Image.Name),
				)
			})
		})

		b.When("pack for oci index", func(b bdd.T) {
			kpkg := bdd.DoValue(b, func() (*kubepkgv1alpha1.KubePkg, error) {
				kpkg := &kubepkgv1alpha1.KubePkg{}
				if err := json.Unmarshal(kubepkgExample, kpkg); err != nil {
					return nil, err
				}
				return kpkg, nil
			})

			p := &Packer{
				Namespace: ns,
				Renamer:   bdd.Must(renamer.NewTemplate("cr.io/x/{{ .name }}")),
			}

			idx, err := p.PackAsIndex(ctx, kpkg)
			b.Then("success", bdd.NoError(err))

			filename := "testdata/.tmp/example.kubepkg.tar"

			b.Then("oci tar written",
				bdd.NoError(ocitar.WriteFile(filename, idx)),
			)

			b.When("push", func(b bdd.T) {
				err := remote.PushIndex(ctx, idx, ns)

				b.Then("success", bdd.NoError(err))

				b.When("read the tar", func(b bdd.T) {
					idx := bdd.DoValue(b, func() (oci.Index, error) {
						return ocitar.Index(func() (io.ReadCloser, error) {
							return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
						})
					})

					i := bdd.DoValue(b, func() (ocispecv1.Index, error) {
						return idx.Value(ctx)
					})

					b.Then("got 3 images",
						bdd.Equal(3, len(i.Manifests)),
					)

					for m := range idx.Manifests(ctx) {
						switch x := m.(type) {
						case oci.Index:
							d := bdd.DoValue(b, func() (ocispecv1.Descriptor, error) {
								return x.Descriptor(ctx)
							})

							if d.ArtifactType == "" {
								b.Then("each image should contains 2 platforms",
									bdd.EqualDoValue(2, func() (int, error) {
										m, err := x.Value(ctx)
										if err != nil {
											return 0, err
										}
										return xiter.Count(xiter.Filter(xiter.Of(m.Manifests...), func(e ocispecv1.Descriptor) bool {
											return e.Platform != nil
										})), nil
									}),
								)
							}
						}
					}
				})
			})
		})
	})
}
