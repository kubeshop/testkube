package leasebackend

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

const (
	// maxK8sNameLen is the maximum length of a DNS-1123 label (Kubernetes name).
	maxK8sNameLen = 63
	// hashSuffixLen is the length of the SHA-256 hash suffix (8 hex chars).
	hashSuffixLen = 8
	// hashSeparatorLen is the total length consumed by the separator ("-") plus the hash.
	hashSeparatorLen = 1 + hashSuffixLen // "-" + 8 hex chars = 9
)

var k8sNameInvalidChars = regexp.MustCompile("[^a-z0-9-]+")

// SanitizeForK8sName produces a DNS-1123 label-safe string: lowercased,
// non-alphanumeric characters replaced with hyphens, leading/trailing hyphens
// trimmed, and capped at 63 characters. Unlike utils.SanitizeName, it does not
// strip file extensions (dots are treated as invalid characters and replaced).
//
// A hash suffix derived from the original input is appended in two cases:
//  1. When sanitization modifies the lowercased input (character replacement or
//     trimming) — e.g. "env_a" vs "env-a" produce distinct outputs.
//  2. When the sanitized name exceeds 63 characters and must be truncated.
//
// Case-only differences are intentionally not hashed: DNS names are
// case-insensitive, so "My-Cluster" and "my-cluster" correctly collapse to the
// same lease name.
func SanitizeForK8sName(name string) string {
	original := name
	name = strings.ToLower(name)
	lowered := name
	name = k8sNameInvalidChars.ReplaceAllString(name, "-")
	name = strings.TrimLeft(name, "-")
	name = strings.TrimRight(name, "-")
	if name == "" {
		h := sha256.Sum256([]byte(original))
		return hex.EncodeToString(h[:hashSuffixLen/2]) // 8 hex chars, deterministic fallback
	}
	// Append a hash when sanitization changed the input (beyond lowercasing)
	// or when truncation is needed, to preserve uniqueness of distinct originals.
	needsHash := name != lowered || len(name) > maxK8sNameLen
	if needsHash {
		h := sha256.Sum256([]byte(original))
		suffix := hex.EncodeToString(h[:hashSuffixLen/2]) // 8 hex chars
		if len(name) > maxK8sNameLen-hashSeparatorLen {
			name = name[:maxK8sNameLen-hashSeparatorLen]
		}
		// TrimRight removes trailing hyphens that would produce "name---suffix".
		// The result is at most maxK8sNameLen chars: up to (maxK8sNameLen - hashSeparatorLen)
		// chars + "-" + hashSuffixLen chars = maxK8sNameLen.
		name = strings.TrimRight(name, "-") + "-" + suffix
	}
	return name
}
