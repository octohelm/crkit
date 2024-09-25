package kubepkg

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync"

	"github.com/octohelm/crkit/pkg/ociutil"

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
	"github.com/pkg/errors"
)

const (
	AnnotationImageName           = "kubepkg.image.name"
	AnnotationSourceBaseImageName = "kubepkg.source.image.base.name"
)

type Packer struct {
	Registry        Registry
	Renamer         Renamer
	WithAnnotations []string
	ImageOnly       bool

	CreatePuller func(ref name.Reference, options ...remote.Option) (*remote.Puller, error)

	Cache        cache.Cache
	Platforms    []string
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
		return registry.Repo(repoName), nil
	}
	return name.NewRepository(repoName)
}

func (p *Packer) Puller(ref name.Reference, options ...remote.Option) (*remote.Puller, error) {
	puller, err := p.CreatePuller(ref, options...)
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
	kubePkgIndex, err := p.PackAsKubePkgIndex(ctx, kpkg)
	if err != nil {
		return nil, err
	}

	var finalIndex v1.ImageIndex = empty.Index

	namespace := kpkg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	r, err := p.Repository(fmt.Sprintf("%s/artifact-kubepkg-%s", namespace, kpkg.Name))
	if err != nil {
		return nil, err
	}

	idx, err := kubePkgIndex.IndexManifest()
	if err != nil {
		return nil, err
	}

	imageNames := make([]string, 0)
	imageIndexes := make(map[string]v1.ImageIndex)

	for _, desc := range idx.Manifests {
		if desc.ArtifactType == ArtifactType {
			continue
		}

		if desc.MediaType.IsImage() && len(desc.Annotations) > 0 {
			imageName := desc.Annotations[AnnotationImageName]
			sourceRepo := desc.Annotations[AnnotationSourceBaseImageName]

			repo, err := p.Repository(sourceRepo)
			if err != nil {
				return nil, err
			}

			if _, ok := imageIndexes[imageName]; !ok {
				imageNames = append(imageNames, imageName)
				imageIndexes[imageName] = empty.Index
			}

			puller, err := p.Puller(repo.Digest(desc.Digest.String()))
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

			if artifactType, err := ociutil.ArtifactType(img); err == nil {
				desc.ArtifactType = artifactType
				img = ociutil.Artifact(img, artifactType)
			}

			index, err := p.appendManifests(imageIndexes[imageName], p.Image(img), &desc, nil)
			if err != nil {
				return nil, err
			}

			imageIndexes[imageName] = index
		}
	}

	sort.Strings(imageNames)

	for _, imageName := range imageNames {
		nameAndTag := strings.Split(imageName, ":")
		if len(nameAndTag) != 2 {
			return nil, errors.Errorf("invalid image name %s", nameAndTag)
		}

		index := imageIndexes[imageName]

		if p.ImageOnly && len(imageNames) == 1 {
			ann, err := p.pickAnnotations(kpkg.Annotations)
			if err != nil {
				return nil, err
			}
			index = mutate.Annotations(index, ann).(v1.ImageIndex)
		}

		finalIndex, err = p.appendManifests(finalIndex, index, nil, &kubepkgv1alpha1.Image{
			Name: nameAndTag[0],
			Tag:  nameAndTag[1],
		})
		if err != nil {
			return nil, err
		}
	}

	if !p.ImageOnly {
		finalIndex, err = p.appendManifests(finalIndex, kubePkgIndex, nil, &kubepkgv1alpha1.Image{
			Name: p.ImageName(r),
			Tag:  kpkg.Spec.Version,
		})
		if err != nil {
			return nil, err
		}
	}

	return finalIndex, nil
}

func (p *Packer) pickAnnotations(annotations map[string]string) (map[string]string, error) {
	picked := map[string]string{}

	if len(annotations) > 0 && len(p.WithAnnotations) > 0 {
		glob, err := Compile(p.WithAnnotations)
		if err != nil {
			return nil, err
		}

		for k, v := range annotations {
			if glob.Match(k) {
				picked[k] = v
			}
		}

	}
	return picked, nil
}

func (p *Packer) PackAsKubePkgIndex(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (v1.ImageIndex, error) {
	ann, err := p.pickAnnotations(kpkg.Annotations)
	if err != nil {
		return nil, err
	}

	ann[specv1.AnnotationRefName] = kpkg.Spec.Version

	workloadImages := workload.Images(func(yield func(object.Object) bool) {
		if !yield(kpkg) {
			return
		}
	})

	if len(p.Platforms) == 0 {
		for image := range workloadImages {
			if len(p.Platforms) == 0 {
				p.Platforms = image.Platforms
			} else if len(image.Platforms) > 0 {
				p.Platforms = intersection(p.Platforms, image.Platforms)
			}
		}
	}

	var kubepkgIdx v1.ImageIndex = empty.Index

	for image := range workloadImages {
		repo, err := p.Repository(image.Name)
		if err != nil {
			return nil, err
		}

		image.Name = p.ImageName(repo)
		image.Digest = ""

		for platform := range p.SupportedPlatforms(image.Platforms) {
			puller, err := p.CreatePuller(repo.Tag(image.Tag), remote.WithPlatform(platform))
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

			d, err := partial.Descriptor(img)
			if err != nil {
				return nil, err
			}

			if artifactType, err := ociutil.ArtifactType(img); err == nil {
				d.ArtifactType = artifactType

				img = ociutil.Artifact(img, artifactType)
			}

			if d.Platform == nil {
				d.Platform = &platform
			}

			if d.Annotations == nil {
				d.Annotations = map[string]string{}
			}

			d.Annotations[AnnotationSourceBaseImageName] = p.SourceImageName(image.Name)

			kubepkgIdx, err = p.appendManifests(kubepkgIdx, p.Image(img), d, image)
			if err != nil {
				return nil, err
			}
		}
	}

	kubepkgArtifact, err := artifact.Artifact(empty.Image, &Config{KubePkg: kpkg}, artifact.WithAnnotations(ann))
	if err != nil {
		return nil, err
	}

	kubepkgIdx, err = p.appendManifests(kubepkgIdx, kubepkgArtifact, nil, nil)
	if err != nil {
		return nil, err
	}

	return artifact.IndexWithArtifactType(kubepkgIdx, IndexArtifactType, artifact.WithAnnotations(ann))
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

		if image.Name != "" {
			add.Annotations[specv1.AnnotationBaseImageName] = image.Name

			add.Annotations[AnnotationImageName] = image.FullName()

			if add.ArtifactType == "" {
				add.Annotations[images.AnnotationImageName] = image.FullName()
			}
		}

		if image.Tag != "" {
			add.Annotations[specv1.AnnotationRefName] = image.Tag
		}
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

func intersection[E comparable](a []E, b []E) (c []E) {
	includes := map[E]bool{}
	for i := range a {
		includes[a[i]] = true
	}

	c = make([]E, 0, len(a)+len(b))
	for i := range b {
		x := b[i]

		if _, ok := includes[x]; ok {
			c = append(c, x)
		}
	}

	return
}
