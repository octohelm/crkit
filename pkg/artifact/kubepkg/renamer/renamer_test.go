package renamer

import (
	"testing"

	. "github.com/octohelm/x/testing/v2"
)

func TestRename(t *testing.T) {
	t.Run("template renamer", func(t *testing.T) {
		re := MustValue(t, func() (Renamer, error) {
			return NewTemplate(`
docker.io/x/
{{ if ( hasPrefix .name "artifact-") }}
	{{ .name }}
{{ else }}
	prefix-{{ .name }}
{{ end }}
`)
		})

		t.Run("rename std", func(t *testing.T) {
			renamed := re.Rename("docker.io/y/x")

			Then(t, "should rename standard image correctly",
				Expect(renamed, Equal("docker.io/x/prefix-x")),
			)
		})

		t.Run("rename prefix", func(t *testing.T) {
			renamed := re.Rename("docker.io/y/artifact-x")

			Then(t, "should rename artifact prefixed image correctly",
				Expect(renamed, Equal("docker.io/x/artifact-x")),
			)
		})
	})
}
