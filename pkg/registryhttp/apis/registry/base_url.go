package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
)

type BaseURL struct {
	courierhttp.MethodGet
}

func (r *BaseURL) Output(ctx context.Context) (any, error) {
	return map[string]string{}, nil
}
