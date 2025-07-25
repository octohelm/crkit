package kubepkg

import (
	"path"
	"strings"
	"text/template"

	"github.com/google/go-containerregistry/pkg/name"
)

type Renamer interface {
	Rename(repo name.Repository) string
}

func NewTemplateRenamer(text string) (Renamer, error) {
	tpl := template.New(text).Funcs(template.FuncMap{
		"hasPrefix": strings.HasPrefix,
	})
	t, err := tpl.Parse(text)
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
		"namespace": path.Dir(repo.RepositoryStr()),
		"name":      path.Base(repo.Name()),
	}

	if err := t.Execute(b, ctx); err == nil {
		return strings.TrimSpace(b.String())
	}

	return repo.String()
}
