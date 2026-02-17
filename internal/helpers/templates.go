package helpers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin/render"
)

// TemplateRenderer is a custom Gin HTML renderer that supports template composition.
type TemplateRenderer struct {
	templates  map[string]*template.Template
	entryNames map[string]string // maps template key to entry point name
}

// Instance returns the render instance for a given template name and data.
func (r *TemplateRenderer) Instance(name string, data interface{}) render.Render {
	tmpl := r.templates[name]
	entry := r.entryNames[name]
	return &PageRender{
		Template: tmpl,
		Name:     entry,
		Data:     data,
	}
}

// PageRender executes a template by entry point name.
type PageRender struct {
	Template *template.Template
	Name     string
	Data     interface{}
}

func (r *PageRender) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	return r.Template.ExecuteTemplate(w, r.Name, r.Data)
}

func (r *PageRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

// NewTemplateRenderer creates a TemplateRenderer with composed templates.
func NewTemplateRenderer(templatesDir string, funcMap template.FuncMap) (*TemplateRenderer, error) {
	templates := make(map[string]*template.Template)
	entryNames := make(map[string]string)

	// Shared files for layout-based pages
	sharedFiles := []string{
		filepath.Join(templatesDir, "base.html"),
		filepath.Join(templatesDir, "layouts", "main.html"),
		filepath.Join(templatesDir, "partials", "sidebar.html"),
		filepath.Join(templatesDir, "partials", "topbar.html"),
		filepath.Join(templatesDir, "partials", "breadcrumb.html"),
		filepath.Join(templatesDir, "partials", "wizard_steps.html"),
	}

	// Parse page templates (use layout) - top level
	pageFiles, err := filepath.Glob(filepath.Join(templatesDir, "pages", "*.html"))
	if err != nil {
		return nil, err
	}

	for _, page := range pageFiles {
		name := filepath.Base(page)
		files := append(append([]string{}, sharedFiles...), page)

		t, err := template.New("").Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			return nil, err
		}
		templates[name] = t
		entryNames[name] = "base"
	}

	// Parse page templates in subdirectories (e.g., pages/projects/*.html, pages/admin/users/*.html)
	pagesDir := filepath.Join(templatesDir, "pages")
	filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		rel, _ := filepath.Rel(pagesDir, path)
		// Skip top-level files (already handled above)
		if filepath.Dir(rel) == "." {
			return nil
		}
		name := rel // e.g. "admin/users/list.html"
		files := append(append([]string{}, sharedFiles...), path)
		t, err := template.New("").Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			return err
		}
		templates[name] = t
		entryNames[name] = "base"
		return nil
	})

	// Parse standalone templates
	standaloneFiles, err := filepath.Glob(filepath.Join(templatesDir, "standalone", "*.html"))
	if err != nil {
		return nil, err
	}

	for _, standalone := range standaloneFiles {
		name := filepath.Base(standalone)
		t, err := template.New("").Funcs(funcMap).ParseFiles(standalone)
		if err != nil {
			return nil, err
		}
		templates[name] = t
		// Standalone templates define themselves by filename
		entryNames[name] = name
	}

	// Parse HTMX partial templates (rendered without layout)
	htmxBaseDir := filepath.Join(templatesDir, "htmx")
	filepath.Walk(htmxBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		rel, _ := filepath.Rel(htmxBaseDir, path)
		name := "htmx/" + rel
		t, parseErr := template.New("").Funcs(funcMap).ParseFiles(path)
		if parseErr != nil {
			return parseErr
		}
		templates[name] = t
		entryNames[name] = filepath.Base(path)
		return nil
	})

	return &TemplateRenderer{templates: templates, entryNames: entryNames}, nil
}
