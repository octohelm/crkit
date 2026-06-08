package util

import (
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

// Deprecated: use apiregistryv2 instead.
var (
	ParseRange      = registryv2.ParseRange
	ErrInvalidRange = registryv2.ErrInvalidRange
)

// Deprecated: use apiregistryv2.Range instead.
//
//go:fix inline
type Range = registryv2.Range
