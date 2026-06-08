package util

import (
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

var (
	ParseRange      = registryv2.ParseRange
	ErrInvalidRange = registryv2.ErrInvalidRange
)

type Range = registryv2.Range
