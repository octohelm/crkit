package registry

import (
	"context"
	"fmt"
	"github.com/distribution/distribution/v3/manifest/manifestlist"
	"log/slog"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func MarkAndSweep(ctx context.Context, storageDriver driver.StorageDriver, registry distribution.Namespace, opts storage.GCOpts) error {
	s := &sweeper{
		opts:     opts,
		registry: registry,
		vacuum:   storage.NewVacuum(ctx, storageDriver),
		markSet:  map[digest.Digest]struct{}{},
	}

	if err := s.Mark(ctx); err != nil {
		return err
	}

	return s.Sweep(ctx)
}

type sweeper struct {
	opts        storage.GCOpts
	registry    distribution.Namespace
	vacuum      storage.Vacuum
	markSet     map[digest.Digest]struct{}
	manifestArr []storage.ManifestDel
}

func (s *sweeper) Mark(ctx context.Context) error {
	ctx, l := logr.FromContext(ctx).Start(ctx, "Mark")
	defer l.End()

	repositoryEnumerator, ok := s.registry.(distribution.RepositoryEnumerator)
	if !ok {
		return errors.New("unable to convert Namespace to RepositoryEnumerator")
	}

	return repositoryEnumerator.Enumerate(ctx, func(repoName string) error {
		named, err := reference.WithName(repoName)
		if err != nil {
			return errors.Wrapf(err, "failed to parse repo name %s", repoName)
		}
		repository, err := s.registry.Repository(ctx, named)
		if err != nil {
			return errors.Wrapf(err, "failed to construct repository")
		}
		manifestService, err := repository.Manifests(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to construct manifest service")
		}

		manifestEnumerator, ok := manifestService.(distribution.ManifestEnumerator)
		if !ok {
			return errors.Wrap(err, "unable to convert ManifestService into ManifestEnumerator")
		}

		err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {
			if s.opts.RemoveUntagged {
				// fetch all tags where this manifest is the latest one
				tags, err := repository.Tags(ctx).Lookup(ctx, distribution.Descriptor{Digest: dgst})
				if err != nil {
					return errors.Wrapf(err, "failed to retrieve tags for digest %v", dgst)
				}
				if len(tags) == 0 {
					l.Info("manifest eligible for deletion: %s", dgst)
					// fetch all tags from repository
					// all of these tags could contain manifest in history
					// which means that we need check (and delete) those references when deleting manifest
					allTags, err := repository.Tags(ctx).All(ctx)
					if err != nil {
						return errors.Wrapf(err, "failed to retrieve tags")
					}
					s.manifestArr = append(s.manifestArr, storage.ManifestDel{Name: repoName, Digest: dgst, Tags: allTags})
					return nil
				}
			}

			return s.MarkManifest(ctx, named, dgst)
		})

		// In certain situations such as unfinished uploads, deleting all
		// tags in S3 or removing the _manifests folder manually, this
		// error may be of type PathNotFound.
		//
		// In these cases we can continue marking other manifests safely.
		if errors.As(err, &driver.PathNotFoundError{}) {
			return nil
		}

		return err
	})
}

func (s *sweeper) MarkManifest(ctx context.Context, named reference.Named, dgst digest.Digest) error {
	ctx, l := logr.FromContext(ctx).Start(ctx, "MarkManifest", slog.String("repo", named.String()))
	defer l.End()

	repository, err := s.registry.Repository(ctx, named)
	if err != nil {
		return errors.Wrapf(err, "failed to construct repository")
	}

	manifestService, err := repository.Manifests(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to construct manifest service")
	}

	// Mark the manifest's blob
	l.Info(fmt.Sprintf("marking manifest %s", dgst))
	s.markSet[dgst] = struct{}{}

	manifest, err := manifestService.Get(ctx, dgst)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve manifest for digest %v", dgst)
	}

	descriptors := manifest.References()
	for _, descriptor := range descriptors {
		switch descriptor.MediaType {
		case specv1.MediaTypeImageIndex, specv1.MediaTypeImageManifest,
			schema2.MediaTypeManifest, manifestlist.MediaTypeManifestList:
			if err := s.MarkManifest(ctx, named, descriptor.Digest); err != nil {
				return err
			}
		default:
			s.markSet[descriptor.Digest] = struct{}{}

			l.WithValues(slog.String("mediaType", descriptor.MediaType)).
				Info(fmt.Sprintf("marking blob %s", descriptor.Digest))
		}
	}

	return nil
}

func (s *sweeper) Sweep(ctx context.Context) error {
	if len(s.markSet) == 0 || len(s.manifestArr) == 0 {
		return nil
	}

	ctx, l := logr.FromContext(ctx).Start(ctx, "Sweep")
	defer l.End()

	opts := s.opts
	vacuum := s.vacuum

	if !opts.DryRun {
		for _, obj := range s.manifestArr {
			err := vacuum.RemoveManifest(obj.Name, obj.Digest, obj.Tags)
			if err != nil {
				return errors.Wrapf(err, "failed to delete manifest %s", obj.Digest)
			}
		}
	}

	blobService := s.registry.Blobs()
	deleteSet := make(map[digest.Digest]struct{})
	err := blobService.Enumerate(ctx, func(dgst digest.Digest) error {
		// check if digest is in markSet. If not, delete it!
		if _, ok := s.markSet[dgst]; !ok {
			deleteSet[dgst] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error enumerating blobs")
	}

	l.Info("%d blobs marked, %d blobs and %d manifests eligible for deletion", len(s.markSet), len(deleteSet), len(s.manifestArr))
	for dgst := range deleteSet {
		l.Info("blob eligible for deletion: %s", dgst)
		if opts.DryRun {
			continue
		}
		err = vacuum.RemoveBlob(string(dgst))
		if err != nil {
			return errors.Wrapf(err, "failed to delete blob %s", dgst)
		}
	}

	return err
}
