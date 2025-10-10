package uploadpurger

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"path"
	"time"

	"github.com/innoai-tech/infra/pkg/agent"
	"github.com/innoai-tech/infra/pkg/cron"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/exp/xiter"
	"github.com/octohelm/x/logr"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
)

// +gengo:injectable
type UploadPurger struct {
	agent.Agent

	ExpiresIn strfmt.Duration `flags:",omitzero"`
	Period    cron.Spec       `flags:",omitzero"`

	driver driver.Driver `inject:",opt"`
}

func (u *UploadPurger) Disabled(ctx context.Context) bool {
	return u.driver == nil
}

func (u *UploadPurger) SetDefaults() {
	if u.ExpiresIn == 0 {
		u.ExpiresIn = strfmt.Duration(2 * time.Hour)
	}

	if u.Period.IsZero() {
		u.Period = "@every 10m"
	}
}

func (r *UploadPurger) afterInit(ctx context.Context) error {
	if r.Disabled(ctx) {
		return nil
	}

	r.Host("Purge Uploads", func(ctx context.Context) error {
		for range xiter.Merge(
			xiter.Of(time.Now()),
			r.Period.Times(ctx),
		) {
			r.Go(ctx, func(ctx context.Context) error {
				ctx, l := logr.FromContext(ctx).Start(ctx, "purging")
				defer l.End()

				return r.Purge(ctx)
			})
		}

		return nil
	})

	return nil
}

func (r *UploadPurger) Purge(ctx context.Context) error {
	expiredAt := time.Now().Add(-time.Duration(r.ExpiresIn))

	for bu, err := range r.blobUploads(ctx) {
		if err != nil {
			return err
		}
		if bu.startedAt.Before(expiredAt) {
			if err := r.driver.Delete(ctx, bu.path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *UploadPurger) blobUploads(ctx context.Context) iter.Seq2[*blobUpload, error] {
	return func(yield func(*blobUpload, error) bool) {
		yieldBlobUpload := func(bu *blobUpload) bool {
			return yield(bu, nil)
		}

		err := r.driver.WalkDir(ctx, layout.Default.UploadPath(), func(pathname string, d fs.DirEntry, err error) error {
			if pathname == "." {
				return nil
			}

			if d.IsDir() {
				bu := &blobUpload{}
				bu.id = pathname
				bu.path = path.Dir(layout.Default.UploadDataPath(bu.id))

				content, _ := r.driver.GetContent(ctx, layout.Default.UploadStartedAtPath(bu.id))
				if len(content) > 0 {
					bu.startedAt, _ = time.Parse(time.RFC3339, string(content))
				}

				if !yieldBlobUpload(bu) {
					return fs.SkipAll
				}

				return fs.SkipDir
			}

			return nil
		})
		if err != nil {
			if errors.Is(err, fs.SkipDir) {
				return
			}

			if !yield(nil, err) {
				return
			}
		}
	}
}

type blobUpload struct {
	id        string
	path      string
	startedAt time.Time
}
