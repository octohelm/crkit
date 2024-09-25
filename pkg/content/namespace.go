package content

import (
	"context"

	"github.com/distribution/reference"
	contextx "github.com/octohelm/x/context"
)

type Namespace interface {
	Repository(ctx context.Context, named reference.Named) (Repository, error)
}

var NamespaceContext = contextx.New[Namespace]()
