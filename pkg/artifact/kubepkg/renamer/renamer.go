package renamer

import (
	"io"
	"strings"
	"text/template"
	"unicode"
)

type Renamer interface {
	Rename(name string) string
}

func NewTemplate(text string) (Renamer, error) {
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

func (t *templateRenamer) Rename(name string) string {
	b := &strings.Builder{}

	namespace := ""
	registry := ""

	namespaceStart := 0
	parts := strings.Split(name, "/")

	n := len(parts)
	if n >= 3 {
		registry = parts[0]
		namespaceStart = 1
	}
	if n >= 2 {
		namespace = strings.Join(parts[namespaceStart:len(parts)-2], "/")
	}

	base := parts[len(parts)-1]

	ctx := map[string]any{
		"registry":  registry,
		"namespace": namespace,
		"name":      base,
	}

	if err := t.Execute(&nonSpaceWriter{Writer: b}, ctx); err == nil {
		return b.String()
	}

	return name
}

type nonSpaceWriter struct {
	Writer io.Writer
}

func (w *nonSpaceWriter) Write(p []byte) (n int, err error) {
	filtered := make([]byte, 0, len(p))

	for _, b := range p {
		if !unicode.IsSpace(rune(b)) {
			filtered = append(filtered, b)
		}
	}

	if len(filtered) > 0 {
		return w.Writer.Write(filtered)
	}

	return 0, nil
}
