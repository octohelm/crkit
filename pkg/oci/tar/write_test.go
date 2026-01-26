package tar

import (
	"io"
	"os"
	"path"
	"testing"

	"github.com/octohelm/x/testing/bdd"

	"github.com/octohelm/crkit/pkg/oci/partial"
	"github.com/octohelm/crkit/pkg/oci/random"
)

func TestOciTar(t *testing.T) {
	d := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(d)
	})

	b := bdd.FromT(t)

	b.Given("index", func(b bdd.T) {
		imageCount := 2
		layerCountPerImage := 5

		imageIndex := bdd.Must(random.Index(10, layerCountPerImage, imageCount))

		b.When("write as tar", func(b bdd.T) {
			filename := path.Join(d, "x.tar")

			f := bdd.Must(os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600))

			b.Then("success written",
				bdd.NoError(Write(f, imageIndex)),
			)

			_ = f.Close()

			b.When("read the tar", func(b bdd.T) {
				idx := bdd.Must(Index(func() (io.ReadCloser, error) {
					return os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
				}))

				images := bdd.Must(partial.CollectImages(t.Context(), idx))
				descriptors := bdd.Must(partial.CollectChildDescriptors(t.Context(), idx))

				b.Then("images should got same",
					bdd.Equal(imageCount, len(images)),
					bdd.Equal((layerCountPerImage /* layers */ +1 /* config */)*imageCount+imageCount, len(descriptors)),
				)

				b.When("write as diffed tar", func(b bdd.T) {
					filenameDiff := path.Join(d, "x.diff.tar")

					f := bdd.Must(os.OpenFile(filenameDiff, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600))

					b.Then("success", bdd.NoError(
						Write(f, imageIndex, ExcludeImageIndex(t.Context(), idx)),
					))

					_ = f.Close()
				})
			})
		})
	})
}
