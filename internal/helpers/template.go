package helpers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
)

// TemplateFuncs returns a map of template helper functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"userInitials": UserInitials,
		"hasPrefix":    strings.HasPrefix,
		"formatDate":   FormatDate,
		"derefStr":     DerefStr,
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
		"mul":          func(a, b int) int { return a * b },
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"sanitizeField": func(name string) string {
			name = strings.ReplaceAll(name, " ", "_")
			name = strings.ReplaceAll(name, "/", "_")
			return strings.ToLower(name)
		},
		"mapGet": func(m map[string]string, key string) string {
			return m[key]
		},
		"addressLabel": func(data map[string]string, columns []string) string {
			// Build a label from the first 2 column values
			var parts []string
			for _, col := range columns {
				v := strings.TrimSpace(data[col])
				if v != "" {
					parts = append(parts, v)
					if len(parts) >= 2 {
						break
					}
				}
			}
			if len(parts) == 0 {
				return "(unnamed)"
			}
			return strings.Join(parts, " - ")
		},
		"toJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"join": func(strs []string, sep string) string {
			return strings.Join(strs, sep)
		},
		"formatINR": func(amount float64) string {
			// Format number with Indian comma grouping (e.g., 1,23,456.00)
			isNeg := amount < 0
			amount = math.Abs(amount)
			whole := int64(amount)
			decimal := int64(math.Round((amount - float64(whole)) * 100))

			s := fmt.Sprintf("%d", whole)
			// Indian grouping: last 3 digits, then groups of 2
			if len(s) > 3 {
				result := s[len(s)-3:]
				s = s[:len(s)-3]
				for len(s) > 2 {
					result = s[len(s)-2:] + "," + result
					s = s[:len(s)-2]
				}
				if len(s) > 0 {
					result = s + "," + result
				}
				s = result
			}

			prefix := ""
			if isNeg {
				prefix = "-"
			}
			return fmt.Sprintf("%s%s.%02d", prefix, s, decimal)
		},
		"numberToWords": func(amount float64) string {
			return NumberToIndianWords(amount)
		},
		"derefFloat": func(f *float64) float64 {
			if f == nil {
				return 0
			}
			return *f
		},
		"intToStr": func(i int) string {
			return fmt.Sprintf("%d", i)
		},
		"timeAgo": TimeAgo,
		"containsInt": func(slice []int, val int) bool {
			for _, v := range slice {
				if v == val {
					return true
				}
			}
			return false
		},
		"eq_str": func(a, b string) bool { return a == b },
		"vehiclesJSON": func(v interface{}) template.HTMLAttr {
			b, _ := json.Marshal(v)
			return template.HTMLAttr(b)
		},
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

// TimeAgo returns a human-readable relative time string
func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2, 2006")
	}
}
