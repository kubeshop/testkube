package export

import (
	"regexp"
	"strings"
)

var sensitiveHelmSetKey = regexp.MustCompile(`(?i)(password|dsn|license|credential|token|secretkey)`)

// formatHelmSetArg builds a Helm --set argument with value characters escaped per strvals rules.
func formatHelmSetArg(key, value string) string {
	return key + "=" + escapeHelmSetValue(value)
}

func escapeHelmSetValue(value string) string {
	if value == "" {
		return value
	}
	var b strings.Builder
	b.Grow(len(value) + 8)
	for _, r := range value {
		switch r {
		case '\\', ',':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func redactHelmSetValueForLog(key, value string) string {
	if sensitiveHelmSetKey.MatchString(key) {
		return "[REDACTED]"
	}
	return value
}

func redactHelmArgsForLog(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	for i := 0; i < len(out); i++ {
		if out[i] != "--set" || i+1 >= len(out) {
			continue
		}
		setArg := out[i+1]
		key, value, ok := strings.Cut(setArg, "=")
		if !ok {
			continue
		}
		out[i+1] = key + "=" + redactHelmSetValueForLog(key, value)
		i++
	}
	return out
}
