package garbagecollector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/x/ptr"
	"github.com/opencontainers/go-digest"
)

func MarkAndSweepExcludeModifiedIn(
	ctx context.Context,
	namespace content.Namespace, d driver.Driver,
	excludeModifiedIn time.Duration,
	dryRun bool,
) error {
	if underlying, ok := namespace.(content.PersistNamespaceWrapper); ok {
		namespace = underlying.UnwarpPersistNamespace()
	}

	repositoryNameIterable, ok := namespace.(content.RepositoryNameIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("RepositoryNameIterable of Namespace")}
	}

	blobDigestIterable, ok := namespace.(content.DigestIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("DigestIterable of Namespace")}
	}

	stabled := time.Now().Add(-excludeModifiedIn)

	c := &collector{
		Vacuum:    NewVacuum(d, dryRun),
		namespace: namespace,
		recentlyActivated: func(modTime time.Time) bool {
			return modTime.After(stabled)
		},
	}

	return c.MarkAndSweep(ctx, repositoryNameIterable, blobDigestIterable)
}

type collector struct {
	Vacuum

	namespace         content.Namespace
	recentlyActivated func(modTime time.Time) bool

	blobUsed map[digest.Digest]struct{}
	refUsed  map[string]map[digest.Digest]struct{}
}

func (c *collector) mark(named reference.Named, dgst digest.Digest) {
	if c.blobUsed == nil {
		c.blobUsed = map[digest.Digest]struct{}{}
	}
	c.blobUsed[dgst] = struct{}{}

	if c.refUsed == nil {
		c.refUsed = map[string]map[digest.Digest]struct{}{}
	}

	name := named.Name()
	if c.refUsed[name] == nil {
		c.refUsed[name] = map[digest.Digest]struct{}{}
	}
	c.refUsed[name][dgst] = struct{}{}
}

func (c *collector) referenced(dgst digest.Digest) bool {
	_, ok := c.blobUsed[dgst]
	return ok
}

func (c *collector) referencedOrRecentlyActivated(named reference.Named, ld content.LinkedDigest) bool {
	if c.recentlyActivated(ld.ModTime) {
		c.mark(named, ld.Digest)
		return true
	}

	repoUsed, ok := c.refUsed[named.Name()]
	if !ok {
		return false
	}
	_, ok = repoUsed[ld.Digest]
	return ok
}

func (c *collector) MarkAndSweep(pctx context.Context, repositoryNameIterable content.RepositoryNameIterable, blobDigestIterable content.DigestIterable) error {
	ctx, l := logr.FromContext(pctx).Start(pctx, "MarkAndSweep")
	defer l.End()

	for named, err := range repositoryNameIterable.RepositoryNames(ctx) {
		if err != nil {
			return err
		}

		if err := c.markAndSweepRepository(ctx, named); err != nil {
			return fmt.Errorf("failed to mark and sweep repository %s: %w", named, err)
		}
	}

	for d, err := range blobDigestIterable.Digests(ctx) {
		if err != nil {
			return err
		}

		l.WithValues(slog.String("blob", string(d))).Debug("checking")

		if c.referenced(d) {
			continue
		}

		if err := c.RemoveBlob(ctx, d); err != nil {
			return fmt.Errorf("failed to remove blob %s: %w", d, err)
		}
	}

	l.Info("all done")

	return nil
}

func (c *collector) markAndSweepRepository(ctx context.Context, named reference.Named) error {
	l := logr.FromContext(ctx)

	repository, err := c.namespace.Repository(ctx, named)
	if err != nil {
		return fmt.Errorf("failed to construct repository: %w", err)
	}

	tagService, err := repository.Tags(ctx)
	if err != nil {
		return fmt.Errorf("failed to tag service: %w", err)
	}

	manifestService, err := repository.Manifests(ctx)
	if err != nil {
		return fmt.Errorf("failed to manifest service: %w", err)
	}

	blobStore, err := repository.Blobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to blob store: %w", err)
	}

	manifestDigestIterable, ok := manifestService.(content.LinkedDigestIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("LinkedDigestIterable of ManifestService")}
	}

	layerDigestIterable, ok := blobStore.(content.LinkedDigestIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("LinkedDigestIterable of BlobStore")}
	}

	allTags, err := tagService.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all tags: %w", err)
	}

	l.WithValues(slog.String("name", named.String())).Info("marking")

	for _, tag := range allTags {
		l.WithValues(
			slog.String("name", named.String()),
			slog.String("tag", tag),
		).Debug("resolving")

		d, err := tagService.Get(ctx, tag)
		if err != nil {
			return fmt.Errorf("failed to get tag %s: %w", tag, err)
		}

		if err := c.markManifest(ctx, named, manifestService, d.Digest); err != nil {
			return fmt.Errorf("failed to mark manifests %s: %w", tag, err)
		}
	}

	for ld, err := range manifestDigestIterable.LinkedDigests(ctx) {
		if err != nil {
			return fmt.Errorf("failed to get linked digest: %w", err)
		}

		l.WithValues(
			slog.String("name", named.String()),
			slog.String("manifest", string(ld.Digest)),
		).Info("checking")

		if c.referencedOrRecentlyActivated(named, ld) {
			continue
		}

		if err := c.RemoveManifest(ctx, named, ld.Digest, allTags); err != nil {
			return fmt.Errorf("failed to remove manifest %s@%s: %w", named, ld.Digest, err)
		}
	}

	for ld, err := range layerDigestIterable.LinkedDigests(ctx) {
		if err != nil {
			return err
		}

		l.WithValues(
			slog.String("name", named.String()),
			slog.String("layer", string(ld.Digest)),
		).Debug("checking")

		if c.referencedOrRecentlyActivated(named, ld) {
			continue
		}

		if err := c.RemoveLayer(ctx, named, ld.Digest); err != nil {
			return fmt.Errorf("failed to remove layer %s@%s: %w", named, ld.Digest, err)
		}
	}

	return nil
}

func (c *collector) markManifest(ctx context.Context, named reference.Named, manifestService content.ManifestService, manifestDigest digest.Digest) error {
	if manifestDigest == "" {
		return nil
	}

	m, err := manifestService.Get(ctx, manifestDigest)
	if err != nil {
		return err
	}

	c.mark(named, manifestDigest)

	switch m.Type() {
	case manifestv1.DockerMediaTypeManifestList, manifestv1.MediaTypeImageIndex:
		for d := range m.References() {
			if err := c.markManifest(ctx, named, manifestService, d.Digest); err != nil {
				// skip for partial cached
				if errors.As(err, ptr.Ptr(&content.ErrManifestUnknownRevision{})) {
					continue
				}
				return err
			}
		}
	default:
		for d := range m.References() {
			c.mark(named, d.Digest)
		}
	}

	return nil
}
