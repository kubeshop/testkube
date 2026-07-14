package utils

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"k8s.io/apimachinery/pkg/util/validation"
)

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	return false
}

func ContainsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func RemoveDuplicates(s []string) []string {
	m := make(map[string]struct{})
	result := []string{}

	for _, v := range s {
		if _, value := m[v]; !value {
			m[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// RoundDuration rounds duration to default value if no round passed
func RoundDuration(duration time.Duration, to ...time.Duration) time.Duration {
	roundTo := 10 * time.Millisecond
	if len(to) > 0 {
		roundTo = to[0]
	}
	return duration.Round(roundTo)
}

// ReadLongLine reads long line
func ReadLongLine(r *bufio.Reader) (line []byte, err error) {
	var buffer []byte
	var isPrefix bool

	for {
		buffer, isPrefix, err = r.ReadLine()
		line = append(line, buffer...)
		if err != nil {
			break
		}

		if !isPrefix {
			break
		}
	}

	return line, err
}

func RandAlphanum(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		b[i] = letters[nBig.Int64()]
	}
	return string(b)
}

func CheckStringKey(m map[string]any, key string) error {
	if _, ok := m[key]; !ok {
		return errors.New(key + " is missing")
	}
	if _, ok := m[key].(string); !ok {
		return errors.New(key + " is not a string")
	}
	return nil
}

func GetStringKey(m map[string]any, key string) (string, error) {
	if _, ok := m[key]; !ok {
		return "", errors.New(key + " is missing")
	}
	s, ok := m[key].(string)
	if !ok {
		return "", errors.New(key + " is not a string")
	}
	return s, nil
}

// SanitizeName sanitizes test name
func SanitizeName(path string) string {
	path = strings.TrimSuffix(path, filepath.Ext(path))

	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	path = reg.ReplaceAllString(path, "-")
	path = strings.TrimLeft(path, "-")
	path = strings.TrimRight(path, "-")
	path = strings.ToLower(path)

	if len(path) > 63 {
		return path[:63]
	}

	return path
}

// SanitizeLabelValue sanitizes a string so it conforms to Kubernetes label value rules
// (max 63 characters, alphanumeric plus '-', '_' and '.', beginning and ending with an
// alphanumeric character). Invalid characters are replaced with '-'. It returns an empty
// string if the value cannot be made into a valid label value.
func SanitizeLabelValue(value string) string {
	if value == "" {
		return value
	}

	// Replace invalid characters with hyphens
	sanitized := strings.Map(func(r rune) rune {
		if isAllowedLabelRune(r) {
			return r
		}
		return '-'
	}, value)

	// Truncate if too long
	if len(sanitized) > validation.LabelValueMaxLength {
		sanitized = sanitized[:validation.LabelValueMaxLength]
	}

	// Trim non-alphanumeric characters from start and end
	sanitized = strings.TrimLeftFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })
	sanitized = strings.TrimRightFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })

	// Final validation
	if errs := validation.IsValidLabelValue(sanitized); len(errs) > 0 {
		return ""
	}

	return sanitized
}

// isAllowedLabelRune returns true if the rune is allowed in a Kubernetes label value.
func isAllowedLabelRune(r rune) bool {
	return isAlphaNumeric(r) || r == '-' || r == '_' || r == '.'
}

// isAlphaNumeric returns true if the rune is an ASCII alphanumeric character.
func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// EscapeDots escapes dots for MongoDB fields
func EscapeDots(source string) string {
	return strings.ReplaceAll(source, ".", string([]rune{0xFF0E}))
}

// UnescapeDots unescapes dots from MongoDB fields
func UnescapeDots(source string) string {
	return strings.ReplaceAll(source, string([]rune{0xFF0E}), ".")
}

func NewTemplate(name string) *template.Template {
	return template.New(name).Funcs(sprig.FuncMap())
}

// IsBase64Encoded check if string is base84 encoded
func IsBase64Encoded(base64Val string) bool {
	decoded, err := base64.StdEncoding.DecodeString(base64Val)
	if err != nil {
		return false
	}

	encoded := base64.StdEncoding.EncodeToString(decoded)
	return base64Val == encoded
}

// GetEnvVarWithDeprecation returns the value of the environment variable with the given key,
// or the value of the environment variable with the given deprecated key, or the default value
// if neither is set
func GetEnvVarWithDeprecation(key, deprecatedKey, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	if val, ok := os.LookupEnv(deprecatedKey); ok {
		return val
	}
	return defaultVal
}

// TruncateName truncates name to k8s name length
func TruncateName(value string) string {
	if len(value) > 63 {
		return value[:63]
	}

	return value
}
