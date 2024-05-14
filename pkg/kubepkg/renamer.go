package kubepkg

import (
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-containerregistry/pkg/name"
)

type Renamer interface {
	Rename(repo name.Repository) string
}

func NewTemplateRenamer(text string) (Renamer, error) {
	t, err := template.New(text).Parse(text)
	if err != nil {
		return nil, err
	}

	return &templateRenamer{
		Template: t,
	}, nil
}

type templateRenamer struct {
	*template.Template
}

func (t *templateRenamer) Rename(repo name.Repository) string {
	b := &strings.Builder{}

	ctx := map[string]any{
		"registry":  repo.RegistryStr(),
		"namespace": filepath.Dir(repo.RepositoryStr()),
		"name":      filepath.Base(repo.Name()),
	}

	if err := t.Execute(b, ctx); err == nil {
		return b.String()
	}

	return repo.String()
}
