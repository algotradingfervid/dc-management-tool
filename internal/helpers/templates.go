package helpers

import (
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer implements echo.Renderer with composed template support.
type TemplateRenderer struct {
	templates  map[string]*template.Template
	entryNames map[string]string // maps template key to entry point name
}

// Render implements the echo.Renderer interface.
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "template not found: "+name)
	}
	entry := r.entryNames[name]
	// Set content type if writing to an HTTP response
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	return tmpl.ExecuteTemplate(w, entry, data)
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

		t, parseErr := template.New("").Funcs(funcMap).ParseFiles(files...)
		if parseErr != nil {
			return nil, parseErr
		}
		templates[name] = t
		entryNames[name] = "base"
	}

	// Parse page templates in subdirectories (e.g., pages/projects/*.html, pages/admin/users/*.html)
	pagesDir := filepath.Join(templatesDir, "pages")
	if walkErr := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		rel, relErr := filepath.Rel(pagesDir, path)
		if relErr != nil {
			return relErr
		}
		// Skip top-level files (already handled above)
		if filepath.Dir(rel) == "." {
			return nil
		}
		name := rel // e.g. "admin/users/list.html"
		files := append(append([]string{}, sharedFiles...), path)
		t, parseErr := template.New("").Funcs(funcMap).ParseFiles(files...)
		if parseErr != nil {
			return parseErr
		}
		templates[name] = t
		entryNames[name] = "base"
		return nil
	}); walkErr != nil {
		return nil, walkErr
	}

	// Parse standalone templates
	standaloneFiles, err := filepath.Glob(filepath.Join(templatesDir, "standalone", "*.html"))
	if err != nil {
		return nil, err
	}

	for _, standalone := range standaloneFiles {
		name := filepath.Base(standalone)
		t, parseErr := template.New("").Funcs(funcMap).ParseFiles(standalone)
		if parseErr != nil {
			return nil, parseErr
		}
		templates[name] = t
		// Standalone templates define themselves by filename
		entryNames[name] = name
	}

	// Parse HTMX partial templates (rendered without layout)
	htmxBaseDir := filepath.Join(templatesDir, "htmx")
	if walkErr := filepath.Walk(htmxBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		rel, relErr := filepath.Rel(htmxBaseDir, path)
		if relErr != nil {
			return relErr
		}
		name := "htmx/" + rel
		t, parseErr := template.New("").Funcs(funcMap).ParseFiles(path)
		if parseErr != nil {
			return parseErr
		}
		templates[name] = t
		entryNames[name] = filepath.Base(path)
		return nil
	}); walkErr != nil {
		return nil, walkErr
	}

	return &TemplateRenderer{templates: templates, entryNames: entryNames}, nil
}
