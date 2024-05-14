package kubepkg

import (
	"context"
	"iter"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/containerd/containerd/images"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/octohelm/crkit/pkg/artifact"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	"github.com/octohelm/kubepkgspec/pkg/object"
	"github.com/octohelm/kubepkgspec/pkg/workload"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	AnnotationSourceBaseImageName = "kubepkg.source.image.base.name"
)

type Packer struct {
	CreatePuller func(repo name.Repository, options ...remote.Option) (*remote.Puller, error)
	Cache        cache.Cache
	Registry     *name.Registry
	Platforms    []string
	Renamer      Renamer

	sourceImages sync.Map
}

func (p *Packer) SupportedPlatforms(supportedPlatform []string) iter.Seq[v1.Platform] {
	if len(p.Platforms) == 0 {
		return func(yield func(v1.Platform) bool) {
			for _, platform := range supportedPlatform {
				p, _ := v1.ParsePlatform(platform)
				if p != nil {
					if !yield(*p) {
						return
					}
				}
			}
		}
	}

	supportedPlatforms := map[string]bool{}
	for _, platform := range supportedPlatform {
		supportedPlatforms[platform] = true
	}

	return func(yield func(v1.Platform) bool) {
		for _, platform := range p.Platforms {
			if len(supportedPlatforms) > 0 {
				_, supported := supportedPlatforms[platform]
				if !supported {
					continue
				}
			}

			p, _ := v1.ParsePlatform(platform)
			if p != nil {
				if !yield(*p) {
					return
				}
			}
		}
	}
}

func (p *Packer) Repository(repoName string) (name.Repository, error) {
	if registry := p.Registry; registry != nil {
		registryName := registry.Name()
		if strings.HasPrefix(repoName, registryName) {
			return registry.Repo(repoName[len(registryName)+1:]), nil
		}
		return registry.Repo(repoName), nil
	}
	return name.NewRepository(repoName)
}

func (p *Packer) Puller(repo name.Repository, options ...remote.Option) (*remote.Puller, error) {
	puller, err := p.CreatePuller(repo, options...)
	if err != nil {
		return nil, err
	}
	return puller, nil
}

func (p *Packer) Image(i v1.Image) v1.Image {
	if c := p.Cache; c != nil {
		return cache.Image(i, c)
	}
	return i
}

func (p *Packer) PackAsIndex(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (v1.ImageIndex, error) {
	kubePkgImage, err := p.PackAsKubePkgImage(ctx, kpkg)
	if err != nil {
		return nil, err
	}

	var finalIndex v1.ImageIndex = empty.Index

	finalIndex, err = p.appendManifests(finalIndex, kubePkgImage, nil, nil)
	if err != nil {
		return nil, err
	}

	layers, err := kubePkgImage.Layers()
	if err != nil {
		return nil, err
	}

	imageNames := make([]string, 0)
	imageIndexes := make(map[string]v1.ImageIndex)

	for _, l := range layers {
		desc, err := partial.Descriptor(l)
		if err != nil {
			return nil, err
		}

		if desc.MediaType.IsImage() && len(desc.Annotations) > 0 {
			imageName := desc.Annotations[images.AnnotationImageName]

			sourceRepo := desc.Annotations[AnnotationSourceBaseImageName]
			repo, err := p.Repository(sourceRepo)
			if err != nil {
				return nil, err
			}

			if _, ok := imageIndexes[imageName]; !ok {
				imageNames = append(imageNames, imageName)
				imageIndexes[imageName] = empty.Index
			}

			puller, err := p.Puller(repo)
			if err != nil {
				return nil, err
			}

			resolvedDesc, err := puller.Get(ctx, repo.Digest(desc.Digest.String()))
			if err != nil {
				return nil, err
			}

			img, err := resolvedDesc.Image()
			if err != nil {
				return nil, err
			}

			imageIndexes[imageName], err = p.appendManifests(imageIndexes[imageName], p.Image(img), desc, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	sort.Strings(imageNames)

	for _, imageName := range imageNames {
		index := imageIndexes[imageName]

		nameAndTag := strings.Split(imageName, ":")
		if len(nameAndTag) != 2 {
			return nil, errors.Errorf("invalid image name %s", nameAndTag)
		}

		finalIndex, err = p.appendManifests(finalIndex, index, nil, &kubepkgv1alpha1.Image{
			Name: nameAndTag[0],
			Tag:  nameAndTag[1],
		})
		if err != nil {
			return nil, err
		}
	}

	return finalIndex, nil
}

func (p *Packer) PackAsKubePkgImage(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (v1.Image, error) {
	workloadImages := workload.Images(func(yield func(object.Object) bool) {
		if !yield(kpkg) {
			return
		}
	})

	var kubepkgImage v1.Image = empty.Image

	for image := range workloadImages {
		repo, err := p.Repository(image.Name)
		if err != nil {
			return nil, err
		}

		image.Name = p.ImageName(repo)
		image.Digest = ""

		for platform := range p.SupportedPlatforms(image.Platforms) {
			puller, err := p.CreatePuller(repo, remote.WithPlatform(platform))
			if err != nil {
				return nil, err
			}

			desc, err := puller.Get(ctx, repo.Tag(image.Tag))
			if err != nil {
				return nil, err
			}

			img, err := desc.Image()
			if err != nil {
				return nil, err
			}

			kubepkgImage, err = p.appendArtifactLayer(kubepkgImage, p.Image(img), image)
			if err != nil {
				return nil, err
			}
		}
	}

	return artifact.Artifact(kubepkgImage, &Config{KubePkg: kpkg})
}

func (p *Packer) appendArtifactLayer(kubepkgImage v1.Image, src v1.Image, img *kubepkgv1alpha1.Image) (v1.Image, error) {
	d, err := partial.Descriptor(src)
	if err != nil {
		return nil, err
	}

	if d.Annotations == nil {
		d.Annotations = map[string]string{}
	}

	d.Annotations[specv1.AnnotationBaseImageName] = img.Name
	d.Annotations[AnnotationSourceBaseImageName] = p.SourceImageName(img.Name)

	d.Annotations[specv1.AnnotationRefName] = img.Tag
	d.Annotations[images.AnnotationImageName] = img.FullName()

	raw, err := src.RawManifest()
	if err != nil {
		return nil, err
	}

	layer, err := artifact.FromBytes(string(d.MediaType), raw)
	if err != nil {
		return nil, err
	}

	return mutate.AppendLayers(kubepkgImage, artifact.WithDescriptor(layer, *d))
}

func (p *Packer) appendManifests(idx v1.ImageIndex, source partial.Describable, desc *v1.Descriptor, image *kubepkgv1alpha1.Image) (v1.ImageIndex, error) {
	if desc == nil {
		d, err := partial.Descriptor(source)
		if err != nil {
			return nil, err
		}
		desc = d
	}

	add := mutate.IndexAddendum{
		Add:        source,
		Descriptor: *desc,
	}

	if image != nil {
		if add.Annotations == nil {
			add.Annotations = map[string]string{}
		}
		add.Annotations[specv1.AnnotationBaseImageName] = image.Name
		add.Annotations[specv1.AnnotationRefName] = image.Tag
		add.Annotations[images.AnnotationImageName] = image.FullName()
	}

	return mutate.AppendManifests(idx, add), nil
}

func (p *Packer) SourceImageName(name string) string {
	if v, ok := p.sourceImages.Load(name); ok {
		return v.(string)
	}
	return name
}

func (p *Packer) ImageName(repoName name.Repository) (name string) {
	defer func() {
		p.sourceImages.Store(name, repoName.String())
	}()

	if p.Renamer != nil {
		return p.Renamer.Rename(repoName)
	}
	if strings.HasPrefix(repoName.String(), "index.docker.io/") {
		return "docker.io/" + repoName.RepositoryStr()
	}
	return repoName.String()
}
