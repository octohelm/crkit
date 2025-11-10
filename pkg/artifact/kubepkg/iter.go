package kubepkg

import (
	"encoding/json"
	"iter"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"

	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
)

func KubePkg(idx v1.ImageIndex) (*kubepkgv1alpha1.KubePkg, error) {
	indexManifest, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}

	for _, m := range indexManifest.Manifests {
		if m.ArtifactType == IndexArtifactType {
			kubePkgIndex, err := idx.ImageIndex(m.Digest)
			if err != nil {
				return nil, err
			}

			kubePkgIndexManifest, err := kubePkgIndex.IndexManifest()
			if err != nil {
				return nil, err
			}

			for _, m := range kubePkgIndexManifest.Manifests {
				if m.ArtifactType == ArtifactType {
					img, err := kubePkgIndex.Image(m.Digest)
					if err != nil {
						return nil, err
					}
					rawConfig, err := img.RawConfigFile()
					if err != nil {
						return nil, err
					}
					kpkg := &kubepkgv1alpha1.KubePkg{}
					if err := json.Unmarshal(rawConfig, kpkg); err != nil {
						return nil, err
					}
					return kpkg, nil
				}
			}
		}
	}

	return nil, nil
}

func NewImageIter(idx v1.ImageIndex) ImageIter {
	return &imageIter{
		ImageIndex: idx,
	}
}

type ImageIter interface {
	Images() iter.Seq2[name.Reference, remote.Taggable]
	Err() error
}

type imageIter struct {
	ImageIndex v1.ImageIndex
	err        error
}

func (r *imageIter) Err() error {
	return r.err
}

func (r *imageIter) Done(err error) {
	r.err = err
}

func (p *imageIter) Repository(repoName string) (name.Repository, error) {
	return name.NewRepository(repoName)
}

func (r *imageIter) Images() iter.Seq2[name.Reference, remote.Taggable] {
	indexManifest, err := r.ImageIndex.IndexManifest()
	if err != nil {
		r.Done(err)

		return func(yield func(name.Reference, remote.Taggable) bool) {
		}
	}

	return func(yield func(name.Reference, remote.Taggable) bool) {
		for _, d := range indexManifest.Manifests {
			if d.Annotations == nil {
				continue
			}

			imageName := d.Annotations[specv1.AnnotationBaseImageName]
			if imageName == "" {
				continue
			}

			tag := d.Annotations[specv1.AnnotationRefName]
			if tag == "" {
				continue
			}

			repo, err := r.Repository(imageName)
			if err != nil {
				r.Done(err)
				return
			}

			ref := repo.Tag(tag)

			if d.MediaType.IsImage() {
				img, err := r.ImageIndex.Image(d.Digest)
				if err != nil {
					r.Done(err)
					return
				}

				if !yield(ref, img) {
					return
				}
			} else if d.MediaType.IsIndex() {
				index, err := r.ImageIndex.ImageIndex(d.Digest)
				if err != nil {
					r.Done(err)
					return
				}
				if !yield(ref, index) {
					return
				}
			}
		}
	}
}
