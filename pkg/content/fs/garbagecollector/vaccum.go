package garbagecollector

import (
	"context"
	"fmt"
	"github.com/go-courier/logr"
	"log/slog"
	"os"
	"path"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/opencontainers/go-digest"
)

type Vacuum interface {
	RemoveBlob(ctx context.Context, dgst digest.Digest) error
	RemoveLayer(ctx context.Context, named reference.Named, dgst digest.Digest) error
	RemoveManifest(ctx context.Context, named reference.Named, dgst digest.Digest, allTags []string) error
	RemoveRepository(ctx context.Context, named reference.Named) error
}

func NewVacuum(driver driver.Driver, dryRun bool) Vacuum {
	return &maybeVacuum{
		vacuum: &vacuum{driver: driver, layout: layout.Default},
		dryRun: dryRun,
	}
}

type maybeVacuum struct {
	vacuum *vacuum
	dryRun bool
}

func (m *maybeVacuum) RemoveBlob(ctx context.Context, dgst digest.Digest) error {
	logr.FromContext(ctx).
		WithValues(
			slog.String("blob", string(dgst)),
		).
		Info("removing")
	if m.dryRun {
		return nil
	}
	return m.vacuum.RemoveBlob(ctx, dgst)
}

func (m *maybeVacuum) RemoveLayer(ctx context.Context, named reference.Named, dgst digest.Digest) error {
	logr.FromContext(ctx).
		WithValues(
			slog.String("name", named.String()),
			slog.String("layer", string(dgst)),
		).
		Info("removing")

	if m.dryRun {
		return nil
	}
	return m.vacuum.RemoveLayer(ctx, named, dgst)
}

func (m *maybeVacuum) RemoveManifest(ctx context.Context, named reference.Named, dgst digest.Digest, allTags []string) error {
	logr.FromContext(ctx).
		WithValues(
			slog.String("name", named.String()),
			slog.String("manifest", string(dgst)),
		).
		Info("removing")

	if m.dryRun {
		return nil
	}

	return m.vacuum.RemoveManifest(ctx, named, dgst, allTags)
}

func (m *maybeVacuum) RemoveRepository(ctx context.Context, named reference.Named) error {
	logr.FromContext(ctx).
		WithValues(
			slog.String("repository", named.Name()),
		).
		Info("removing")
	if m.dryRun {
		return nil
	}

	return m.vacuum.RemoveRepository(ctx, named)
}

type vacuum struct {
	driver driver.Driver
	layout layout.Layout
}

func (v *vacuum) RemoveBlob(ctx context.Context, dgst digest.Digest) error {
	if err := dgst.Validate(); err != nil {
		return fmt.Errorf("invalid digest: %s %w", dgst, err)
	}

	// delete blobs/{algorithm}/{hex_digest_prefix_2}/{hex_digest}/data
	return v.driver.Delete(ctx, path.Dir(v.layout.BlobDataPath(dgst)))
}

func (v *vacuum) RemoveLayer(ctx context.Context, named reference.Named, dgst digest.Digest) error {
	if err := dgst.Validate(); err != nil {
		return fmt.Errorf("invalid digest: %s %w", dgst, err)
	}

	// delete repositories/{name}/_layers/{algorithm}/{hex_digest}/link
	return v.driver.Delete(ctx, path.Dir(v.layout.RepositoryLayerLinkPath(named, dgst)))
}

func (v *vacuum) RemoveManifest(ctx context.Context, named reference.Named, dgst digest.Digest, allTags []string) error {
	if err := dgst.Validate(); err != nil {
		return fmt.Errorf("invalid digest: %s %w", dgst, err)
	}

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

func (v *vacuum) RemoveRepository(ctx context.Context, name reference.Named) error {
	return v.driver.Delete(ctx, v.layout.RepositoryPath(name))
}
