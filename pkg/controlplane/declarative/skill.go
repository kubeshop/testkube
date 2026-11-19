package declarative

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation/field"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
)

const (
	// MaxSkillNameLength, MaxSkillDescriptionLength and MaxSkillContentLength
	// are the Agent Skills spec's limits, the Control Plane's too. The lengths
	// of a description and content are in characters.
	MaxSkillNameLength        = 64
	MaxSkillDescriptionLength = 1024
	MaxSkillContentLength     = 32000
	// MaxSkillCompatibilityLength bounds, in characters, what environment a
	// Skill says it needs, as the Agent Skills spec does.
	MaxSkillCompatibilityLength = 500

	// MaxAlwaysIncludedSkillsLength caps, in characters, the content of all
	// the Skills an owner always includes together: every prompt carries them.
	MaxAlwaysIncludedSkillsLength = 32000
)

// skillNamePattern is the Agent Skills spec's name: lowercase letters, digits
// and single hyphens.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skillFields are the fields of a Skill every kind's Skill type has. from is
// an AI Agent's skillFrom, which no other owner's Skills have.
type skillFields struct {
	name, description *string
	content           string
	metadata          map[string]string
	from              *aiv1.SkillSource
}

// normalizeSkills trims the names and descriptions of an owner's Skills and
// checks them: a spec name unique within the owner, a description and content
// within the spec's limits, and metadata keys. alwaysIncluded is the content
// length of the Skills the owner always includes, which is checked against
// MaxAlwaysIncludedSkillsLength. No content is echoed.
func normalizeSkills(skills []skillFields, alwaysIncluded int, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	seen := map[string]bool{}
	for i, s := range skills {
		skillPath := path.Index(i)
		*s.name = strings.TrimSpace(*s.name)
		name := *s.name
		switch {
		case name == "":
			errs = append(errs, field.Required(skillPath.Child("name"), "a name such as k6-load-testing"))
		case !validSkillName(name):
			errs = append(errs, invalidSkillName(skillPath.Child("name"), name))
		case seen[name]:
			errs = append(errs, field.Duplicate(skillPath.Child("name"), name))
		}
		seen[name] = true
		if s.from != nil {
			errs = append(errs, validateSkillSource(s.from, skillPath.Child("skillFrom"))...)
			errs = append(errs, forbidWithSkillFrom(skillPath, map[string]bool{
				"description": *s.description != "",
				"content":     s.content != "",
				"metadata":    len(s.metadata) > 0,
			})...)
			errs = append(errs, unresolvedSkillSource(skillPath.Child("skillFrom")))
			continue
		}
		errs = append(errs, normalizeSkillBody(s.description, s.content, s.metadata, skillPath)...)
	}
	if alwaysIncluded > MaxAlwaysIncludedSkillsLength {
		errs = append(errs, field.Invalid(path, field.OmitValueType{},
			"the content of the Skills always included is over 32000 characters in total; shorten one, or include it on demand"))
	}
	return errs
}

// tooManyCharacters is the error for a value over a limit counted in
// characters. field.TooLong says bytes, and would echo the value.
func tooManyCharacters(path *field.Path, limit int) *field.Error {
	return field.Invalid(path, field.OmitValueType{}, fmt.Sprintf("may not be more than %d characters", limit))
}

// validSkillName reports whether name is an Agent Skills spec name.
func validSkillName(name string) bool {
	return len(name) <= MaxSkillNameLength && skillNamePattern.MatchString(name)
}

func invalidSkillName(path *field.Path, name string) *field.Error {
	return field.Invalid(path, name, "must be up to 64 lowercase letters, digits and single hyphens, such as k6-load-testing")
}

// normalizeSkillBody trims a Skill's description and checks it, its content
// and its metadata keys against the spec's limits, under path.
func normalizeSkillBody(description *string, content string, metadata map[string]string, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	*description = strings.TrimSpace(*description)
	if *description == "" {
		errs = append(errs, field.Required(path.Child("description"), "what the Skill does and when to use it"))
	} else if utf8.RuneCountInString(*description) > MaxSkillDescriptionLength {
		errs = append(errs, tooManyCharacters(path.Child("description"), MaxSkillDescriptionLength))
	}
	if strings.TrimSpace(content) == "" {
		errs = append(errs, field.Required(path.Child("content"), "the guidance the AI follows"))
	} else if utf8.RuneCountInString(content) > MaxSkillContentLength {
		errs = append(errs, tooManyCharacters(path.Child("content"), MaxSkillContentLength))
	}
	for _, key := range sortedKeys(metadata) {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, field.Invalid(path.Child("metadata"), key, "a metadata entry needs a key"))
		}
	}
	return errs
}

// normalizeSkill checks a Skill of the environment's library. Its
// metadata.name is both a custom resource's name and an Agent Skills spec
// name, so it is a DNS-1123 label without consecutive hyphens.
func normalizeSkill(s *aiv1.Skill) field.ErrorList {
	namePath := field.NewPath("metadata", "name")
	errs := validateAIName(s.Name, namePath)
	if len(errs) == 0 && !validSkillName(s.Name) {
		errs = append(errs, invalidSkillName(namePath, s.Name))
	}
	spec := &s.Spec
	path := field.NewPath("spec")
	if t := spec.Testkube; t != nil && !t.AlwaysInclude && !t.AddToNewTestBundles {
		// Both off is what an absent block means; dropping it keeps an export,
		// which leaves it out, applying without a diff.
		spec.Testkube = nil
	}
	if spec.SkillFrom != nil {
		errs = append(errs, validateSkillSource(spec.SkillFrom, path.Child("skillFrom"))...)
		errs = append(errs, forbidWithSkillFrom(path, map[string]bool{
			"description":   spec.Description != "",
			"license":       spec.License != "",
			"compatibility": spec.Compatibility != "",
			"metadata":      len(spec.Metadata) > 0,
			"allowed-tools": spec.AllowedTools != "",
			"content":       spec.Content != "",
		})...)
		return append(errs, unresolvedSkillSource(path.Child("skillFrom")))
	}
	errs = append(errs, normalizeSkillBody(&spec.Description, spec.Content, spec.Metadata, path)...)
	spec.License = strings.TrimSpace(spec.License)
	spec.Compatibility = strings.TrimSpace(spec.Compatibility)
	if utf8.RuneCountInString(spec.Compatibility) > MaxSkillCompatibilityLength {
		errs = append(errs, tooManyCharacters(path.Child("compatibility"), MaxSkillCompatibilityLength))
	}
	// Space separated, as in a SKILL.md, however the YAML wrapped it.
	spec.AllowedTools = strings.Join(strings.Fields(spec.AllowedTools), " ")
	return errs
}

// normalizeSkillRefs checks an AI Agent's references to library Skills: each
// names a Skill once, in a known scope, and none takes the name of one of the
// agent's own Skills, which ownSkills holds. The order is kept, as it is the
// order the agent reads them in.
func normalizeSkillRefs(refs []aiv1.AIAgentSkillRef, ownSkills map[string]bool, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	seen := map[string]bool{}
	scopes := []string{aiv1.SkillScopeEnvironment, aiv1.SkillScopeOrganization}
	for i := range refs {
		ref := &refs[i]
		refPath := path.Index(i)
		ref.Name = strings.TrimSpace(ref.Name)
		switch {
		case ref.Name == "":
			errs = append(errs, field.Required(refPath.Child("name"), "the name of a Skill of the environment's or the organization's library"))
		case !validSkillName(ref.Name):
			errs = append(errs, invalidSkillName(refPath.Child("name"), ref.Name))
		case seen[ref.Name]:
			// Two scopes may hold a Skill of the same name, but the agent
			// knows its Skills by name alone.
			errs = append(errs, field.Duplicate(refPath.Child("name"), ref.Name))
		case ownSkills[ref.Name]:
			errs = append(errs, field.Invalid(refPath.Child("name"), ref.Name, "is also the name of one of the agent's own Skills in spec.skills; rename one of them"))
		}
		seen[ref.Name] = true
		if ref.Scope != "" && !slices.Contains(scopes, ref.Scope) {
			errs = append(errs, field.NotSupported(refPath.Child("scope"), ref.Scope, scopes))
		}
	}
	return errs
}
