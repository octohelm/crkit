package garbagecollector

import (
	"context"
	"os"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/opencontainers/go-digest"
)

func NewVacuum(driver driver.Driver) *Vacuum {
	return &Vacuum{driver: driver, layout: layout.Default}
}

type Vacuum struct {
	driver driver.Driver
	layout layout.Layout
}

func (v *Vacuum) RemoveBlob(ctx context.Context, dgst digest.Digest) error {
	// delete blobs/{algorithm}/{hex_digest_prefix_2}/{hex_digest}/data
	return v.driver.Delete(ctx, v.layout.BlobDataPath(dgst))
}

func (v *Vacuum) RemoveLayer(ctx context.Context, named reference.Named, dgst digest.Digest) error {
	// delete repositories/{name}/_layers/{algorithm}/{hex_digest}/link
	return v.driver.Delete(ctx, v.layout.RepositoryLayerLinkPath(named, dgst))
}

func (v *Vacuum) RemoveManifest(ctx context.Context, named reference.Named, dgst digest.Digest, allTags []string) error {
	for _, tag := range allTags {
		// delete repositories/{named}/_manifests/tags/{tag}/index/{algorithm}/{hex_digest}/*
		tagIndexEntryPath := v.layout.RepositoryManifestTagIndexEntryPath(named, tag, dgst)

		if _, err := v.driver.Stat(ctx, tagIndexEntryPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return err
		}

		if err := v.driver.Delete(ctx, tagIndexEntryPath); err != nil {
			return err
		}
	}

	// delete repositories/{named}/_manifests/revisions/{algorithm}/{hex_digest}
	return v.driver.Delete(ctx, v.layout.RepositoryManifestRevisionPath(named, dgst))
}

func (v *Vacuum) RemoveRepository(ctx context.Context, name reference.Named) error {
	return v.driver.Delete(ctx, v.layout.RepositoryPath(name))
}
