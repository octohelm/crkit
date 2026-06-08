package content

import (
	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type (
	// Deprecated: use apiregistryv2.Name instead.
	//go:fix inline
	Name = apiregistryv2.Name

	// Deprecated: use apiregistryv2.Digest instead.
	//go:fix inline
	Digest = apiregistryv2.Digest

	// Deprecated: use apiregistryv2.Reference instead.
	//go:fix inline
	Reference = apiregistryv2.Reference
)
