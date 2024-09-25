package operator

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
)

type NameScoped struct {
	courierhttp.Method `path:"/{name...}"`

	Name content.Name `name:"name" in:"path"`
}

func (req *NameScoped) Output(ctx context.Context) (any, error) {
	n := content.NamespaceContext.From(ctx)
	repo, err := n.Repository(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return content.RepositoryContext.Inject(ctx, repo), nil
}
