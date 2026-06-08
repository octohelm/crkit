package v2

import (
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type Catalog struct {
	courierhttp.MethodGet `path:"/_catalog"`
}

func (Catalog) ResponseData() *registryv2.CatalogResponse {
	return new(registryv2.CatalogResponse)
}
