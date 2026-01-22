package httphandler

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// TemplateFuncs returns the custom template functions for HTML templates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// Time formatting
		"formatTime":      formatTime,
		"formatDate":      formatDate,
		"formatDateTime":  formatDateTime,
		"formatDateInput": formatDateInput,
		"timeAgo":         timeAgo,

		// String helpers
		"truncate":  truncate,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.Title, //nolint:staticcheck // Simple title case is fine for display
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace":   strings.ReplaceAll,
		"trimSpace": strings.TrimSpace,
		"join":      strings.Join,
		"split":     strings.Split,
		"pluralize": pluralize,
		"initials":  initials,

		// Conditional helpers
		"eq":       eq,
		"ne":       ne,
		"lt":       lt,
		"le":       le,
		"gt":       gt,
		"ge":       ge,
		"and":      and,
		"or":       or,
		"not":      not,
		"default":  defaultValue,
		"coalesce": coalesce,
		"ternary":  ternary,

		// Collection helpers
		"first": first,
		"last":  last,
		"slice": sliceFunc,
		"len":   length,
		"empty": empty,
		"seq":   seq,
		"dict":  dict,
		"list":  list,

		// HTML helpers
		"safeHTML":       safeHTML,
		"safeURL":        safeURL,
		"safeCSS":        safeCSS,
		"safeJS":         safeJS,
		"attr":           attr,
		"renderMarkdown": renderMarkdown,

		// Math helpers
		"add": add,
		"sub": sub,
		"mul": mul,
		"div": div,
		"mod": mod,
	}
}

// Time formatting functions

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04")
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

func formatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006 15:04")
}

func formatDateInput(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// Time-related constants for timeAgo function.
const (
	hoursPerDay  = 24
	daysPerWeek  = 7
	ellipsisSize = 3
)

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < hoursPerDay*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < daysPerWeek*hoursPerDay*time.Hour:
		days := int(diff.Hours() / hoursPerDay)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// String helpers

// truncate truncates a string to n characters, adding "..." if truncated.
// Arguments are (n int, s string) to work with template pipes: {{.Title | truncate 30}}
func truncate(n int, s string) string {
	if len(s) <= n {
		return s
	}
	if n <= ellipsisSize {
		return s[:n]
	}
	return s[:n-ellipsisSize] + "..."
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func initials(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		if len(parts[0]) > 0 {
			return strings.ToUpper(string(parts[0][0]))
		}
		return ""
	}
	result := ""
	if len(parts[0]) > 0 {
		result += string(parts[0][0])
	}
	if len(parts[len(parts)-1]) > 0 {
		result += string(parts[len(parts)-1][0])
	}
	return strings.ToUpper(result)
}

// Conditional helpers

func eq(a, b any) bool {
	return a == b
}

func ne(a, b any) bool {
	return a != b
}

func lt(a, b int) bool {
	return a < b
}

func le(a, b int) bool {
	return a <= b
}

func gt(a, b int) bool {
	return a > b
}

func ge(a, b int) bool {
	return a >= b
}

func and(a, b bool) bool {
	return a && b
}

func or(a, b bool) bool {
	return a || b
}

func not(a bool) bool {
	return !a
}

func defaultValue(def, val any) any {
	if val == nil || val == "" || val == 0 || val == false {
		return def
	}
	return val
}

func coalesce(values ...any) any {
	for _, v := range values {
		if v != nil && v != "" && v != 0 && v != false {
			return v
		}
	}
	if len(values) > 0 {
		return values[len(values)-1]
	}
	return nil
}

func ternary(condition bool, ifTrue, ifFalse any) any {
	if condition {
		return ifTrue
	}
	return ifFalse
}

// Collection helpers

func first(list []any) any {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

func last(list []any) any {
	if len(list) == 0 {
		return nil
	}
	return list[len(list)-1]
}

func sliceFunc(list []any, start, end int) []any {
	if start < 0 {
		start = 0
	}
	if end > len(list) {
		end = len(list)
	}
	if start > end {
		return nil
	}
	return list[start:end]
}

func length(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		return len(val)
	case []any:
		return len(val)
	case map[string]any:
		return len(val)
	// Common slice types used in templates
	case []string:
		return len(val)
	case []int:
		return len(val)
	case []MessageTagData:
		return len(val)
	case []MessageReactionData:
		return len(val)
	case []MessageViewData:
		return len(val)
	case []ChatViewData:
		return len(val)
	case []ParticipantViewData:
		return len(val)
	case []TaskCardViewData:
		return len(val)
	case []ActivityViewData:
		return len(val)
	case []SelectOption:
		return len(val)
	case []MemberViewData:
		return len(val)
	case []NotificationViewData:
		return len(val)
	default:
		return 0
	}
}

func empty(v any) bool {
	return length(v) == 0
}

func seq(start, end int) []int {
	if end < start {
		return nil
	}
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}

func dict(pairs ...any) map[string]any {
	result := make(map[string]any)
	for i := 0; i < len(pairs)-1; i += 2 {
		key, ok := pairs[i].(string)
		if ok {
			result[key] = pairs[i+1]
		}
	}
	return result
}

func list(items ...any) []any {
	return items
}

// HTML helpers

func safeHTML(s string) template.HTML {
	return template.HTML(s) //nolint:gosec // Intentional for trusted content
}

func safeURL(s string) template.URL {
	return template.URL(s) //nolint:gosec // Intentional for trusted URLs
}

func safeCSS(s string) template.CSS {
	return template.CSS(s) //nolint:gosec // Intentional for trusted CSS
}

func safeJS(s string) template.JS {
	return template.JS(s) //nolint:gosec // Intentional for trusted JS
}

func attr(name, value string) template.HTMLAttr {
	escaped := template.HTMLEscapeString(value)
	//nolint:gosec // Safe because we escape the value
	return template.HTMLAttr(fmt.Sprintf(`%s="%s"`, name, escaped))
}

// Math helpers

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func div(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}

func mod(a, b int) int {
	if b == 0 {
		return 0
	}
	return a % b
}

// renderMarkdown converts markdown-like content to basic HTML.
// This is a simple implementation that handles basic formatting.
// For production, consider using a proper markdown library.
func renderMarkdown(s string) string {
	if s == "" {
		return ""
	}

	// Escape HTML first to prevent XSS
	escaped := template.HTMLEscapeString(s)

	// Simple replacements for basic markdown
	// Bold: **text** or __text__
	result := escaped

	// Convert newlines to <br>
	result = strings.ReplaceAll(result, "\n", "<br>")

	// Wrap in paragraph
	return "<p>" + result + "</p>"
}
