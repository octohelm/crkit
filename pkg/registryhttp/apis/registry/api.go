// +gengo:operator:register=R
// +gengo:operator:tag=containerregistry
package registry

import (
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
)

var R = courier.NewRouter(
	courierhttp.Group("/v2"),
)
