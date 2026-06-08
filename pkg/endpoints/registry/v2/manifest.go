package v2

import (
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type GetManifest struct {
	courierhttp.MethodGet `path:"/{name...}/manifests/{reference}"`

	Name      registryv2.Name      `name:"name" in:"path"`
	Accept    string               `name:"Accept,omitzero" in:"header"`
	Reference registryv2.Reference `name:"reference" in:"path"`
}

func (GetManifest) ResponseData() *manifestv1.Payload {
	return new(manifestv1.Payload)
}

type HeadManifest struct {
	courierhttp.MethodHead `path:"/{name...}/manifests/{reference}"`

	Name      registryv2.Name      `name:"name" in:"path"`
	Accept    string               `name:"Accept,omitzero" in:"header"`
	Reference registryv2.Reference `name:"reference" in:"path"`
}

func (HeadManifest) ResponseData() *courier.NoContent {
	return nil
}

type PutManifest struct {
	courierhttp.MethodPut `path:"/{name...}/manifests/{reference}"`

	Name      registryv2.Name      `name:"name" in:"path"`
	Reference registryv2.Reference `name:"reference" in:"path"`
	Manifest  manifestv1.Payload   `in:"body"`
}

func (PutManifest) ResponseData() *courier.NoContent {
	return nil
}

type DeleteManifest struct {
	courierhttp.MethodDelete `path:"/{name...}/manifests/{reference}"`

	Name      registryv2.Name      `name:"name" in:"path"`
	Reference registryv2.Reference `name:"reference" in:"path"`
}

func (DeleteManifest) ResponseData() *courier.NoContent {
	return nil
}
