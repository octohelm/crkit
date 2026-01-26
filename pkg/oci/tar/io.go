package tar

import (
	"io"

	"github.com/octohelm/crkit/pkg/oci/internal"
)

type Opener = internal.Opener

type FileOpener interface {
	Open(filename string) (io.ReadCloser, error)
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r *readCloser) Close() error {
	return r.close()
}
