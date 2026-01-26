package kubepkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-json-experiment/json"

	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"

	"github.com/octohelm/crkit/pkg/oci"
)

func KubePkg(ctx context.Context, m oci.Manifest) (*kubepkgv1alpha1.KubePkg, error) {
	d, err := m.Descriptor(ctx)
	if err != nil {
		return nil, err
	}

	switch d.ArtifactType {
	case IndexArtifactType:
		if kubePkgArtifactIndex, ok := m.(oci.Index); ok {
			for sub := range kubePkgArtifactIndex.Manifests(ctx) {
				d, err := sub.Descriptor(ctx)
				if err != nil {
					return nil, err
				}

				if d.ArtifactType == IndexArtifactType || d.ArtifactType == ArtifactType {
					return KubePkg(ctx, sub)
				}
			}
		}
	case ArtifactType:
		if kubePkgArtifact, ok := m.(oci.Image); ok {
			return func() (*kubepkgv1alpha1.KubePkg, error) {
				img, err := kubePkgArtifact.Config(ctx)
				if err != nil {
					return nil, fmt.Errorf("open config failed: %w", err)
				}

				f, err := img.Open(ctx)
				if err != nil {
					return nil, err
				}
				defer f.Close()

				kpkg := &kubepkgv1alpha1.KubePkg{}
				if err := json.UnmarshalRead(f, kpkg); err != nil {
					return nil, err
				}
				return kpkg, nil
			}()
		}
	}

	return nil, errors.New("not kubepkg artifact or index")
}
