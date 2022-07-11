package docgen

import (
	"embed"
	"html/template"
)

//go:embed templates
var templates embed.FS

//go:embed templates/main.css
var css string

//go:embed templates/main.js
var js string

var Template = template.Must(template.New("").ParseFS(templates, "templates/main.tmpl", "templates/*.tmpl")).Lookup("main.tmpl")

type TemplateArguments struct {
	TableOfContents template.HTML
	Breadcrumbs     template.HTML
	Main            template.HTML
}
