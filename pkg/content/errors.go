package content

import (
	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type (
	// Deprecated: use apiregistryv2.ErrNotImplemented instead.
	//go:fix inline
	ErrNotImplemented = apiregistryv2.ErrNotImplemented

	// Deprecated: use apiregistryv2.ErrBlobUnknown instead.
	//go:fix inline
	ErrBlobUnknown = apiregistryv2.ErrBlobUnknown

	// Deprecated: use apiregistryv2.ErrBlobInvalidLength instead.
	//go:fix inline
	ErrBlobInvalidLength = apiregistryv2.ErrBlobInvalidLength

	// Deprecated: use apiregistryv2.ErrRepositoryUnknown instead.
	//go:fix inline
	ErrRepositoryUnknown = apiregistryv2.ErrRepositoryUnknown

	// Deprecated: use apiregistryv2.ErrBlobInvalidDigest instead.
	//go:fix inline
	ErrBlobInvalidDigest = apiregistryv2.ErrBlobInvalidDigest

	// Deprecated: use apiregistryv2.ErrRepositoryNameInvalid instead.
	//go:fix inline
	ErrRepositoryNameInvalid = apiregistryv2.ErrRepositoryNameInvalid

	// Deprecated: use apiregistryv2.ErrTagUnknown instead.
	//go:fix inline
	ErrTagUnknown = apiregistryv2.ErrTagUnknown

	// Deprecated: use apiregistryv2.ErrManifestUnknownRevision instead.
	//go:fix inline
	ErrManifestUnknownRevision = apiregistryv2.ErrManifestUnknownRevision

	// Deprecated: use apiregistryv2.ErrManifestUnverified instead.
	//go:fix inline
	ErrManifestUnverified = apiregistryv2.ErrManifestUnverified

	// Deprecated: use apiregistryv2.ErrManifestBlobUnknown instead.
	//go:fix inline
	ErrManifestBlobUnknown = apiregistryv2.ErrManifestBlobUnknown

	// Deprecated: use apiregistryv2.ErrManifestNameInvalid instead.
	//go:fix inline
	ErrManifestNameInvalid = apiregistryv2.ErrManifestNameInvalid

	// Deprecated: use apiregistryv2.ErrBlobUploadUnknown instead.
	//go:fix inline
	ErrBlobUploadUnknown = apiregistryv2.ErrBlobUploadUnknown
)
