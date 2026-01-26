package oci

import (
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
)

func IsImage(mediaType string) bool {
	switch mediaType {
	case ocispecv1.MediaTypeImageManifest, manifestv1.DockerMediaTypeManifest:
		return true
	}
	return false
}

func IsIndex(mediaType string) bool {
	switch mediaType {
	case ocispecv1.MediaTypeImageIndex, manifestv1.DockerMediaTypeManifestList:
		return true
	}
	return false
}
