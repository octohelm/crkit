package kubepkg

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/octohelm/crkit/pkg/artifact/kubepkg/cache"
	"github.com/octohelm/crkit/pkg/ocitar"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	testingx "github.com/octohelm/x/testing"
	"github.com/octohelm/x/testing/bdd"
)

//go:embed testdata/example.kubepkg.json
var kubepkgExample []byte

func Test(t *testing.T) {
	imageIndex := mutate.AppendManifests(
		empty.Index,
		mutate.IndexAddendum{
			Add: bdd.Must(random.Image(10, 1)),
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{
					OS:           "linux",
					Architecture: "amd64",
				},
			},
		},
		mutate.IndexAddendum{
			Add: bdd.Must(random.Image(10, 1)),
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{
					OS:           "linux",
					Architecture: "arm64",
				},
			},
		},
	)

	s := httptest.NewServer(registry.New())

	r, err := NewRegistry(s.URL)
	testingx.Expect(t, err, testingx.BeNil[error]())

	err = remote.Push(
		r.Repo("docker.io/library/nginx").Tag("1.25.0-alpine"),
		imageIndex,
	)
	testingx.Expect(t, err, testingx.BeNil[error]())

	// {{registry}}/{{namespace}}/{{name}}
	renamer, _ := NewTemplateRenamer("docker.io/x/{{ .name }}")

	t.Run("test rename", func(t *testing.T) {
		r, _ := name.NewRepository("docker.io/library/nginx")
		testingx.Expect(t, renamer.Rename(r), testingx.Be("docker.io/x/nginx"))
	})

	c := cache.NewFilesystemCache("testdata/.tmp/cache")

	t.Run("with single arch", func(t *testing.T) {
		kpkg := &kubepkgv1alpha1.KubePkg{}
		_ = json.Unmarshal(kubepkgExample, kpkg)

		p := &Packer{
			Cache: c,
			CreatePuller: func(ref name.Reference, options ...remote.Option) (*remote.Puller, error) {
				return remote.NewPuller(options...)
			},
			Registry: r,
			Platforms: []string{
				"linux/amd64",
			},
			Renamer: renamer,
		}

		t.Run("should pack as kubepkg image", func(t *testing.T) {
			kpkg := &kubepkgv1alpha1.KubePkg{}
			_ = json.Unmarshal(kubepkgExample, kpkg)

			ctx := context.Background()

			i, err := p.PackAsKubePkgIndex(ctx, kpkg)
			testingx.Expect(t, err, testingx.BeNil[error]())
			testingx.Expect(t, kpkg.Spec.Containers["web"].Image.Name, testingx.Be("docker.io/x/nginx"))

			index, err := i.IndexManifest()
			testingx.Expect(t, err, testingx.BeNil[error]())
			testingx.Expect(t, len(index.Manifests), testingx.Be(2))
		})

		t.Run("should pack as index", func(t *testing.T) {
			ctx := context.Background()

			idx, err := p.PackAsIndex(ctx, kpkg)
			testingx.Expect(t, err, testingx.BeNil[error]())

			filename := "testdata/.tmp/example.kubepkg.amd64.tar"

			err = writeAsOciTar(filename, idx)
			testingx.Expect(t, err, testingx.BeNil[error]())

			t.Run("should read", func(t *testing.T) {
				idx, err := ocitar.Index(func() (io.ReadCloser, error) {
					return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
				})
				testingx.Expect(t, err, testingx.BeNil[error]())

				found, err := KubePkg(idx)
				testingx.Expect(t, err, testingx.BeNil[error]())
				testingx.Expect(t, found.Spec.Version, testingx.Be(kpkg.Spec.Version))

				t.Run("then could push", func(t *testing.T) {
					pusher := &Pusher{
						Registry: r,
						Renamer:  renamer,
						CreatePusher: func(ref name.Reference, options ...remote.Option) (*remote.Pusher, error) {
							return remote.NewPusher(options...)
						},
					}
					err = pusher.PushIndex(ctx, idx)
					testingx.Expect(t, err, testingx.BeNil[error]())
				})
			})
		})
	})

	t.Run("with multi arch", func(t *testing.T) {
		p := &Packer{
			Cache:    c,
			Registry: r,
			CreatePuller: func(ref name.Reference, options ...remote.Option) (*remote.Puller, error) {
				return remote.NewPuller(options...)
			},
			Platforms: []string{
				"linux/amd64",
				"linux/arm64",
			},
			Renamer: renamer,
			WithAnnotations: []string{
				"kubernetes.io/*",
			},
		}

		t.Run("should pack as kubepkg image", func(t *testing.T) {
			kpkg := &kubepkgv1alpha1.KubePkg{}
			_ = json.Unmarshal(kubepkgExample, kpkg)

			ctx := context.Background()

			i, err := p.PackAsKubePkgIndex(ctx, kpkg)
			testingx.Expect(t, err, testingx.BeNil[error]())

			idx, err := i.IndexManifest()
			testingx.Expect(t, err, testingx.BeNil[error]())
			testingx.Expect(t, len(idx.Manifests), testingx.Be(3))
			testingx.Expect(t, idx.Annotations, testingx.Equal(map[string]string{
				"kubernetes.io/description":         "demo",
				"org.opencontainers.image.ref.name": "v0.0.2",
			}))
		})

		filename := "testdata/.tmp/example.kubepkg.tar"

		t.Run("should pack as index", func(t *testing.T) {
			kpkg := &kubepkgv1alpha1.KubePkg{}
			_ = json.Unmarshal(kubepkgExample, kpkg)

			ctx := context.Background()

			idx, err := p.PackAsIndex(ctx, kpkg)
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = writeAsOciTar(filename, idx)
			testingx.Expect(t, err, testingx.BeNil[error]())
		})

		t.Run("should pack as index", func(t *testing.T) {
			kpkg := &kubepkgv1alpha1.KubePkg{}
			_ = json.Unmarshal(kubepkgExample, kpkg)

			ctx := context.Background()

			idx, err := p.PackAsIndex(ctx, kpkg)
			testingx.Expect(t, err, testingx.BeNil[error]())

			err = writeAsOciTar(filename, idx)
			testingx.Expect(t, err, testingx.BeNil[error]())

			t.Run("should read", func(t *testing.T) {
				idx, err := ocitar.Index(func() (io.ReadCloser, error) {
					return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
				})
				testingx.Expect(t, err, testingx.BeNil[error]())

				found, err := KubePkg(idx)
				testingx.Expect(t, err, testingx.BeNil[error]())
				testingx.Expect(t, found.Spec.Version, testingx.Be(kpkg.Spec.Version))
				testingx.Expect(t, found.Spec.Containers["web"].Image.Name, testingx.Be("docker.io/x/nginx"))

				t.Run("then could push", func(t *testing.T) {
					pusher := &Pusher{
						Registry: r,
						Renamer:  renamer,
						CreatePusher: func(ref name.Reference, options ...remote.Option) (*remote.Pusher, error) {
							return remote.NewPusher(options...)
						},
					}

					err = pusher.PushIndex(ctx, idx)
					testingx.Expect(t, err, testingx.BeNil[error]())
				})
			})
		})
	})
}

func writeAsOciTar(filename string, idx v1.ImageIndex) error {
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return ocitar.Write(f, idx)
}
