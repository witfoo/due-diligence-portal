// Package sanitize provides input sanitization and log injection prevention.
// Ported from the WitFoo Analytics project following the WitFoo Way.
// CWE-117: Improper Output Neutralization for Logs
package sanitize

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxLogValueLen = 1024

// LogValue sanitizes a value for safe inclusion in log output.
// It replaces all control characters (< 0x20) with spaces and truncates
// values exceeding 1024 runes. It is O(min(n, 1024)) — bounded work
// regardless of input size.
func LogValue(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	count := 0
	for i := 0; i < len(s) && count < maxLogValueLen; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r < 0x20 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
		i += size
		count++
	}

	result := b.String()
	if count >= maxLogValueLen && utf8.RuneCountInString(s) > maxLogValueLen {
		result += "...[truncated]"
	}
	return result
}

// URL sanitizes a URL path for safe inclusion in log output.
// It removes control characters and truncates long paths.
func URL(path string) string {
	return LogValue(path)
}

// cssBlockedPatterns contains CSS patterns that are blocked for security.
var cssBlockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)@import`),
	regexp.MustCompile(`(?i)url\s*\(`),
	regexp.MustCompile(`(?i)expression\s*\(`),
	regexp.MustCompile(`(?i)behavior\s*:`),
	regexp.MustCompile(`(?i)-moz-binding\s*:`),
	regexp.MustCompile(`(?i)javascript\s*:`),
	regexp.MustCompile(`(?i)data\s*:`),
	regexp.MustCompile(`(?i)vbscript\s*:`),
	regexp.MustCompile(`<!--`),
	regexp.MustCompile(`!important`),
}

const maxCSSSize = 64 * 1024 // 64KB

// CSS sanitizes user-provided CSS by stripping dangerous patterns.
// Returns the sanitized CSS and whether any content was removed.
func CSS(input string) (string, bool) {
	if len(input) > maxCSSSize {
		return "", true
	}

	modified := false
	result := input
	for _, pattern := range cssBlockedPatterns {
		if pattern.MatchString(result) {
			result = pattern.ReplaceAllString(result, "/* blocked */")
			modified = true
		}
	}
	return result, modified
}

// svgBlockedElements contains SVG elements that are blocked for security.
var svgBlockedElements = map[string]bool{
	"script":        true,
	"foreignobject": true,
	"iframe":        true,
	"embed":         true,
	"object":        true,
	"applet":        true,
	"form":          true,
	"input":         true,
	"base":          true,
}

// svgBlockedAttrPrefixes contains attribute prefixes that are blocked in SVGs.
var svgBlockedAttrPrefixes = []string{
	"on", // onclick, onerror, onload, etc.
}

// svgBlockedAttrValues contains URI schemes that are blocked in SVG attributes.
var svgBlockedAttrValues = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*javascript\s*:`),
	regexp.MustCompile(`(?i)^\s*data\s*:`),
	regexp.MustCompile(`(?i)^\s*vbscript\s*:`),
}

// IsSVGElementBlocked returns true if the given SVG element name is blocked.
func IsSVGElementBlocked(name string) bool {
	return svgBlockedElements[strings.ToLower(name)]
}

// IsSVGAttrBlocked returns true if the given SVG attribute name or value is blocked.
func IsSVGAttrBlocked(name, value string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range svgBlockedAttrPrefixes {
		if strings.HasPrefix(lower, prefix) && len(lower) > len(prefix) {
			return true
		}
	}

	if lower == "href" || lower == "src" || lower == "xlink:href" {
		for _, pattern := range svgBlockedAttrValues {
			if pattern.MatchString(value) {
				return true
			}
		}
	}
	return false
}

var (
	filenameUnsafe = regexp.MustCompile(`[^A-Za-z0-9._ -]`)
	dotRuns        = regexp.MustCompile(`\.{2,}`)
	hexColorRe     = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	rgbColorRe     = regexp.MustCompile(`^rgba?\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*(?:,\s*(?:0|1|0?\.\d+)\s*)?\)$`)
)

// FileName sanitizes a filename for safe use, removing any path component and
// neutralizing traversal sequences. The result is restricted to the
// [A-Za-z0-9._-] allowlist with collapsed dot-runs, so it cannot reconstruct ".."
// or contain separators regardless of input.
func FileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/") // treat Windows separators as path separators
	name = filepath.Base(name)                 // drop any directory component
	name = strings.ReplaceAll(name, "\x00", "")
	name = filenameUnsafe.ReplaceAllString(name, "_") // allowlist
	name = dotRuns.ReplaceAllString(name, ".")        // collapse ".." -> "."
	name = strings.Trim(name, ". ")
	if name == "" {
		name = "unnamed"
	}
	return name
}

// IsColor reports whether s is a syntactically valid CSS color (hex or rgb/rgba),
// used to reject untrusted branding color values before they are injected into a
// stylesheet.
func IsColor(s string) bool {
	s = strings.TrimSpace(s)
	return hexColorRe.MatchString(s) || rgbColorRe.MatchString(s)
}
