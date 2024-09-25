package fs_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	testingx "github.com/octohelm/x/testing"
	"github.com/opencontainers/go-digest"
)

func TestBlobStore(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	fs := local.NewFS(tmp)

	s := contentfs.NewBlobStore(fs)

	str := "12345678"

	t.Run("put contents", func(t *testing.T) {
		ctx := context.Background()

		w, err := s.Writer(ctx)
		testingx.Expect(t, err, testingx.Be[error](nil))
		defer w.Close()

		buf := bytes.NewBufferString(str)
		_, _ = io.Copy(w, buf)

		d, err := w.Commit(ctx, manifestv1.Descriptor{})
		testingx.Expect(t, err, testingx.Be[error](nil))
		testingx.Expect(t, d.Size, testingx.Be(int64(len(str))))
		testingx.Expect(t, d.Digest, testingx.Be(digest.FromString(str)))

		t.Run("info", func(t *testing.T) {
			d, err := s.Info(ctx, digest.FromString(str))
			testingx.Expect(t, err, testingx.Be[error](nil))
			testingx.Expect(t, d.Size, testingx.Be(int64(len(str))))
		})

		t.Run("open", func(t *testing.T) {
			r, err := s.Open(ctx, digest.FromString(str))
			testingx.Expect(t, err, testingx.Be[error](nil))
			defer r.Close()

			data, _ := io.ReadAll(r)
			testingx.Expect(t, string(data), testingx.Be(str))
		})
	})
}
