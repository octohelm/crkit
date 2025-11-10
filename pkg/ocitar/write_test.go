package ocitar

import (
	"io"
	"os"
	"path"
	"testing"

	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/random"

	testingx "github.com/octohelm/x/testing"
)

func TestOciTar(t *testing.T) {
	d := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(d)
	})

	index, err := random.Index(10, 5, 2)
	testingx.Expect(t, err, testingx.BeNil[error]())

	expectImages, err := partial.FindImages(index, func(desc containerregistryv1.Descriptor) bool {
		return true
	})
	testingx.Expect(t, err, testingx.BeNil[error]())

	filename := path.Join(d, "x.tar")

	t.Run("should write", func(t *testing.T) {
		f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
		testingx.Expect(t, err, testingx.BeNil[error]())

		err = Write(f, index)
		testingx.Expect(t, err, testingx.BeNil[error]())
		_ = f.Close()

		t.Run("then should read", func(t *testing.T) {
			idx, err := Index(func() (io.ReadCloser, error) {
				return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
			})
			testingx.Expect(t, err, testingx.BeNil[error]())

			images, err := partial.FindImages(idx, func(desc containerregistryv1.Descriptor) bool {
				return true
			})
			testingx.Expect(t, err, testingx.BeNil[error]())
			testingx.Expect(t, len(images), testingx.Be(len(expectImages)))

			t.Run("should write diff", func(t *testing.T) {
				filenameDiff := path.Join(d, "x.diff.tar")

				f, err := os.OpenFile(filenameDiff, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
				testingx.Expect(t, err, testingx.BeNil[error]())

				err = Write(f, index, ExcludeImageIndex(idx))
				testingx.Expect(t, err, testingx.BeNil[error]())
				_ = f.Close()
			})
		})
	})
}
