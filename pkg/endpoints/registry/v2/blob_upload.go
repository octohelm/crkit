package v2

import (
	"io"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

// CreateBlobUpload 创建分块上传
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

func (CreateBlobUpload) ResponseErrors() []error {
	return []error{
		&registryv2.ErrRepositoryNameInvalid{},
		&registryv2.ErrRepositoryUnknown{},
		&registryv2.ErrBlobInvalidDigest{},
	}
}

// GetBlobUpload 查询分块上传状态
type GetBlobUpload struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/uploads/{id}"`

	Name registryv2.Name `name:"name" in:"path"`
	ID   string          `name:"id" in:"path"`
}

func (GetBlobUpload) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUploadUnknown{},
	}
}

// PatchBlobUpload 上传分块数据
type PatchBlobUpload struct {
	courierhttp.MethodPatch `path:"/{name...}/blobs/uploads/{id}"`

	Name  registryv2.Name `name:"name" in:"path"`
	ID    string          `name:"id" in:"path"`
	Chunk io.ReadCloser   `in:"body"`
}

func (PatchBlobUpload) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUploadUnknown{},
	}
}

// PutBlobUpload 完成分块上传
type PutBlobUpload struct {
	courierhttp.MethodPut `path:"/{name...}/blobs/uploads/{id}"`

	Name          registryv2.Name   `name:"name" in:"path"`
	ID            string            `name:"id" in:"path"`
	ContentRange  registryv2.Range  `name:"Content-Range,omitzero" in:"header"`
	ContentLength int64             `name:"Content-Length,omitzero" in:"header"`
	Digest        registryv2.Digest `name:"digest" in:"query"`
	Chunk         io.ReadCloser     `in:"body"`
}

func (PutBlobUpload) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUploadUnknown{},
		&registryv2.ErrBlobInvalidDigest{},
		&registryv2.ErrBlobInvalidLength{},
	}
}

// CancelBlobUpload 取消分块上传
type CancelBlobUpload struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/uploads/{id}"`

	Name registryv2.Name `name:"name" in:"path"`
	ID   string          `name:"id" in:"path"`
}

func (CancelBlobUpload) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUploadUnknown{},
	}
}
