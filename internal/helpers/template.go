package helpers

import (
	"html/template"
	"strings"
)

// TemplateFuncs returns a map of template helper functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"userInitials": UserInitials,
		"hasPrefix":    strings.HasPrefix,
		"formatDate":   FormatDate,
		"derefStr":     DerefStr,
		"add":          func(a, b int) int { return a + b },
	}
}

// UserInitials extracts initials from full name
func UserInitials(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		return strings.ToUpper(string(parts[0][0]))
	}
	return strings.ToUpper(string(parts[0][0]) + string(parts[len(parts)-1][0]))
}

// FormatDate formats a date string
func FormatDate(date string) string {
	return date
}

// DerefStr dereferences a string pointer, returning empty string if nil.
// If the value looks like an ISO datetime, it extracts just the date part.
func DerefStr(s *string) string {
	if s == nil {
		return ""
	}
	v := *s
	// Strip time portion from datetime strings like "2025-06-15T00:00:00Z"
	if len(v) > 10 && v[10] == 'T' {
		return v[:10]
	}
	return v
}

// Breadcrumb represents a breadcrumb item
type Breadcrumb struct {
	Title string
	URL   string
}

// BuildBreadcrumbs creates breadcrumb trail
func BuildBreadcrumbs(items ...Breadcrumb) []Breadcrumb {
	return items
}
