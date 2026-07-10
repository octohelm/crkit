package kubepkg

import (
	"context"
	"strings"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/artifact/kubepkg/renamer"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	"github.com/octohelm/kubepkgspec/pkg/object"
	"github.com/octohelm/kubepkgspec/pkg/workload"
	syncx "github.com/octohelm/x/sync"
)

type Renamer struct {
	Renamer renamer.Renamer

	images syncx.Map[string, string]
}

func (p *Renamer) Rename(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (*kubepkgv1alpha1.KubePkg, error) {
	workloadImages := workload.Images(func(yield func(object.Object) bool) {
		if !yield(kpkg) {
			return
		}
	})

	for image := range workloadImages {
		named, err := reference.ParseNormalizedNamed(image.Name)
		if err != nil {
			return nil, err
		}

		image.Name = p.ImageName(named.String())
	}

	return kpkg, nil
}

func (p *Renamer) ImageName(srcName string) (name string) {
	defer func() {
		p.images.Store(name, srcName)
	}()

	if p.Renamer != nil {
		return p.Renamer.Rename(srcName)
	}

	if strings.HasPrefix(srcName, "index.docker.io/") {
		return "docker.io/" + srcName[len("index.docker.io/"):]
	}

	return srcName
}

func (p *Renamer) SourceImageName(name string) string {
	if srcName, ok := p.images.Load(name); ok {
		return srcName
	}
	return name
}
