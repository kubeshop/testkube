package leasebackend

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var k8sNameInvalidChars = regexp.MustCompile("[^a-z0-9-]+")

// SanitizeForK8sName produces a DNS-1123 label-safe string: lowercased,
// non-alphanumeric characters replaced with hyphens, leading/trailing hyphens
// trimmed, and capped at 63 characters. Unlike utils.SanitizeName, it does not
// strip file extensions (dots are treated as invalid characters and replaced).
// When sanitization modifies the input (beyond lowercasing) or when truncation
// is needed, a hash suffix derived from the original input is appended to
// preserve uniqueness — distinct original inputs that normalize to the same
// sanitized value (e.g. "env_a" vs "env-a") will produce distinct outputs.
func SanitizeForK8sName(name string) string {
	original := name
	name = strings.ToLower(name)
	lowered := name
	name = k8sNameInvalidChars.ReplaceAllString(name, "-")
	name = strings.TrimLeft(name, "-")
	name = strings.TrimRight(name, "-")
	if name == "" {
		h := sha256.Sum256([]byte(original))
		return hex.EncodeToString(h[:4]) // 8 hex chars, deterministic fallback
	}
	// Append a hash when sanitization changed the input (beyond lowercasing)
	// or when truncation is needed, to preserve uniqueness of distinct originals.
	needsHash := name != lowered || len(name) > 63
	if needsHash {
		h := sha256.Sum256([]byte(original))
		suffix := hex.EncodeToString(h[:4]) // 8 hex chars
		if len(name) > 63-9 {
			name = name[:63-9]
		}
		name = strings.TrimRight(name, "-") + "-" + suffix
	}
	return name
}
