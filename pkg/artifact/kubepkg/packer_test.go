package kubepkg

import (
	"log/slog"
	"net/http/httptest"
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

	ns := bdd.Must(contentremote.New(t.Context(), contentremote.RegistryHosts{
		"docker.io": contentremote.RegistryHost{
			Server: s.URL,
		},
		"cr.io": contentremote.RegistryHost{
			Server: s.URL,
		},
	}))

	b.Given("kubepkg related images", func(b bdd.T) {
		named := bdd.Must(reference.ParseNormalizedNamed("docker.io/library/nginx"))
		ctx := logr.LoggerInjectContext(b.Context(), logrslog.Logger(slog.Default()))

		_ = bdd.MustDo(func() (content.Repository, error) {
			repo := bdd.Must(ns.Repository(ctx, named))

			idx := bdd.Must(mutate.AppendManifests(
				empty.Index,
				bdd.Must(mutate.WithPlatform(bdd.Must(random.Image(10, 1)), "linux/amd64")),
				bdd.Must(mutate.WithPlatform(bdd.Must(random.Image(10, 1)), "linux/arm64")),
			))

			if err := remote.Push(ctx, idx, repo, "1.25.0"); err != nil {
				return nil, err
			}

			if err := remote.Push(ctx, idx, repo, "1.24.0"); err != nil {
				return nil, err
			}

			return repo, nil
		})

		b.When("pack for single amd64", func(b bdd.T) {
			kpkg := bdd.MustDo(func() (*kubepkgv1alpha1.KubePkg, error) {
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

			i, err := p.Pack(ctx, kpkg)
			b.Then("success", bdd.NoError(err))

			m := bdd.Must(i.Value(ctx))

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
				k := bdd.Must(KubePkg(ctx, i))

				b.Then("container image should be resolved",
					bdd.Equal("docker.io/x/nginx", k.Spec.Containers["web"].Image.Name),
				)
			})
		})

		b.When("pack for oci index", func(b bdd.T) {
			kpkg := bdd.MustDo(func() (*kubepkgv1alpha1.KubePkg, error) {
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

			b.Then("oci tar written",
				bdd.NoError(ocitar.WriteFile("testdata/.tmp/example.kubepkg.tar", idx)),
			)

			b.When("push", func(b bdd.T) {
				err := remote.PushIndex(ctx, idx, ns)

				b.Then("success", bdd.NoError(err))
			})
		})
	})
}
