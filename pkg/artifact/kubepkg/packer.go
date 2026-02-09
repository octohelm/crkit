package kubepkg

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	"github.com/octohelm/kubepkgspec/pkg/object"
	"github.com/octohelm/kubepkgspec/pkg/workload"
	"github.com/octohelm/x/logr"
	syncx "github.com/octohelm/x/sync"

	"github.com/octohelm/crkit/pkg/artifact/kubepkg/renamer"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/remote"
)

const (
	AnnotationImageName           = "kubepkg.image.name"
	AnnotationImageRef            = "kubepkg.image.ref"
	AnnotationSourceBaseImageName = "kubepkg.source.image.base.name"
)

type Packer struct {
	Namespace content.Namespace

	Renamer         renamer.Renamer
	WithAnnotations []string
	ImageOnly       bool

	Platforms []string

	images syncx.Map[string, string]
}

func (p *Packer) PackAsIndex(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (oci.Index, error) {
	kubePkgIndex, err := p.Pack(ctx, kpkg)
	if err != nil {
		return nil, err
	}

	finalIndex := empty.Index

	namespace := kpkg.Namespace
	if namespace == "" {
		namespace = "default"
	}

	named, err := reference.ParseNormalizedNamed(fmt.Sprintf("%s/artifact-kubepkg-%s", namespace, kpkg.Name))
	if err != nil {
		return nil, err
	}

	idx, err := kubePkgIndex.Value(ctx)
	if err != nil {
		return nil, err
	}

	imageIndexes := make(map[string]oci.Index)
	artifacts := make(map[string]struct{})

	for _, desc := range idx.Manifests {
		// skip kubepkg
		if desc.ArtifactType == ArtifactType {
			continue
		}

		if oci.IsImage(desc.MediaType) && len(desc.Annotations) > 0 && desc.Platform != nil {
			sourceNamed, err := reference.ParseNormalizedNamed(desc.Annotations[AnnotationSourceBaseImageName])
			if err != nil {
				return nil, err
			}

			imageName := desc.Annotations[AnnotationImageName]
			imageRef := desc.Annotations[AnnotationImageRef]

			imageRepo, err := p.Namespace.Repository(ctx, sourceNamed)
			if err != nil {
				return nil, err
			}

			if _, ok := imageIndexes[imageName]; !ok {
				imageIndexes[imageName] = empty.Index
			}

			img, err := remote.Manifest(ctx, imageRepo, imageRef)
			if err != nil {
				return nil, err
			}

			d, err := img.Descriptor(ctx)
			if err != nil {
				return nil, err
			}

			if d.MediaType == "" {
				return nil, fmt.Errorf("invalid descriptor %s@%s", imageName, imageRef)
			}

			if d.ArtifactType != "" {
				artifacts[imageName] = struct{}{}
			}

			if d.Platform != nil {
				d.Platform = desc.Platform
			}

			if d.Platform == nil && desc.Platform != nil {
				switch x := img.(type) {
				case oci.Image:
					img, err = mutate.WithPlatform(x, platforms.Format(*desc.Platform))
					if err != nil {
						return nil, err
					}
				}
			}

			imageIndex, err := mutate.AppendManifests(imageIndexes[imageName], img)
			if err != nil {
				return nil, err
			}

			imageIndexes[imageName] = imageIndex
		}
	}

	for _, imageName := range slices.Sorted(maps.Keys(imageIndexes)) {
		nameAndTag := strings.Split(imageName, ":")
		if len(nameAndTag) != 2 {
			return nil, fmt.Errorf("invalid image name %s", nameAndTag)
		}

		index := imageIndexes[imageName]

		if p.ImageOnly && len(imageIndexes) == 1 {
			ann, err := p.includedAnnotations(kpkg.Annotations)
			if err != nil {
				return nil, err
			}
			index, err = mutate.WithAnnotations(index, ann)
			if err != nil {
				return nil, err
			}
		}

		index, err = mutate.AnnotateOpenContainerImageName(index, nameAndTag[0], nameAndTag[1])
		if err != nil {
			return nil, err
		}

		if _, ok := artifacts[imageName]; !ok {
			// only no-artifact could be ctr/docker imported
			index, err = mutate.AnnotateContainerdImageName(index, imageName)
			if err != nil {
				return nil, err
			}
		}

		finalIndex, err = mutate.AppendManifests(finalIndex, index)
		if err != nil {
			return nil, err
		}
	}

	if !p.ImageOnly {
		kubePkgIndex, err = mutate.AnnotateOpenContainerImageName(kubePkgIndex, p.ImageName(named.String()), kpkg.Spec.Version)
		if err != nil {
			return nil, err
		}

		finalIndex, err = mutate.AppendManifests(finalIndex, kubePkgIndex)
		if err != nil {
			return nil, err
		}
	}

	return finalIndex, nil
}

func (p *Packer) Pack(pctx context.Context, kpkg *kubepkgv1alpha1.KubePkg) (oci.Index, error) {
	ctx, l := logr.FromContext(pctx).Start(pctx, "PackKubePkg")
	defer l.End()

	annotations, err := p.includedAnnotations(kpkg.Annotations)
	if err != nil {
		return nil, err
	}

	// kubepkg version as image tag
	annotations[ocispecv1.AnnotationRefName] = kpkg.Spec.Version

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

	kubepkgIdx, err := mutate.WithArtifactType(empty.Index, IndexArtifactType)
	if err != nil {
		return nil, err
	}

	added := map[string]struct{}{}

	for image := range workloadImages {
		fullName := image.FullName()

		named, err := reference.ParseNormalizedNamed(image.Name)
		if err != nil {
			return nil, err
		}

		// maybe renamed
		// if needed, always rename
		image.Name = p.ImageName(named.String())
		// force delete for versioned resolved
		image.Digest = ""

		// only one image could added
		if _, ok := added[fullName]; ok {
			continue
		}

		added[fullName] = struct{}{}

		repo, err := p.Namespace.Repository(ctx, named)
		if err != nil {
			return nil, err
		}

		for platform := range p.SupportedPlatforms(image.Platforms) {
			m, err := remote.Manifest(ctx, repo, image.Tag)
			if err != nil {
				return nil, fmt.Errorf("pull manifest failed: %w", err)
			}

			img, _, err := p.resolveMatchedImage(ctx, m, named, platform)
			if err != nil {
				return nil, fmt.Errorf("resolve image failed: %w", err)
			}

			img, err = mutate.WithAnnotations(img, map[string]string{
				AnnotationSourceBaseImageName: p.SourceImageName(image.Name),
				AnnotationImageName:           image.FullName(),
			})
			if err != nil {
				return nil, err
			}

			img, err = mutate.AnnotateOpenContainerImageName(img, image.Name, image.Tag)
			if err != nil {
				return nil, err
			}

			kubepkgIdx, err = mutate.AppendManifests(kubepkgIdx, img)
			if err != nil {
				return nil, fmt.Errorf("append image failed: %w", err)
			}
		}
	}

	kubepkgArtifact, err := p.kubepkgArtifact(ctx, kpkg, annotations)
	if err != nil {
		return nil, err
	}

	kubepkgIdx, err = mutate.AppendManifests(kubepkgIdx, kubepkgArtifact)
	if err != nil {
		return nil, err
	}

	return mutate.WithAnnotations(kubepkgIdx, annotations)
}

func (p *Packer) kubepkgArtifact(ctx context.Context, kpkg *kubepkgv1alpha1.KubePkg, annotations map[string]string) (oci.Image, error) {
	kubepkgArtifact, err := mutate.With(empty.Image,
		func(base oci.Image) (oci.Image, error) {
			return WithConfig(base, kpkg)
		},
		func(base oci.Image) (oci.Image, error) {
			return mutate.WithAnnotations(base, annotations)
		},
		func(base oci.Image) (oci.Image, error) {
			return mutate.WithArtifactType(base, ArtifactType)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create kubepkg artifact failed: %w", err)
	}
	return kubepkgArtifact, nil
}

func (p *Packer) resolveMatchedImage(ctx context.Context, m oci.Manifest, named reference.Named, platform ocispecv1.Platform) (oci.Image, bool, error) {
	d, err := m.Descriptor(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("read descriptor failed: %w", err)
	}

	matcher := platforms.NewMatcher(platform)

	switch x := m.(type) {
	case oci.Image:
		if d.Platform != nil {
			if matcher.Match(*d.Platform) {
				i, err := mutate.WithPlatform(x, platforms.Format(platform))
				if err != nil {
					return nil, false, err
				}
				i, err = mutate.WithAnnotations(i, map[string]string{
					// remain origin digest for pull
					AnnotationImageRef: string(d.Digest),
				})
				if err != nil {
					return nil, false, err
				}

				return i, true, nil
			}
		}
	case oci.Index:
		for sub, err := range x.Manifests(ctx) {
			if err != nil {
				return nil, false, fmt.Errorf("iter manifest failed: %w", err)
			}
			i, ok, _ := p.resolveMatchedImage(ctx, sub, named, platform)
			if ok {
				return i, true, nil
			}
		}
	}

	return nil, false, fmt.Errorf("%w: %s of %s", ErrPlatformNotMatched, platform, named)
}

var ErrPlatformNotMatched = errors.New("platform no matched")

func (p *Packer) ImageName(srcName string) (name string) {
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

func (p *Packer) SourceImageName(name string) string {
	if srcName, ok := p.images.Load(name); ok {
		return srcName
	}
	return name
}

func (p *Packer) SupportedPlatforms(supportedPlatform []string) iter.Seq[ocispecv1.Platform] {
	if len(p.Platforms) == 0 {
		return func(yield func(ocispecv1.Platform) bool) {
			for _, platform := range supportedPlatform {
				p, err := platforms.Parse(platform)
				if err == nil {
					if !yield(p) {
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

	return func(yield func(ocispecv1.Platform) bool) {
		for _, platform := range p.Platforms {
			if len(supportedPlatforms) > 0 {
				_, supported := supportedPlatforms[platform]
				if !supported {
					continue
				}
			}

			p, err := platforms.Parse(platform)
			if err == nil {
				if !yield(p) {
					return
				}
			}
		}
	}
}

func (p *Packer) includedAnnotations(annotations map[string]string) (map[string]string, error) {
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
