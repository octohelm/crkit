package kubepkg

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/octohelm/crkit/pkg/ocitar"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	testingx "github.com/octohelm/x/testing"
)

//go:embed testdata/example.kubepkg.json
var kubepkgExample []byte

func TestPacker(t *testing.T) {
	t.Skip()

	registry, _ := name.NewRegistry(os.Getenv("CONTAINER_REGISTRY"))

	a := authn.FromConfig(authn.AuthConfig{
		Username: os.Getenv("CONTAINER_REGISTRY_USERNAME"),
		Password: os.Getenv("CONTAINER_REGISTRY_PASSWORD"),
	})

	// {{registry}}/{{namespace}}/{{name}}
	renamer, _ := NewTemplateRenamer("docker.io/x/{{ .name }}")

	t.Run("test rename", func(t *testing.T) {
		r, _ := name.NewRepository("docker.io/library/nginx")
		testingx.Expect(t, renamer.Rename(r), testingx.Be("docker.io/x/nginx"))
	})

	p := &Packer{
		Cache:    cache.NewFilesystemCache("testdata/.tmp/cache"),
		Registry: &registry,
		CreatePuller: func(name name.Repository, options ...remote.Option) (*remote.Puller, error) {
			return remote.NewPuller(append(options, remote.WithAuth(a))...)
		},
		Platforms: []string{
			"linux/amd64",
		},
		Renamer: renamer,
	}

	t.Run("should pack as kubepkg image", func(t *testing.T) {
		kpkg := &kubepkgv1alpha1.KubePkg{}
		_ = json.Unmarshal(kubepkgExample, kpkg)

		ctx := context.Background()

		i, err := p.PackAsKubePkgImage(ctx, kpkg)
		testingx.Expect(t, err, testingx.BeNil[error]())

		raw, _ := i.RawManifest()
		fmt.Println(string(raw))
	})

	t.Run("should pack as index", func(t *testing.T) {
		kpkg := &kubepkgv1alpha1.KubePkg{}
		_ = json.Unmarshal(kubepkgExample, kpkg)

		ctx := context.Background()

		idx, err := p.PackAsIndex(ctx, kpkg)
		testingx.Expect(t, err, testingx.BeNil[error]())

		f, err := os.OpenFile("testdata/.tmp/example.kubepkg.tar", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.BeNil[error]())
		defer f.Close()

		err = ocitar.Write(f, idx)
		testingx.Expect(t, err, testingx.BeNil[error]())
	})
}
