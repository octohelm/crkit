package content

import (
	"fmt"

	"github.com/octohelm/courier/pkg/statuserror"
	"github.com/opencontainers/go-digest"
)

type ErrBlobUnknown struct {
	statuserror.NotFound

	Digest digest.Digest
}

func (err *ErrBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob digest=%s", err.Digest)
}

type ErrBlobInvalidLength struct {
	statuserror.RequestedRangeNotSatisfiable

	Reason string
}

func (err *ErrBlobInvalidLength) Error() string {
	return fmt.Sprintf("blob invalid length: %s", err.Reason)
}

type ErrTagUnknown struct {
	statuserror.NotFound

	Tag string
}

func (err *ErrTagUnknown) Error() string {
	return fmt.Sprintf("unknown tag=%s", err.Tag)
}

type ErrRepositoryUnknown struct {
	statuserror.NotFound

	Name string
}

func (err *ErrRepositoryUnknown) Error() string {
	return fmt.Sprintf("unknown repository name=%s", err.Name)
}

type ErrBlobInvalidDigest struct {
	statuserror.BadRequest

	Digest digest.Digest
	Reason error
}

func (err *ErrBlobInvalidDigest) Error() string {
	return fmt.Sprintf("invalid digest %q: %v", err.Digest, err.Reason)
}

type ErrRepositoryNameInvalid struct {
	statuserror.BadRequest

	Name   string
	Reason error
}

func (err *ErrRepositoryNameInvalid) Error() string {
	return fmt.Sprintf("repository name %q invalid: %v", err.Name, err.Reason)
}

type ErrManifestUnknown struct {
	statuserror.NotFound

	Name string
	Tag  string
}

func (err *ErrManifestUnknown) Error() string {
	return fmt.Sprintf("unknown manifest name=%s tag=%s", err.Name, err.Tag)
}

type ErrManifestUnknownRevision struct {
	statuserror.NotFound

	Name     string
	Revision digest.Digest
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

func (err *ErrManifestBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob %v on manifest", err.Digest)
}

type ErrManifestNameInvalid struct {
	statuserror.BadRequest

	Name   string
	Reason error
}

func (err *ErrManifestNameInvalid) Error() string {
	return fmt.Sprintf("manifest name %q invalid: %v", err.Name, err.Reason)
}
