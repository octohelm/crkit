package distribution

import (
	"errors"
	"fmt"
	"strings"
)

var ErrAccessDenied = errors.New("access denied")

var ErrManifestNotModified = errors.New("manifest not modified")

var ErrUnsupported = errors.New("operation unsupported")

var ErrSchemaV1Unsupported = errors.New("manifest schema v1 unsupported")

// ErrTagUnknown is returned if the given tag is not known by the tag service
type ErrTagUnknown struct {
	Tag string
}

func (err ErrTagUnknown) Error() string {
	return fmt.Sprintf("unknown tag=%s", err.Tag)
}

type ErrRepositoryUnknown struct {
	Name string
}

func (err ErrRepositoryUnknown) Error() string {
	return fmt.Sprintf("unknown repository name=%s", err.Name)
}

type ErrRepositoryNameInvalid struct {
	Name   string
	Reason error
}

func (err ErrRepositoryNameInvalid) Error() string {
	return fmt.Sprintf("repository name %q invalid: %v", err.Name, err.Reason)
}

type ErrManifestUnknown struct {
	Name string
	Tag  string
}

func (err ErrManifestUnknown) Error() string {
	return fmt.Sprintf("unknown manifest name=%s tag=%s", err.Name, err.Tag)
}

// ErrManifestUnknownRevision is returned when a manifest cannot be found by
// revision within a repository.
type ErrManifestUnknownRevision struct {
	Name     string
	Revision Digest
}

func (err ErrManifestUnknownRevision) Error() string {
	return fmt.Sprintf("unknown manifest name=%s revision=%s", err.Name, err.Revision)
}

type ErrManifestUnverified struct{}

func (ErrManifestUnverified) Error() string {
	return "unverified manifest"
}

type ErrManifestVerification []error

func (errs ErrManifestVerification) Error() string {
	var parts []string
	for _, err := range errs {
		parts = append(parts, err.Error())
	}

	return fmt.Sprintf("errors verifying manifest: %v", strings.Join(parts, ","))
}

type ErrManifestBlobUnknown struct {
	Digest Digest
}

func (err ErrManifestBlobUnknown) Error() string {
	return fmt.Sprintf("unknown blob %v on manifest", err.Digest)
}

type ErrManifestNameInvalid struct {
	Name   string
	Reason error
}

func (err ErrManifestNameInvalid) Error() string {
	return fmt.Sprintf("manifest name %q invalid: %v", err.Name, err.Reason)
}

type ErrDigestNotMatch struct {
	Expect Digest
	Actual Digest
}

func (err ErrDigestNotMatch) Error() string {
	return fmt.Sprintf("digest not match, expect %q, but got %q", err.Expect, err.Actual)
}
