package executable

import (
	"context"
	"io"
	"os"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/octohelm/crkit/pkg/ocitar"
	testingx "github.com/octohelm/x/testing"
)

func TestPacker(t *testing.T) {
	p := &Packer{}

	t.Run("should pack as index", func(t *testing.T) {
		amd64Bin, err := PlatformedBinary("linux/amd64", func() (io.ReadCloser, error) {
			return os.Open("../../target/crkit_linux_amd64/crkit")
		})
		testingx.Expect(t, err, testingx.BeNil[error]())
		arm64Bin, err := PlatformedBinary("linux/arm64", func() (io.ReadCloser, error) {
			return os.Open("../../target/crkit_linux_arm64/crkit")
		})
		testingx.Expect(t, err, testingx.BeNil[error]())

		idx, err := p.PackAsIndexOfOciTar(
			context.Background(),
			[]LayerWithPlatform{
				amd64Bin,
				arm64Bin,
			},
			WithImageName("docker.io/x/crkit:v0"),
		)
		testingx.Expect(t, err, testingx.BeNil[error]())

		err = writeAsOciTar("../../target/bin.oci.tar", idx)
		testingx.Expect(t, err, testingx.BeNil[error]())
	})
}

func writeAsOciTar(filename string, idx v1.ImageIndex) error {
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return ocitar.Write(f, idx)
}
