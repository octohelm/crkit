package content

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/statuserror"
)

type ErrNotImplemented struct {
	statuserror.NotImplemented

	Reason error
}

func (err *ErrNotImplemented) Error() string {
	return fmt.Sprintf("not implemented: %s", err.Reason)
}

type ErrBlobUnknown struct {
	statuserror.NotFound

	Digest digest.Digest
}

func (ErrBlobUnknown) ErrCode() string {
	return "BLOB_UNKNOWN"
}

func (err *ErrBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob digest=%s", err.Digest)
}

type ErrBlobInvalidLength struct {
	statuserror.RequestedRangeNotSatisfiable

	Reason string
}

func (ErrBlobInvalidLength) ErrCode() string {
	return "SIZE_INVALID"
}

func (err *ErrBlobInvalidLength) Error() string {
	return fmt.Sprintf("blob invalid length: %s", err.Reason)
}

type ErrTagUnknown struct {
	statuserror.NotFound

	Tag string
}

func (ErrTagUnknown) ErrCode() string {
	return "MANIFEST_UNKNOWN"
}

func (err *ErrTagUnknown) Error() string {
	return fmt.Sprintf("unknown tag=%s", err.Tag)
}

type ErrRepositoryUnknown struct {
	statuserror.NotFound

	Name string
}

func (ErrRepositoryUnknown) ErrCode() string {
	return "NAME_UNKNOWN"
}

func (err *ErrRepositoryUnknown) Error() string {
	return fmt.Sprintf("unknown repository name=%s", err.Name)
}

type ErrBlobInvalidDigest struct {
	statuserror.BadRequest

	Digest digest.Digest
	Reason error
}

func (ErrBlobInvalidDigest) ErrCode() string {
	return "DIGEST_INVALID"
}

func (err *ErrBlobInvalidDigest) Error() string {
	return fmt.Sprintf("invalid digest %q: %v", err.Digest, err.Reason)
}

type ErrRepositoryNameInvalid struct {
	statuserror.BadRequest

	Name   string
	Reason error
}

func (ErrRepositoryNameInvalid) ErrCode() string {
	return "NAME_INVALID"
}

func (err *ErrRepositoryNameInvalid) Error() string {
	return fmt.Sprintf("repository name %q invalid: %v", err.Name, err.Reason)
}

type ErrManifestUnknown struct {
	statuserror.NotFound

	Name string
	Tag  string
}

func (ErrManifestUnknown) ErrCode() string {
	return "MANIFEST_UNKNOWN"
}

func (err *ErrManifestUnknown) Error() string {
	return fmt.Sprintf("unknown manifest name=%s tag=%s", err.Name, err.Tag)
}

type ErrManifestUnknownRevision struct {
	statuserror.NotFound

	Name     string
	Revision digest.Digest
}

func (ErrManifestUnknownRevision) ErrCode() string {
	return "MANIFEST_UNKNOWN"
}

func (err *ErrManifestUnknownRevision) Error() string {
	return fmt.Sprintf("unknown manifest name=%s revision=%s", err.Name, err.Revision)
}

type ErrManifestUnverified struct {
	statuserror.BadRequest
}

func (ErrManifestUnverified) Error() string {
	return "unverified manifest"
}

type ErrManifestBlobUnknown struct {
	statuserror.NotFound

	Digest digest.Digest
}

func (ErrManifestBlobUnknown) ErrCode() string {
	return "MANIFEST_BLOB_UNKNOWN"
}

func (err *ErrManifestBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob %v on manifest", err.Digest)
}

type ErrManifestNameInvalid struct {
	statuserror.BadRequest

	Name   string
	Reason error
}

func (ErrManifestNameInvalid) ErrCode() string {
	return "NAME_INVALID"
}

func (err *ErrManifestNameInvalid) Error() string {
	return fmt.Sprintf("manifest name %q invalid: %v", err.Name, err.Reason)
}

type ErrBlobUploadUnknown struct {
	statuserror.NotFound
}

func (ErrBlobUploadUnknown) ErrCode() string {
	return "BLOB_UPLOAD_UNKNOWN"
}

func (err *ErrBlobUploadUnknown) Error() string {
	return "blob upload unknown"
}
