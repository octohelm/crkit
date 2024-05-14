package ocitar

import (
	"io"
	"sync/atomic"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Update struct {
	Repository name.Repository
	Digest     v1.Hash
	Total      int64
	Complete   int64
}

type progress struct {
	updates chan<- Update
}

func (p *progress) complete(u Update) {
	p.updates <- u
}

type progressReader struct {
	r          io.Reader
	digest     v1.Hash
	repository name.Repository
	total      int64
	count      *int64 // number of bytes this reader has read, to support resetting on retry.
	progress   *progress
}

func (r *progressReader) Read(b []byte) (int, error) {
	n, err := r.r.Read(b)
	if err != nil {
		return n, err
	}
	r.progress.complete(Update{
		Digest:   r.digest,
		Total:    r.total,
		Complete: atomic.AddInt64(r.count, int64(n)),
	})
	return n, nil
}
