package executable

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/octohelm/exp/xiter"
	"github.com/octohelm/x/testing/bdd"

	ocitar "github.com/octohelm/crkit/pkg/oci/tar"
)

func TestPacker(t *testing.T) {
	b := bdd.FromT(t)

	b.Given("given two binaries", func(b bdd.T) {
		amd64Bin := bdd.Must(Platformed("linux/amd64", func(ctx context.Context) (io.ReadCloser, error) {
			return os.Open("testdata/x.sh")
		}))

		arm64Bin := bdd.Must(Platformed("linux/arm64", func(ctx context.Context) (io.ReadCloser, error) {
			return os.Open("testdata/x.sh")
		}))

		p := &Packer{}

		b.When("pack", func(b bdd.T) {
			idx, err := p.Pack(
				b.Context(),
				xiter.Of(
					amd64Bin,
					arm64Bin,
				),
			)
			b.Then("success", bdd.NoError(err))

			i := bdd.Must(idx.Value(b.Context()))

			b.Then("2 manifests", bdd.Equal(2, len(i.Manifests)))
		})

		b.When("pack as index", func(b bdd.T) {
			idx, err := p.PackAsIndex(
				b.Context(),
				xiter.Of(
					amd64Bin,
					arm64Bin,
				),
				WithImageName("x/bin:latest"),
			)
			b.Then("success", bdd.NoError(err))

			b.Then("write oci tar success",
				bdd.NoError(ocitar.WriteFile("./target/bin.oci.tar", idx)),
			)
		})
	})
}
