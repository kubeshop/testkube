package resourcepattern

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/common"
)

type resourcePattern struct {
	pattern string
	regex   *regexp.Regexp
	fields  []string
}

type Metadata struct {
	Name    string
	Generic map[string]string
}

type Pattern interface {
	Parse(name string, metadata map[string]string) (*Metadata, bool)
	Compile(metadata *Metadata) (string, bool)
}

func New(pattern string) (Pattern, error) {
	if pattern == "" {
		pattern = "<name>"
	}
	patternRegex := regexp.QuoteMeta(pattern)
	patternRegex = strings.ReplaceAll(patternRegex, "<", "(?P<")
	patternRegex = strings.ReplaceAll(patternRegex, ">", ">[^/<>]+)")
	regex, err := regexp.Compile("^" + patternRegex + "$")
	if err != nil {
		return nil, errors.Wrap(err, "invalid resource pattern")
	}
	return &resourcePattern{
		pattern: pattern,
		regex:   regex,
		fields: common.MapSlice(regexp.MustCompile("<[^>]+>").FindAllString(pattern, -1), func(t string) string {
			return t[1 : len(t)-1]
		}),
	}, nil
}

func (r *resourcePattern) parse(name string, metadata map[string]string) (*Metadata, bool) {
	match := r.regex.FindStringSubmatch(name)
	if match == nil {
		return nil, false
	}
	generic := make(map[string]string)
	for i, key := range r.regex.SubexpNames() {
		if key == "" {
			continue
		}
		if generic[key] != "" && generic[key] != match[i] {
			// Avoid if duplicated value is not matching
			return nil, false
		}
		if metadata[key] != "" && metadata[key] != match[i] {
			// Avoid if the value is not accepted
			return nil, false
		}
		generic[key] = match[i]
	}
	result := &Metadata{Name: generic["name"], Generic: generic}
	if result.Name == "" {
		return nil, false
	}
	delete(result.Generic, "name")
	return result, true
}

func (r *resourcePattern) Parse(name string, metadata map[string]string) (*Metadata, bool) {
	result, ok := r.parse(name, metadata)
	if !ok {
		return nil, false
	}

	// Avoid circular patterns
	for {
		var nextMetadata *Metadata
		nextMetadata, ok = r.parse(result.Name, result.Generic)
		if !ok || result.Name == nextMetadata.Name {
			return result, true
		}
		result = nextMetadata
	}
}

func (r *resourcePattern) compile(metadata *Metadata) (string, bool) {
	if metadata == nil {
		return "", false
	}

	// Replace data in the pattern
	vals := []string{"<name>", metadata.Name}
	for k := range metadata.Generic {
		vals = append(vals, "<"+k+">", metadata.Generic[k])
	}
	return strings.NewReplacer(vals...).Replace(r.pattern), true
}

func (r *resourcePattern) Compile(metadata *Metadata) (string, bool) {
	name, ok := r.compile(metadata)
	if !ok {
		return "", false
	}

	// Validate if it's possible
	nextMetadata, ok := r.parse(name, metadata.Generic)
	if !ok {
		return "", false
	}

	// Avoid circular patterns
	metadata = nextMetadata
	for {
		nextMetadata, ok = r.parse(metadata.Name, metadata.Generic)
		if !ok {
			return name, true
		}
		var nextName string
		nextName, ok = r.compile(nextMetadata)
		if !ok || nextName == name {
			return name, true
		}
		metadata = nextMetadata
		name = nextName
	}
}
