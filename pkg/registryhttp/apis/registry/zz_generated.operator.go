/*
Package registry GENERATED BY gengo:operator
DON'T EDIT THIS FILE
*/
package registry

import (
	io "io"

	courier "github.com/octohelm/courier/pkg/courier"
	statuserror "github.com/octohelm/courier/pkg/statuserror"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	content "github.com/octohelm/crkit/pkg/content"
)

func init() {
	R.Register(courier.NewRouter(&BaseURL{}))
}

func (BaseURL) ResponseContent() any {
	return new(map[string]string)
}

func (BaseURL) ResponseData() *map[string]string {
	return new(map[string]string)
}

func init() {
	R.Register(courier.NewRouter(&CancelBlobUpload{}))
}

func (CancelBlobUpload) ResponseContent() any {
	return nil
}

func (CancelBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&Catalog{}))
}

func (Catalog) ResponseContent() any {
	return new(CatalogResponse)
}

func (Catalog) ResponseData() *CatalogResponse {
	return new(CatalogResponse)
}

func (Catalog) ResponseErrors() []error {
	return []error{
		&statuserror.Descriptor{
			Code:    statuserror.ErrCodeFor[content.ErrNotImplemented](),
			Message: "not implemented: {Reason}",
			Status:  501,
		},
	}
}

func init() {
	R.Register(courier.NewRouter(&CreateBlobUpload{}))
}

func (CreateBlobUpload) ResponseContent() any {
	return nil
}

func (CreateBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&DeleteBlob{}))
}

func (DeleteBlob) ResponseContent() any {
	return nil
}

func (DeleteBlob) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&DeleteManifest{}))
}

func (DeleteManifest) ResponseContent() any {
	return nil
}

func (DeleteManifest) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&GetBlob{}))
}

func (GetBlob) ResponseContent() any {
	return new(io.ReadCloser)
}

func (GetBlob) ResponseData() *io.ReadCloser {
	return new(io.ReadCloser)
}

func init() {
	R.Register(courier.NewRouter(&GetBlobUpload{}))
}

func (GetBlobUpload) ResponseContent() any {
	return nil
}

func (GetBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&GetManifest{}))
}

func (GetManifest) ResponseContent() any {
	return new(manifestv1.Payload)
}

func (GetManifest) ResponseData() *manifestv1.Payload {
	return new(manifestv1.Payload)
}

func init() {
	R.Register(courier.NewRouter(&HeadBlob{}))
}

func (HeadBlob) ResponseContent() any {
	return nil
}

func (HeadBlob) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&HeadManifest{}))
}

func (HeadManifest) ResponseContent() any {
	return nil
}

func (HeadManifest) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&ListTag{}))
}

func (ListTag) ResponseContent() any {
	return new(content.TagList)
}

func (ListTag) ResponseData() *content.TagList {
	return new(content.TagList)
}

func init() {
	R.Register(courier.NewRouter(&PatchBlobUpload{}))
}

func (PatchBlobUpload) ResponseContent() any {
	return nil
}

func (PatchBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&PutBlobUpload{}))
}

func (PutBlobUpload) ResponseContent() any {
	return nil
}

func (PutBlobUpload) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}

func init() {
	R.Register(courier.NewRouter(&PutManifest{}))
}

func (PutManifest) ResponseContent() any {
	return nil
}

func (PutManifest) ResponseData() *courier.NoContent {
	return new(courier.NoContent)
}
