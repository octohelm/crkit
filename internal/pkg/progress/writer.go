package progress

import (
	"context"
	"io"
	"iter"
	"sync"
	"sync/atomic"
	"time"
)

func New(w io.Writer) *Writer {
	return &Writer{w: w, close: make(chan struct{})}
}

type Writer struct {
	w       io.Writer
	current atomic.Int64

	close chan struct{}
	once  sync.Once
}

func (pw *Writer) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if err == nil {
		pw.current.Add(int64(n))
	}
	return n, err
}

func (pw *Writer) Close() error {
	pw.once.Do(func() {
		close(pw.close)
	})
	return nil
}

func (pw *Writer) Observe(ctx context.Context) iter.Seq[int64] {
	return func(yield func(int64) bool) {
		ticker := time.NewTicker(3 * time.Second)

		for {
			select {
			case <-pw.close:
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !yield(pw.current.Load()) {
					return
				}
			}
		}
	}
}
