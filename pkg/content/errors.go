package content

import (
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type (
	ErrNotImplemented          = registryv2.ErrNotImplemented
	ErrBlobUnknown             = registryv2.ErrBlobUnknown
	ErrBlobInvalidLength       = registryv2.ErrBlobInvalidLength
	ErrRepositoryUnknown       = registryv2.ErrRepositoryUnknown
	ErrBlobInvalidDigest       = registryv2.ErrBlobInvalidDigest
	ErrRepositoryNameInvalid   = registryv2.ErrRepositoryNameInvalid
	ErrTagUnknown              = registryv2.ErrTagUnknown
	ErrManifestUnknownRevision = registryv2.ErrManifestUnknownRevision
	ErrManifestUnverified      = registryv2.ErrManifestUnverified
	ErrManifestBlobUnknown     = registryv2.ErrManifestBlobUnknown
	ErrManifestNameInvalid     = registryv2.ErrManifestNameInvalid
	ErrBlobUploadUnknown       = registryv2.ErrBlobUploadUnknown
)
