package v2

import (
	"io"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

// GetBlob 下载 Blob
type GetBlob struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (GetBlob) ResponseData() *io.ReadCloser {
	return new(io.ReadCloser)
}

func (GetBlob) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUnknown{},
		&registryv2.ErrRepositoryNameInvalid{},
		&registryv2.ErrRepositoryUnknown{},
	}
}

// HeadBlob 检查 Blob 是否存在
type HeadBlob struct {
	courierhttp.MethodHead `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (HeadBlob) ResponseData() *courier.NoContent {
	return nil
}

func (HeadBlob) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUnknown{},
		&registryv2.ErrRepositoryNameInvalid{},
		&registryv2.ErrRepositoryUnknown{},
	}
}

// DeleteBlob 删除 Blob
type DeleteBlob struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/{digest}"`

	Name   registryv2.Name   `name:"name" in:"path"`
	Digest registryv2.Digest `name:"digest" in:"path"`
}

func (DeleteBlob) ResponseData() *courier.NoContent {
	return nil
}

func (DeleteBlob) ResponseErrors() []error {
	return []error{
		&registryv2.ErrBlobUnknown{},
	}
}
