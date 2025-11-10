package apis

import (
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/courier/pkg/courierhttp/handler/httprouter"

	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
)

var R = courierhttp.GroupRouter("/").With(

	courierhttp.GroupRouter("/api/crkit").With(
		courier.NewRouter(&httprouter.OpenAPI{}),
		courier.NewRouter(&httprouter.OpenAPIView{}),
	),

	registry.R,
)
