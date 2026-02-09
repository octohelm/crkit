package executable

import (
	"context"
	"io"
	"os"
	"testing"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/exp/xiter"
	"github.com/octohelm/x/cmp"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
	ocitar "github.com/octohelm/crkit/pkg/oci/tar"
)

func TestPacker(t *testing.T) {
	t.Run("给定两个二进制文件", func(t *testing.T) {
		amd64Bin := MustValue(t, func() (oci.Blob, error) {
			return Platformed("linux/amd64", func(ctx context.Context) (io.ReadCloser, error) {
				return os.Open("testdata/x.sh")
			})
		})

		arm64Bin := MustValue(t, func() (oci.Blob, error) {
			return Platformed("linux/arm64", func(ctx context.Context) (io.ReadCloser, error) {
				return os.Open("testdata/x.sh")
			})
		})

		p := &Packer{}

		t.Run("打包", func(t *testing.T) {
			idx, err := p.Pack(
				t.Context(),
				xiter.Of(
					amd64Bin,
					arm64Bin,
				),
			)

			Then(t, "应该成功",
				Expect(err, Be(cmp.Nil[error]())),
			)

			i := MustValue(t, func() (ocispecv1.Index, error) {
				return idx.Value(t.Context())
			})

			Then(t, "应该有2个manifest",
				Expect(len(i.Manifests), Equal(2)),
			)
		})

		t.Run("打包为索引", func(t *testing.T) {
			idx, err := p.PackAsIndex(
				t.Context(),
				xiter.Of(
					amd64Bin,
					arm64Bin,
				),
				WithImageName("x/bin:latest"),
			)

			Then(t, "应该成功",
				Expect(err, Be(cmp.Nil[error]())),
			)

			Then(t, "写入OCI tar应该成功",
				ExpectDo(
					func() error {
						return ocitar.WriteFile("./target/bin.oci.tar", idx)
					},
				),
			)
		})
	})
}
