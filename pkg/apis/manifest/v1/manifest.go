package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Descriptor = specv1.Descriptor

type Manifest interface {
	Type() string
	References() iter.Seq[Descriptor]
}
