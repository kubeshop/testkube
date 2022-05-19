package text

import "strings"

const (
	defaultLeftCharsShow  = 2
	defaultRightCharsShow = 2
)

// Obfuscate string leave default characters count on left and right
func Obfuscate(in string) string {
	return ObfuscateLR(in, defaultLeftCharsShow, defaultRightCharsShow)
}

func ObfuscateLR(in string, keepLeft, keepRight int) (out string) {
	if len(in) <= 0 {
		return ""
	}
	if keepLeft > len(in) {
		return strings.Repeat("*", len(in))
	}
	if keepRight > len(in) {
		return strings.Repeat("*", len(in))
	}

	if keepLeft+keepRight > len(in) {
		return strings.Repeat("*", len(in))
	}

	repeatCount := len(in) - keepLeft - keepRight
	if repeatCount > 0 {
		return in[:keepLeft] + strings.Repeat("*", repeatCount) + in[len(in)-keepRight:]
	}

	return "***"
}
