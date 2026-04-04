package sanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"clean string", "hello world", "hello world"},
		{"newline injection", "user\nINFO injected log", "user INFO injected log"},
		{"carriage return", "value\r\ninjected", "value  injected"},
		{"tab character", "before\tafter", "before after"},
		{"null byte", "before\x00after", "before after"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, LogValue(tt.input))
		})
	}
}

func TestLogValue_Truncation(t *testing.T) {
	long := strings.Repeat("a", 2000)
	result := LogValue(long)
	assert.Contains(t, result, "...[truncated]")
	assert.Less(t, len(result), 1100) // 1024 chars + truncation message
}

func TestCSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMod  bool
		contains string
	}{
		{"clean CSS", "body { color: red; }", false, "body"},
		{"blocks @import", "@import url('evil.css');", true, "/* blocked */"},
		{"blocks url()", "div { background: url('http://evil.com'); }", true, "/* blocked */"},
		{"blocks expression", "div { width: expression(alert(1)); }", true, "/* blocked */"},
		{"blocks javascript:", "div { background: javascript:alert(1); }", true, "/* blocked */"},
		{"blocks !important", "div { color: red !important; }", true, "/* blocked */"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified := CSS(tt.input)
			assert.Equal(t, tt.wantMod, modified)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestCSS_MaxSize(t *testing.T) {
	huge := strings.Repeat("a", 65*1024)
	result, modified := CSS(huge)
	assert.True(t, modified)
	assert.Empty(t, result)
}

func TestIsSVGElementBlocked(t *testing.T) {
	assert.True(t, IsSVGElementBlocked("script"))
	assert.True(t, IsSVGElementBlocked("SCRIPT"))
	assert.True(t, IsSVGElementBlocked("iframe"))
	assert.True(t, IsSVGElementBlocked("foreignObject"))
	assert.False(t, IsSVGElementBlocked("rect"))
	assert.False(t, IsSVGElementBlocked("path"))
}

func TestIsSVGAttrBlocked(t *testing.T) {
	assert.True(t, IsSVGAttrBlocked("onclick", "alert(1)"))
	assert.True(t, IsSVGAttrBlocked("onerror", "alert(1)"))
	assert.True(t, IsSVGAttrBlocked("onload", "alert(1)"))
	assert.True(t, IsSVGAttrBlocked("href", "javascript:alert(1)"))
	assert.True(t, IsSVGAttrBlocked("src", "data:text/html,<script>"))
	assert.False(t, IsSVGAttrBlocked("fill", "red"))
	assert.False(t, IsSVGAttrBlocked("href", "https://example.com"))
}

func TestFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean name", "report.pdf", "report.pdf"},
		{"path traversal", "../../etc/passwd", "etcpasswd"},
		{"backslash", "..\\..\\windows\\system32", "windowssystem32"},
		{"null byte", "file\x00.pdf", "file.pdf"},
		{"empty becomes unnamed", "", "unnamed"},
		{"spaces only becomes unnamed", "   ", "unnamed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FileName(tt.input))
		})
	}
}
