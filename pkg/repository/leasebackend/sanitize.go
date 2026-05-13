package leasebackend

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var k8sNameInvalidChars = regexp.MustCompile("[^a-zA-Z0-9-]+")

// SanitizeForK8sName produces a DNS-1123 label-safe string: lowercased,
// non-alphanumeric characters replaced with hyphens, leading/trailing hyphens
// trimmed, and capped at 63 characters. Unlike utils.SanitizeName, it does not
// strip file extensions (dots are treated as invalid characters and replaced).
// When truncation is needed, it appends a short hash suffix to preserve
// uniqueness of the sanitized pre-truncation value; distinct original inputs
// that normalize to the same sanitized value may still collide.
func SanitizeForK8sName(name string) string {
	original := name
	name = strings.ToLower(name)
	name = k8sNameInvalidChars.ReplaceAllString(name, "-")
	name = strings.TrimLeft(name, "-")
	name = strings.TrimRight(name, "-")
	if len(name) > 63 {
		h := sha256.Sum256([]byte(original))
		suffix := hex.EncodeToString(h[:4]) // 8 hex chars
		// Reserve space for "-" + 8-char hash suffix = 9 chars
		name = strings.TrimRight(name[:63-9], "-") + "-" + suffix
	}
	if name == "" {
		h := sha256.Sum256([]byte(original))
		return hex.EncodeToString(h[:4]) // 8 hex chars, deterministic fallback
	}
	return name
}
