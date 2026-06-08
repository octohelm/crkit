package v2

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/statuserror"
)

// ErrNotImplemented 操作未实现
type ErrNotImplemented struct {
	statuserror.NotImplemented

	// Reason 未实现的原因
	Reason error
}

func (err *ErrNotImplemented) Error() string {
	return fmt.Sprintf("not implemented: %s", err.Reason)
}

// ErrBlobUnknown Blob 不存在
type ErrBlobUnknown struct {
	statuserror.NotFound

	// Digest Blob 摘要
	Digest digest.Digest
}

func (ErrBlobUnknown) ErrCode() string {
	return "BLOB_UNKNOWN"
}

func (err *ErrBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob digest=%s", err.Digest)
}

// ErrBlobInvalidLength Blob 长度不匹配
type ErrBlobInvalidLength struct {
	statuserror.RequestedRangeNotSatisfiable

	// Reason 不匹配的原因
	Reason string
}

func (ErrBlobInvalidLength) ErrCode() string {
	return "SIZE_INVALID"
}

func (err *ErrBlobInvalidLength) Error() string {
	return fmt.Sprintf("blob invalid length: %s", err.Reason)
}

// ErrRepositoryUnknown 仓库不存在
type ErrRepositoryUnknown struct {
	statuserror.NotFound

	// Name 仓库名称
	Name string
}

func (ErrRepositoryUnknown) ErrCode() string {
	return "NAME_UNKNOWN"
}

func (err *ErrRepositoryUnknown) Error() string {
	return fmt.Sprintf("unknown repository name=%s", err.Name)
}

// ErrBlobInvalidDigest Blob 摘要不匹配
type ErrBlobInvalidDigest struct {
	statuserror.BadRequest

	// Digest 期望的摘要
	Digest digest.Digest
	// Reason 不匹配的原因
	Reason error
}

func (ErrBlobInvalidDigest) ErrCode() string {
	return "DIGEST_INVALID"
}

func (err *ErrBlobInvalidDigest) Error() string {
	return fmt.Sprintf("invalid digest %q: %v", err.Digest, err.Reason)
}

// ErrRepositoryNameInvalid 仓库名称无效
type ErrRepositoryNameInvalid struct {
	statuserror.BadRequest

	// Name 仓库名称
	Name string
	// Reason 无效的原因
	Reason error
}

func (ErrRepositoryNameInvalid) ErrCode() string {
	return "NAME_INVALID"
}

func (err *ErrRepositoryNameInvalid) Error() string {
	return fmt.Sprintf("repository name %q invalid: %v", err.Name, err.Reason)
}

// ErrTagUnknown 标签不存在
type ErrTagUnknown struct {
	statuserror.NotFound

	// Name 仓库名称
	Name string
	// Tag 标签名
	Tag string
}

func (ErrTagUnknown) ErrCode() string {
	return "MANIFEST_UNKNOWN"
}

func (err *ErrTagUnknown) Error() string {
	return fmt.Sprintf("unknown manifest name=%s tag=%s", err.Name, err.Tag)
}

// ErrManifestUnknownRevision 清单修订版本不存在
type ErrManifestUnknownRevision struct {
	statuserror.NotFound

	// Name 仓库名称
	Name string
	// Revision 清单摘要
	Revision digest.Digest
}

func (ErrManifestUnknownRevision) ErrCode() string {
	return "MANIFEST_UNKNOWN"
}

func (err *ErrManifestUnknownRevision) Error() string {
	return fmt.Sprintf("unknown manifest name=%s revision=%s", err.Name, err.Revision)
}

// ErrManifestUnverified 清单校验失败
type ErrManifestUnverified struct {
	statuserror.BadRequest
}

func (ErrManifestUnverified) Error() string {
	return "unverified manifest"
}

// ErrManifestBlobUnknown 清单引用的 Blob 不存在
type ErrManifestBlobUnknown struct {
	statuserror.NotFound

	// Name 仓库名称
	Name string
	// Digest Blob 摘要
	Digest digest.Digest
}

func (ErrManifestBlobUnknown) ErrCode() string {
	return "MANIFEST_BLOB_UNKNOWN"
}

func (err *ErrManifestBlobUnknown) Error() string {
	return fmt.Sprintf("unknown manifest name=%s digest=%s", err.Name, err.Digest)
}

// ErrManifestNameInvalid 清单名称无效
type ErrManifestNameInvalid struct {
	statuserror.BadRequest

	// Name 仓库名称
	Name string
	// Reason 无效的原因
	Reason error
}

func (ErrManifestNameInvalid) ErrCode() string {
	return "NAME_INVALID"
}

func (err *ErrManifestNameInvalid) Error() string {
	return fmt.Sprintf("manifest name %q invalid: %v", err.Name, err.Reason)
}

// ErrBlobUploadUnknown 分块上传不存在
type ErrBlobUploadUnknown struct {
	statuserror.NotFound
}

func (ErrBlobUploadUnknown) ErrCode() string {
	return "BLOB_UPLOAD_UNKNOWN"
}

func (err *ErrBlobUploadUnknown) Error() string {
	return "blob upload unknown"
}
