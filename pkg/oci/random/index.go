package random

import (
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
)

func Index(byteSize int64, layerCountPerImage, imageCount int) (idx oci.Index, err error) {
	idx = empty.Index

	for range imageCount {
		img, err := Image(byteSize, layerCountPerImage)
		if err != nil {
			return nil, err
		}

		idx, err = mutate.AppendManifests(idx, img)
		if err != nil {
			return nil, err
		}
	}

	return idx, nil
}
