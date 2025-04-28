// +gengo:operator:register=R
//
//go:generate go tool devtool gen .
package registry

import (
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
)

var R = courier.NewRouter(
	courierhttp.Group("/v2"),
)
