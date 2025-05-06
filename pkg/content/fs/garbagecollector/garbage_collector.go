package garbagecollector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/agent"
	"github.com/innoai-tech/infra/pkg/cron"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/exp/xiter"
	"github.com/opencontainers/go-digest"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
)

// +gengo:injectable
type GarbageCollector struct {
	agent.Agent

	Period            cron.Spec       `flags:",omitzero"`
	ExcludeModifiedIn strfmt.Duration `flags:",omitzero"`

	driver    driver.Driver     `inject:",opt"`
	namespace content.Namespace `inject:",opt"`
}

func (a *GarbageCollector) Disabled(ctx context.Context) bool {
	return a.driver == nil || a.namespace == nil || a.Period.Schedule() == nil
}

func (a *GarbageCollector) SetDefaults() {
	if a.Period.IsZero() {
		a.Period = "@midnight"
	}

	if a.ExcludeModifiedIn == 0 {
		a.ExcludeModifiedIn = strfmt.Duration(time.Hour)
	}
}

func (a *GarbageCollector) afterInit(ctx context.Context) error {
	if a.Disabled(ctx) {
		return nil
	}

	a.Host("Remove untagged", func(ctx context.Context) error {
		for range xiter.Merge(
			xiter.Of(time.Now()),
			a.Period.Times(ctx),
		) {
			a.Go(ctx, func(ctx context.Context) error {
				ctx, l := logr.FromContext(ctx).Start(ctx, "removing")
				defer l.End()

				return a.MarkAndSweepExcludeModifiedIn(ctx, time.Duration(a.ExcludeModifiedIn))
			})
		}

		return nil
	})

	return nil
}

func (a *GarbageCollector) MarkAndSweepExcludeModifiedIn(ctx context.Context, excludeModifiedIn time.Duration) error {
	repositoryNameIterable, ok := a.namespace.(content.RepositoryNameIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("RepositoryNameIterable of Namespace")}
	}

	blobDigestIterable, ok := a.namespace.(content.DigestIterable)
	if !ok {
		return &content.ErrNotImplemented{Reason: errors.New("DigestIterable of Namespace")}
	}

	stabled := time.Now().Add(-excludeModifiedIn)

	c := &collector{
		Vacuum:    NewVacuum(a.driver),
		namespace: a.namespace,
		used:      map[digest.Digest]struct{}{},
		recentlyActivated: func(modTime time.Time) bool {
			return modTime.After(stabled)
		},
	}

	return c.MarkAndSweep(ctx, repositoryNameIterable, blobDigestIterable)
}

type collector struct {
	*Vacuum
	namespace         content.Namespace
	used              map[digest.Digest]struct{}
	recentlyActivated func(modTime time.Time) bool
}

func (c *collector) count(dgst digest.Digest) {
	c.used[dgst] = struct{}{}
}

func (c *collector) referenced(dgst digest.Digest) bool {
	_, ok := c.used[dgst]
	return ok
}

func (c *collector) referencedOrRecentlyActivated(ld content.LinkedDigest) bool {
	if c.recentlyActivated(ld.ModTime) {
		c.count(ld.Digest)

		return true
	}

	_, ok := c.used[ld.Digest]
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

		if c.referenced(d) {
			continue
		}

		if err := c.RemoveBlob(ctx, d); err != nil {
			return fmt.Errorf("failed to remove blob %s: %w", d, err)
		}

		l.WithValues(slog.String("blob", string(d))).Info("deleted")
	}

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

	for _, tag := range allTags {
		d, err := tagService.Get(ctx, tag)
		if err != nil {
			return err
		}

		if err := c.markManifest(ctx, manifestService, d.Digest); err != nil {
			return err
		}
	}

	for ld, err := range manifestDigestIterable.LinkedDigests(ctx) {
		if err != nil {
			return err
		}

		if c.referencedOrRecentlyActivated(ld) {
			continue
		}

		if err := c.RemoveManifest(ctx, named, ld.Digest, allTags); err != nil {
			return fmt.Errorf("failed to remove manifest %s@%s: %w", named, ld.Digest, err)
		}

		l.WithValues(
			slog.String("name", named.String()),
			slog.String("manifest", string(ld.Digest)),
		).Info("deleted")
	}

	for ld, err := range layerDigestIterable.LinkedDigests(ctx) {
		if err != nil {
			return err
		}

		if c.referencedOrRecentlyActivated(ld) {
			continue
		}

		if err := c.RemoveLayer(ctx, named, ld.Digest); err != nil {
			return fmt.Errorf("failed to remove layer %s@%s: %w", named, ld.Digest, err)
		}

		l.WithValues(
			slog.String("name", named.String()),
			slog.String("layer", string(ld.Digest)),
		).Info("deleted")
	}

	return nil
}

func (c *collector) markManifest(ctx context.Context, manifestService content.ManifestService, manifestDigest digest.Digest) error {
	m, err := manifestService.Get(ctx, manifestDigest)
	if err != nil {
		return err
	}

	c.count(manifestDigest)

	switch m.Type() {
	case manifestv1.DockerMediaTypeManifestList, manifestv1.MediaTypeImageIndex:
		for d := range m.References() {
			if err := c.markManifest(ctx, manifestService, d.Digest); err != nil {
				return err
			}
		}
	default:
		for d := range m.References() {
			c.count(d.Digest)
		}
	}

	return nil
}
