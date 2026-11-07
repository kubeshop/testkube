package declarative

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// MaxSkillNameLength, MaxSkillDescriptionLength and MaxSkillContentLength
	// are the Agent Skills spec's limits, the Control Plane's too. The lengths
	// of a description and content are in characters.
	MaxSkillNameLength        = 64
	MaxSkillDescriptionLength = 1024
	MaxSkillContentLength     = 32000

	// MaxAlwaysIncludedSkillsLength caps, in characters, the content of all
	// the Skills an owner always includes together: every prompt carries them.
	MaxAlwaysIncludedSkillsLength = 32000
)

// skillNamePattern is the Agent Skills spec's name: lowercase letters, digits
// and single hyphens.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skillFields are the fields of a Skill every kind's Skill type has.
type skillFields struct {
	name, description *string
	content           string
	metadata          map[string]string
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
		*s.description = strings.TrimSpace(*s.description)
		name := *s.name
		switch {
		case name == "":
			errs = append(errs, field.Required(skillPath.Child("name"), "a name such as k6-load-testing"))
		case len(name) > MaxSkillNameLength || !skillNamePattern.MatchString(name):
			errs = append(errs, field.Invalid(skillPath.Child("name"), name, "must be up to 64 lowercase letters, digits and single hyphens, such as k6-load-testing"))
		case seen[name]:
			errs = append(errs, field.Duplicate(skillPath.Child("name"), name))
		}
		seen[name] = true
		if *s.description == "" {
			errs = append(errs, field.Required(skillPath.Child("description"), "what the Skill does and when to use it"))
		} else if utf8.RuneCountInString(*s.description) > MaxSkillDescriptionLength {
			errs = append(errs, field.TooLong(skillPath.Child("description"), "", MaxSkillDescriptionLength))
		}
		if strings.TrimSpace(s.content) == "" {
			errs = append(errs, field.Required(skillPath.Child("content"), "the guidance the AI follows"))
		} else if utf8.RuneCountInString(s.content) > MaxSkillContentLength {
			errs = append(errs, field.TooLong(skillPath.Child("content"), "", MaxSkillContentLength))
		}
		for key := range s.metadata {
			if strings.TrimSpace(key) == "" {
				errs = append(errs, field.Invalid(skillPath.Child("metadata"), key, "a metadata entry needs a key"))
			}
		}
	}
	if alwaysIncluded > MaxAlwaysIncludedSkillsLength {
		errs = append(errs, field.Invalid(path, field.OmitValueType{},
			"the content of the Skills always included is over 32000 characters in total; shorten one, or include it on demand"))
	}
	return errs
}
