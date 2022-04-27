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
	if keepLeft > len(in) {
		return strings.Repeat("*", len(in))
	}
	if keepRight > len(in) {
		return strings.Repeat("*", len(in))
	}
	return in[:keepLeft] + strings.Repeat("*", len(in)-keepLeft-keepRight) + in[len(in)-keepRight:]
}
