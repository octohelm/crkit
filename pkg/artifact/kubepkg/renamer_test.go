package kubepkg

import (
	"testing"

	containerregistryname "github.com/google/go-containerregistry/pkg/name"
	testingx "github.com/octohelm/x/testing"
)

func TestRename(t *testing.T) {
	n, err := NewTemplateRenamer(`docker.io/x/{{ if ( hasPrefix .name "artifact-") }}{{ .name }}{{ else }}prefix-{{ .name }}{{ end }}`)
	testingx.Expect(t, err, testingx.BeNil[error]())

	repo0, _ := containerregistryname.NewRepository("docker.io/y/artifact-x")
	testingx.Expect(t, n.Rename(repo0), testingx.Be("docker.io/x/artifact-x"))

	repo1, _ := containerregistryname.NewRepository("docker.io/y/x")
	testingx.Expect(t, n.Rename(repo1), testingx.Be("docker.io/x/prefix-x"))
}
