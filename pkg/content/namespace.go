package content

import (
	"context"
	"iter"

	"github.com/distribution/reference"
)

// +gengo:injectable:provider
type Namespace interface {
	Repository(ctx context.Context, named reference.Named) (Repository, error)
}

type RepositoryNameIterable interface {
	RepositoryNames(ctx context.Context) iter.Seq2[reference.Named, error]
}
