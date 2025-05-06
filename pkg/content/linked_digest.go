package content

import (
	"context"
	"iter"
	"time"

	"github.com/opencontainers/go-digest"
)

type LinkedDigest struct {
	Digest  digest.Digest
	ModTime time.Time
}

type LinkedDigestIterable interface {
	LinkedDigests(ctx context.Context) iter.Seq2[LinkedDigest, error]
}
