package remote

import (
	"testing"

	"github.com/distribution/reference"

	"github.com/octohelm/x/testing/bdd"
)

func TestRegistry(t *testing.T) {
	b := bdd.FromT(t)

	b.Given("registry hosts", func(b bdd.T) {
		p := RegistryHosts{
			"gcr.io": {
				Server: "https://gcr.io",
			},
		}

		b.When("resolve non-domain name", func(b bdd.T) {
			n, rh, err := p.Resolve(b.Context(), bdd.Must(reference.WithName("nginx")))

			b.Then("found",
				bdd.NoError(err),
				bdd.Equal("https://registry-1.docker.io", rh.Server),
				bdd.Equal("library/nginx", n.Name()),
			)
		})

		b.When("resolve docker name", func(b bdd.T) {
			n, rh, err := p.Resolve(b.Context(), bdd.Must(reference.WithName("docker.io/x/nginx")))

			b.Then("found",
				bdd.NoError(err),
				bdd.Equal("https://registry-1.docker.io", rh.Server),
				bdd.Equal("x/nginx", n.Name()),
			)
		})

		b.When("resolve gcr name", func(b bdd.T) {
			n, rh, err := p.Resolve(b.Context(), bdd.Must(reference.WithName("gcr.io/x/nginx")))

			b.Then("found",
				bdd.NoError(err),
				bdd.Equal("https://gcr.io", rh.Server),
				bdd.Equal("x/nginx", n.Name()),
			)
		})
	})
}
