package uploadcache

import (
	"context"
	"sync"
	"time"

	"github.com/go-courier/logr"
	"github.com/innoai-tech/infra/pkg/cron"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	contextx "github.com/octohelm/x/context"
	"github.com/pkg/errors"
)

var Context = contextx.New[UploadCache]()

type UploadCache interface {
	BlobWriter(ctx context.Context, repo content.Repository) (content.BlobWriter, error)
	Resume(ctx context.Context, id string) (content.BlobWriter, error)
}

type MemUploadCache struct {
	cron.Job

	m sync.Map
}

func (c *MemUploadCache) SetDefaults() {
	if c.Job.Cron == "" {
		c.Job.Cron = "@every 1s"
	}
}

func (c *MemUploadCache) cleanup(ctx context.Context) error {
	now := time.Now()

	expiredWriters := make([]string, 0)

	for _, v := range c.m.Range {
		w := v.(*writer)

		if w.expiresAt.Before(now) {
			expiredWriters = append(expiredWriters, w.ID())
		}
	}

	for _, id := range expiredWriters {
		v, ok := c.m.LoadAndDelete(id)
		if ok {
			w := v.(*writer)
			_ = w.Close()
		}
	}

	return nil
}

func (c *MemUploadCache) Init(ctx context.Context) error {
	c.ApplyAction("upload cache cleanup gc", func(ctx context.Context) {
		if err := c.cleanup(ctx); err != nil {
			logr.FromContext(ctx).Error(err)
		}
	})

	return c.Job.Init(ctx)
}

func (c *MemUploadCache) remove(id string) {
	c.m.Delete(id)
}

func (c *MemUploadCache) InjectContext(ctx context.Context) context.Context {
	return Context.Inject(ctx, c)
}

func (c *MemUploadCache) BlobWriter(ctx context.Context, repo content.Repository) (content.BlobWriter, error) {
	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}
	w, err := blobs.Writer(ctx)
	if err != nil {
		return nil, err
	}

	ww := &writer{
		c:          c,
		BlobWriter: w,
	}
	ww.expires()

	c.m.Store(ww.ID(), ww)
	return ww, nil
}

func (c *MemUploadCache) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	v, ok := c.m.Load(id)
	if ok {
		return v.(*writer), nil
	}

	return nil, errors.Errorf("invalid upload session %s", id)
}

type writer struct {
	c         *MemUploadCache
	expiresAt time.Time
	content.BlobWriter
}

func (w *writer) expires() {
	w.expiresAt = time.Now().Add(30 * time.Second)
}

func (w *writer) Write(p []byte) (int, error) {
	defer w.expires()

	return w.BlobWriter.Write(p)
}

func (w *writer) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	defer w.c.remove(w.ID())

	return w.BlobWriter.Commit(ctx, expected)
}
