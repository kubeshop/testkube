package declarative

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
)

const (
	// frontmatterDelimiter opens and closes the YAML frontmatter of a SKILL.md.
	frontmatterDelimiter = "---"
	// byteOrderMark is what some editors put at the start of a UTF-8 file.
	byteOrderMark = "\uFEFF"
)

// SkillMarkdown is a SKILL.md as the Agent Skills spec defines it: its
// frontmatter, and its Markdown body.
type SkillMarkdown struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Metadata      map[string]string
	// AllowedTools is space separated, as the spec writes it.
	AllowedTools string
	Body         string
}

// skillFrontmatter is the frontmatter of a SKILL.md. Keys the Agent Skills
// spec does not define are ignored, since the same file often serves other
// agent tools, which add their own.
type skillFrontmatter struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	License       string          `json:"license"`
	Compatibility string          `json:"compatibility"`
	Metadata      map[string]any  `json:"metadata"`
	AllowedTools  json.RawMessage `json:"allowed-tools"`
}

// ParseSkillMarkdown reads a SKILL.md: YAML frontmatter between two "---"
// lines, then the Markdown body. Metadata values must be strings, as the spec
// says, and allowed-tools is a space-separated string.
func ParseSkillMarkdown(data string) (SkillMarkdown, error) {
	data = strings.TrimPrefix(data, byteOrderMark)
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.SplitAfter(data, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != frontmatterDelimiter {
		return SkillMarkdown{}, errors.New("a SKILL.md starts with YAML frontmatter between two --- lines")
	}
	end := slices.IndexFunc(lines[1:], func(line string) bool { return strings.TrimSpace(line) == frontmatterDelimiter })
	if end < 0 {
		return SkillMarkdown{}, errors.New("the SKILL.md frontmatter has no closing --- line")
	}
	end++
	var front skillFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "")), &front); err != nil {
		return SkillMarkdown{}, fmt.Errorf("the SKILL.md frontmatter is not valid: %w", err)
	}
	md := SkillMarkdown{
		Name:          strings.TrimSpace(front.Name),
		Description:   strings.TrimSpace(front.Description),
		License:       strings.TrimSpace(front.License),
		Compatibility: strings.TrimSpace(front.Compatibility),
		Body:          strings.TrimLeft(strings.Join(lines[end+1:], ""), "\n"),
	}
	if len(front.Metadata) > 0 {
		md.Metadata = make(map[string]string, len(front.Metadata))
		for _, key := range sortedKeys(front.Metadata) {
			value, ok := front.Metadata[key].(string)
			if !ok {
				return SkillMarkdown{}, fmt.Errorf("the SKILL.md metadata %q is not a string; quote it, such as version: \"1.0\"", key)
			}
			md.Metadata[key] = value
		}
	}
	tools, err := parseAllowedTools(front.AllowedTools)
	if err != nil {
		return SkillMarkdown{}, err
	}
	md.AllowedTools = tools
	return md, nil
}

func parseAllowedTools(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.Join(strings.Fields(text), " "), nil
	}
	return "", errors.New("the SKILL.md allowed-tools is not a space-separated string of tools")
}

// SkillSourceReader returns the value of the ConfigMap key ref selects, in the
// namespace of the resource being resolved.
type SkillSourceReader func(ref aiv1.ConfigMapKeyReference) (string, error)

// ErrInvalidSkillSource marks an error of ResolveSkillSources that a change
// to the resource or its SKILL.md fixes, rather than a retry. Every other
// error comes from the SkillSourceReader.
var ErrInvalidSkillSource = errors.New("invalid skillFrom")

// ResolveSkillSources replaces every skillFrom of obj (a Skill's, and those
// of an AI Agent's own Skills) with the SKILL.md it selects, read with read,
// so obj can be normalized and sent. The SKILL.md must name the Skill as obj
// does: by metadata.name for a Skill, or by its name in spec.skills. An AI
// Agent's own Skills take the description, metadata and body; the other keys
// of the frontmatter are kept only by a Skill. Objects without skillFrom are
// left alone. GitOps runners call it; testkube apply takes Skills inline.
func ResolveSkillSources(obj Object, read SkillSourceReader) error {
	var errs field.ErrorList
	switch o := obj.(type) {
	case *aiv1.Skill:
		if o.Spec.SkillFrom == nil {
			return nil
		}
		if errs := forbidWithSkillFrom(field.NewPath("spec"), skillWritten(&o.Spec)); len(errs) > 0 {
			return fmt.Errorf("%w: %w", ErrInvalidSkillSource, errs.ToAggregate())
		}
		path := field.NewPath("spec", "skillFrom")
		md, err := readSkillSource(o.Spec.SkillFrom, path, read)
		if err != nil {
			return err
		}
		switch {
		case md.Name == "":
			errs = append(errs, field.Required(path, fmt.Sprintf("the SKILL.md needs a name: %s", o.Name)))
		case md.Name != o.Name:
			errs = append(errs, field.Invalid(path, md.Name, fmt.Sprintf("the SKILL.md must be named as the Skill is: %s", o.Name)))
		}
		if len(errs) > 0 {
			break
		}
		o.Spec = aiv1.SkillSpec{
			Description:   md.Description,
			License:       md.License,
			Compatibility: md.Compatibility,
			Metadata:      md.Metadata,
			AllowedTools:  md.AllowedTools,
			Content:       md.Body,
			Testkube:      o.Spec.Testkube,
		}
	case *aiv1.AIAgent:
		for i := range o.Spec.Skills {
			s := &o.Spec.Skills[i]
			if s.SkillFrom == nil {
				continue
			}
			skillPath := field.NewPath("spec", "skills").Index(i)
			if forbidden := forbidWithSkillFrom(skillPath, ownSkillWritten(s.Description, s.Content, s.Metadata)); len(forbidden) > 0 {
				errs = append(errs, forbidden...)
				continue
			}
			path := skillPath.Child("skillFrom")
			md, err := readSkillSource(s.SkillFrom, path, read)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(s.Name)
			switch {
			case md.Name == "":
				errs = append(errs, field.Required(path, fmt.Sprintf("the SKILL.md needs a name: %s", name)))
				continue
			case md.Name != name:
				errs = append(errs, field.Invalid(path, md.Name, fmt.Sprintf("the SKILL.md must be named as the Skill is: %s", name)))
				continue
			}
			s.Description, s.Metadata, s.Content, s.SkillFrom = md.Description, md.Metadata, md.Body, nil
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrInvalidSkillSource, errs.ToAggregate())
	}
	return nil
}

// readSkillSource checks a skillFrom, reads the SKILL.md it selects and
// parses it.
func readSkillSource(from *aiv1.SkillSource, path *field.Path, read SkillSourceReader) (SkillMarkdown, error) {
	if errs := validateSkillSource(from, path); len(errs) > 0 {
		return SkillMarkdown{}, fmt.Errorf("%w: %w", ErrInvalidSkillSource, errs.ToAggregate())
	}
	ref := *from.ConfigMapKeyRef
	data, err := read(ref)
	if err != nil {
		return SkillMarkdown{}, fmt.Errorf("%s: read key %q of ConfigMap %q: %w", path, ref.Key, ref.Name, err)
	}
	md, err := ParseSkillMarkdown(data)
	if err != nil {
		return SkillMarkdown{}, fmt.Errorf("%w: %s: key %q of ConfigMap %q: %w", ErrInvalidSkillSource, path, ref.Key, ref.Name, err)
	}
	return md, nil
}

// validateSkillSource checks the shape of a skillFrom: a ConfigMap name and a
// key.
func validateSkillSource(from *aiv1.SkillSource, path *field.Path) field.ErrorList {
	refPath := path.Child("configMapKeyRef")
	ref := from.ConfigMapKeyRef
	if ref == nil {
		return field.ErrorList{field.Required(refPath, "the ConfigMap key holding the SKILL.md")}
	}
	var errs field.ErrorList
	if ref.Name == "" {
		errs = append(errs, field.Required(refPath.Child("name"), "the ConfigMap's name"))
	} else {
		for _, msg := range validation.IsDNS1123Subdomain(ref.Name) {
			errs = append(errs, field.Invalid(refPath.Child("name"), ref.Name, msg))
		}
	}
	if ref.Key == "" {
		errs = append(errs, field.Required(refPath.Child("key"), "the key holding the SKILL.md, such as SKILL.md"))
	} else {
		for _, msg := range validation.IsConfigMapKey(ref.Key) {
			errs = append(errs, field.Invalid(refPath.Child("key"), ref.Key, msg))
		}
	}
	return errs
}

// forbidWithSkillFrom refuses every field that set marks as written in the
// manifest: with skillFrom, the SKILL.md gives them.
func forbidWithSkillFrom(path *field.Path, set map[string]bool) field.ErrorList {
	var errs field.ErrorList
	for _, name := range sortedKeys(set) {
		if set[name] {
			errs = append(errs, field.Forbidden(path.Child(name), "comes from the SKILL.md that skillFrom reads"))
		}
	}
	return errs
}

// skillWritten marks the fields of a Skill's spec that a SKILL.md gives.
func skillWritten(spec *aiv1.SkillSpec) map[string]bool {
	return map[string]bool{
		"description":   spec.Description != "",
		"license":       spec.License != "",
		"compatibility": spec.Compatibility != "",
		"metadata":      len(spec.Metadata) > 0,
		"allowed-tools": spec.AllowedTools != "",
		"content":       spec.Content != "",
	}
}

// ownSkillWritten marks the fields of an AI Agent's own Skill that a SKILL.md
// gives.
func ownSkillWritten(description, content string, metadata map[string]string) map[string]bool {
	return map[string]bool{
		"description": description != "",
		"content":     content != "",
		"metadata":    len(metadata) > 0,
	}
}

// unresolvedSkillSource is the error for a skillFrom left in an object being
// normalized: only a GitOps runner, which can read the ConfigMap, resolves it
// before it sends the object.
func unresolvedSkillSource(path *field.Path) *field.Error {
	return field.Forbidden(path, "only a GitOps runner reads a SKILL.md from a ConfigMap in its cluster; testkube apply takes the Skill inline")
}

// SkillSourceConfigMaps returns the names of the ConfigMaps obj reads Skills
// from, sorted and each once.
func SkillSourceConfigMaps(obj Object) []string {
	var names []string
	add := func(from *aiv1.SkillSource) {
		if from != nil && from.ConfigMapKeyRef != nil && from.ConfigMapKeyRef.Name != "" {
			names = append(names, from.ConfigMapKeyRef.Name)
		}
	}
	switch o := obj.(type) {
	case *aiv1.Skill:
		add(o.Spec.SkillFrom)
	case *aiv1.AIAgent:
		for _, s := range o.Spec.Skills {
			add(s.SkillFrom)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}
