package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/distribution/reference"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/x/logr"
	"github.com/octohelm/x/ptr"

	"github.com/octohelm/crkit/internal/pkg/progress"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
)

func PushIndex(ctx context.Context, idx oci.Index, ns content.Namespace) error {
	for m, err := range idx.Manifests(ctx) {
		if err != nil {
			return fmt.Errorf("iter manifest failed: %w", err)
		}
		d, err := m.Descriptor(ctx)
		if err != nil {
			return fmt.Errorf("resolve failed: %w", err)
		}

		if d.Annotations != nil {
			imageName := d.Annotations[ocispecv1.AnnotationBaseImageName]
			imageRef := d.Annotations[ocispecv1.AnnotationRefName]

			if imageName == "" || imageRef == "" {
				continue
			}

			named, err := reference.ParseNormalizedNamed(imageName)
			if err != nil {
				return fmt.Errorf("invalid image name: %w", err)
			}

			repo, err := ns.Repository(ctx, named)
			if err != nil {
				return fmt.Errorf("resolve repo failed: %w", err)
			}

			if err := Push(ctx, m, repo, imageRef); err != nil {
				return err
			}
		}
	}

	return nil
}

func Push(pctx context.Context, m oci.Manifest, repo content.Repository, tag string) error {
	p := &pusher{
		repo: repo,
	}

	ctx, l := logr.FromContext(pctx).Start(pctx, "Push")
	defer l.End()

	if err := p.push(ctx, m); err != nil {
		return err
	}

	if tag != "" {
		d, err := m.Descriptor(ctx)
		if err != nil {
			return err
		}

		if err := p.tag(ctx, tag, d); err != nil {
			return err
		}
	}

	return nil
}

type pusher struct {
	repo content.Repository
}

func (w *pusher) tag(ctx context.Context, tag string, d ocispecv1.Descriptor) error {
	tags, err := w.repo.Tags(ctx)
	if err != nil {
		return err
	}

	if err := tags.Tag(ctx, tag, d); err != nil {
		return err
	}

	return nil
}

func (w *pusher) push(ctx context.Context, m oci.Manifest) error {
	switch x := m.(type) {
	case oci.Index:
		return w.pushIndex(ctx, x)
	case oci.Image:
		return w.pushImage(ctx, x)
	}
	return nil
}

func (p *pusher) pushIndex(ctx context.Context, idx oci.Index) error {
	d, err := idx.Descriptor(ctx)
	if err != nil {
		return err
	}

	manifests, err := p.repo.Manifests(ctx)
	if err != nil {
		return err
	}

	if _, err := manifests.Info(ctx, d.Digest); err != nil {
		if !errors.As(err, ptr.Ptr(&content.ErrManifestUnknownRevision{})) {
			return err
		}
	} else {
		return nil
	}

	for child, err := range idx.Manifests(ctx) {
		if err != nil {
			return fmt.Errorf("resolve manifests failed, %T: %p", idx, err)
		}

		if err := p.push(ctx, child); err != nil {
			return err
		}
	}

	raw, err := idx.Raw(ctx)
	if err != nil {
		return err
	}

	x, err := manifestv1.FromBytes(raw)
	if err != nil {
		return nil
	}

	if _, err := manifests.Put(ctx, x); err != nil {
		return err
	}

	return nil
}

func (p *pusher) pushImage(ctx context.Context, img oci.Image) error {
	d, err := img.Descriptor(ctx)
	if err != nil {
		return err
	}

	manifests, err := p.repo.Manifests(ctx)
	if err != nil {
		return err
	}

	if _, err := manifests.Info(ctx, d.Digest); err != nil {
		if !errors.As(err, ptr.Ptr(&content.ErrManifestUnknownRevision{})) {
			return err
		}
	} else {
		return nil
	}

	c, err := img.Config(ctx)
	if err != nil {
		return err
	}
	if err := p.pushBlob(ctx, c); err != nil {
		return err
	}

	for b := range img.Layers(ctx) {
		if err := p.pushBlob(ctx, b); err != nil {
			return err
		}
	}

	raw, err := img.Raw(ctx)
	if err != nil {
		return err
	}

	x, err := manifestv1.FromBytes(raw)
	if err != nil {
		return nil
	}

	if _, err := manifests.Put(ctx, x); err != nil {
		return err
	}

	return nil
}

func (p *pusher) pushBlob(ctx context.Context, b oci.Blob) error {
	l := logr.FromContext(ctx)

	d, err := b.Descriptor(ctx)
	if err != nil {
		return err
	}

	l = l.WithValues(
		slog.Any("repo.name", p.repo.Named().Name()),
		slog.Any("progress.total", d.Size),
	)

	blobs, err := p.repo.Blobs(ctx)
	if err != nil {
		return err
	}

	if _, err := blobs.Info(ctx, d.Digest); err != nil {
		if !errors.As(err, ptr.Ptr(&content.ErrManifestBlobUnknown{})) {
			return fmt.Errorf("resolve blob failed: %s", err)
		}
	} else {
		return nil
	}

	w, err := blobs.Writer(ctx)
	if err != nil {
		return err
	}
	defer w.Close()

	r, err := b.Open(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	pw := progress.New(w)
	defer pw.Close()

	go func() {
		l.WithValues(slog.Int64("progress.current", 0)).Info("pushing")

		for cur := range pw.Observe(ctx) {
			l.WithValues(slog.Int64("progress.current", cur)).Info("pushing")
		}
	}()

	if _, err := io.Copy(pw, r); err != nil {
		return err
	}

	_, err = w.Commit(ctx, d)
	return err
}
