package v2

import (
	"io"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type CreateBlobUpload struct {
	courierhttp.MethodPost `path:"/{name...}/blobs/uploads"`

	Name          registryv2.Name   `name:"name" in:"path"`
	ContentLength int               `name:"Content-Length,omitzero" in:"header"`
	ContentType   string            `name:"Content-Type,omitzero" in:"header"`
	Digest        registryv2.Digest `name:"digest,omitzero" in:"query"`
	Blob          io.ReadCloser     `in:"body"`
}

func (CreateBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

type GetBlobUpload struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/uploads/{id}"`

	Name registryv2.Name `name:"name" in:"path"`
	ID   string          `name:"id" in:"path"`
}

type PatchBlobUpload struct {
	courierhttp.MethodPatch `path:"/{name...}/blobs/uploads/{id}"`

	Name  registryv2.Name `name:"name" in:"path"`
	ID    string          `name:"id" in:"path"`
	Chunk io.ReadCloser   `in:"body"`
}

type PutBlobUpload struct {
	courierhttp.MethodPut `path:"/{name...}/blobs/uploads/{id}"`

	Name          registryv2.Name   `name:"name" in:"path"`
	ID            string            `name:"id" in:"path"`
	ContentRange  registryv2.Range  `name:"Content-Range,omitzero" in:"header"`
	ContentLength int64             `name:"Content-Length,omitzero" in:"header"`
	Digest        registryv2.Digest `name:"digest" in:"query"`
	Chunk         io.ReadCloser     `in:"body"`
}

type CancelBlobUpload struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/uploads/{id}"`

	Name registryv2.Name `name:"name" in:"path"`
	ID   string          `name:"id" in:"path"`
}
