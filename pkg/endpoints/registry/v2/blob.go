package v2

import (
	"io"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type GetBlob struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (GetBlob) ResponseData() *io.ReadCloser {
	return new(io.ReadCloser)
}

type HeadBlob struct {
	courierhttp.MethodHead `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (HeadBlob) ResponseData() *courier.NoContent {
	return nil
}

type DeleteBlob struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (DeleteBlob) ResponseData() *courier.NoContent {
	return nil
}
