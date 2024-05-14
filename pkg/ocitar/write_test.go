package ocitar

import (
	"os"
	"path/filepath"
	"testing"

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

	t.Run("should write", func(t *testing.T) {
		filename := filepath.Join(d, "x.tar")

		f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
		testingx.Expect(t, err, testingx.BeNil[error]())
		defer f.Close()

		err = Write(f, index)
		testingx.Expect(t, err, testingx.BeNil[error]())
	})
}
