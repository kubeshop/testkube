package crd

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates
var f embed.FS

// Template is crd template type
type Template string

const (
	// TemplateExecutor is executor crd template
	TemplateExecutor Template = "executor"
)

// ExecuteTemplate executes crd template
func ExecuteTemplate(tmpl Template, data any) (string, error) {
	t, err := template.ParseFS(f, fmt.Sprintf("templates/%s.tmpl", tmpl))
	if err != nil {
		return "", err
	}

	buffer := bytes.NewBuffer([]byte{})
	err = t.Execute(buffer, data)
	return buffer.String(), err
}
