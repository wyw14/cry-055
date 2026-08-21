package redact

import (
	"regexp"
	"strings"
)

var (
	emailPattern = regexp.MustCompile(`([A-Za-z0-9._%+-])[^@\s]*(@[A-Za-z0-9.-]+)`)
	phonePattern = regexp.MustCompile(`\b(1\d{2})\d{4}(\d{4})\b`)
	tokenPattern = regexp.MustCompile(`(?i)(token|secret|password|authorization)\s*[:=]\s*[^\s,;]+`)
)

func Text(value string) string {
	value = emailPattern.ReplaceAllString(value, "$1***$2")
	value = phonePattern.ReplaceAllString(value, "$1****$2")
	value = tokenPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := strings.FieldsFunc(match, func(r rune) bool { return r == ':' || r == '=' })
		if len(parts) == 0 {
			return "[REDACTED]"
		}
		return strings.TrimSpace(parts[0]) + "=[REDACTED]"
	})
	return value
}
