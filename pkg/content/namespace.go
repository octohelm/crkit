package content

import (
	"context"

	"github.com/distribution/reference"
)

// +gengo:injectable:provider
type Namespace interface {
	Repository(ctx context.Context, named reference.Named) (Repository, error)
}
