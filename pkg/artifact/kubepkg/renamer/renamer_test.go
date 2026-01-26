package renamer

import (
	"testing"

	"github.com/octohelm/x/testing/bdd"
)

func TestRename(t *testing.T) {
	b := bdd.FromT(t)
	b.Given("template renamer", func(b bdd.T) {
		re := bdd.Must(NewTemplate(`
docker.io/x/
{{ if ( hasPrefix .name "artifact-") }}
	{{ .name }}
{{ else }}
	prefix-{{ .name }}
{{ end }}
`))

		b.When("rename std", func(b bdd.T) {
			renamed := re.Rename("docker.io/y/x")
			b.Then("success as expect", bdd.Equal("docker.io/x/prefix-x", renamed))
		})

		b.When("rename prefix", func(b bdd.T) {
			renamed := re.Rename("docker.io/y/artifact-x")
			b.Then("success as expect", bdd.Equal("docker.io/x/artifact-x", renamed))
		})
	})
}
