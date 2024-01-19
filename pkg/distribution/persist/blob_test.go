package persist

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/octohelm/crkit/pkg/distribution"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"

	testingx "github.com/octohelm/x/testing"
)

func TestBlobService(t *testing.T) {
	bs := &blobService{fsys: filesystem.NewMemFS()}

	t.Run("full ingest", func(t *testing.T) {
		w, err := bs.Create(context.Background())
		testingx.Expect(t, err, testingx.Be[error](nil))

		b := bytes.NewBuffer([]byte(`1234567`))

		digester := digest.SHA256.Digester()

		provisional := distribution.Descriptor{}
		provisional.Size = int64(b.Len())

		_, _ = io.Copy(w, io.TeeReader(b, digester.Hash()))

		provisional.Digest = digester.Digest()

		_, err = w.Commit(context.Background(), provisional)
		testingx.Expect(t, err, testingx.Be[error](nil))
	})
}
