package v2

import (
	"github.com/octohelm/courier/pkg/courierhttp"

	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

// ListTag 列出标签
type ListTag struct {
	courierhttp.MethodGet `path:"/{name...}/tags/list"`

	Name registryv2.Name `name:"name" in:"path"`
}

func (*ListTag) ResponseData() *registryv2.TagList {
	return new(registryv2.TagList)
}

func (ListTag) ResponseErrors() []error {
	return []error{
		&registryv2.ErrRepositoryUnknown{},
		&registryv2.ErrRepositoryNameInvalid{},
	}
}
