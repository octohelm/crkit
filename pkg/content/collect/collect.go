package collect

import (
	"context"
	"errors"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

func Catalogs(ctx context.Context, ns content.Namespace) (catalogs []string, err error) {
	if underlying, ok := ns.(content.PersistNamespaceWrapper); ok {
		ns = underlying.UnwarpPersistNamespace()
	}

	i, ok := ns.(content.RepositoryNameIterable)
	if !ok {
		return nil, &content.ErrNotImplemented{Reason: errors.New("RepositoryNameIterable of TagService")}
	}

	for n, e := range i.RepositoryNames(ctx) {
		if e != nil {
			err = e
			return
		}
		catalogs = append(catalogs, n.Name())
	}

	return
}

func TagRevisions(ctx context.Context, tagService content.TagService, tag string) (digests []digest.Digest, err error) {
	i, ok := tagService.(content.TagRevisionIterable)
	if !ok {
		return nil, &content.ErrNotImplemented{Reason: errors.New("TagRevisionIterable of TagService")}
	}

	for d, e := range i.TagRevisions(ctx, tag) {
		if e != nil {
			err = e
			return
		}
		digests = append(digests, d.Digest)
	}

	return
}

func Manifests(ctx context.Context, manifestService content.ManifestService) (digests []digest.Digest, err error) {
	i, ok := manifestService.(content.LinkedDigestIterable)
	if !ok {
		return nil, &content.ErrNotImplemented{Reason: errors.New("LinkedDigestIterable of ManifestService")}
	}
	for linkedDigest, e := range i.LinkedDigests(ctx) {
		if e != nil {
			err = e
			return
		}
		digests = append(digests, linkedDigest.Digest)
	}
	return
}

func Layers(ctx context.Context, blobStore content.BlobStore) (digests []digest.Digest, err error) {
	i, ok := blobStore.(content.LinkedDigestIterable)
	if !ok {
		return nil, &content.ErrNotImplemented{Reason: errors.New("LinkedDigestIterable of BlobStore")}
	}
	for d, e := range i.LinkedDigests(ctx) {
		if e != nil {
			err = e
			return
		}
		digests = append(digests, d.Digest)
	}
	return
}

func Blobs(ctx context.Context, ns content.Namespace) (digests []digest.Digest, err error) {
	if underlying, ok := ns.(content.PersistNamespaceWrapper); ok {
		ns = underlying.UnwarpPersistNamespace()
	}

	i, ok := ns.(content.DigestIterable)
	if !ok {
		return nil, &content.ErrNotImplemented{Reason: errors.New("DigestIterable of Namespace")}
	}
	for dgst, e := range i.Digests(ctx) {
		if e != nil {
			err = e
			return
		}
		digests = append(digests, dgst)
	}
	return
}
