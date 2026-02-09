package tar

import (
	"io"
	"os"
	"path"
	"testing"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/partial"
	"github.com/octohelm/crkit/pkg/oci/random"
)

func TestOciTar(t *testing.T) {
	d := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(d)
	})

	t.Run("处理镜像索引", func(t *testing.T) {
		imageCount := 2
		layerCountPerImage := 5

		imageIndex := MustValue(t, func() (oci.Index, error) {
			return random.Index(10, layerCountPerImage, imageCount)
		})

		t.Run("写入tar文件", func(t *testing.T) {
			filename := path.Join(d, "x.tar")

			f := MustValue(t, func() (*os.File, error) {
				return os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
			})

			Then(t, "成功写入索引到 tar",
				ExpectMust(func() error {
					return Write(f, imageIndex)
				}),
			)

			Must(t, func() error {
				return f.Close()
			})

			t.Run("读取tar文件", func(t *testing.T) {
				idx := MustValue(t, func() (oci.Index, error) {
					return Index(func() (io.ReadCloser, error) {
						return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
					})
				})

				images := MustValue(t, func() ([]oci.Image, error) {
					return partial.CollectImages(t.Context(), idx)
				})

				descriptors := MustValue(t, func() ([]ocispecv1.Descriptor, error) {
					return partial.CollectChildDescriptors(t.Context(), idx)
				})

				Then(t, "镜像数量正确",
					Expect(len(images), Equal(imageCount)),
				)

				expectedDescriptors := (layerCountPerImage+1)*imageCount + imageCount
				Then(t, "描述符数量正确",
					Expect(len(descriptors), Equal(expectedDescriptors)),
				)

				t.Run("写入差异tar", func(t *testing.T) {
					filenameDiff := path.Join(d, "x.diff.tar")

					f := MustValue(t, func() (*os.File, error) {
						return os.OpenFile(filenameDiff, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
					})

					Then(t, "成功写入差异 tar",
						ExpectMust(func() error {
							return Write(f, imageIndex, ExcludeImageIndex(t.Context(), idx))
						}),
					)

					Must(t, func() error {
						return f.Close()
					})
				})
			})
		})
	})
}
