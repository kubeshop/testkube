package testworkflowprocessor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// stepIdPattern allows lowercase alphanumeric and underscores.
var stepIdPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]*$`)

// ValidateStepId checks that a step ID contains only lowercase alphanumeric
// characters and underscores.
func ValidateStepId(id string) error {
	if !stepIdPattern.MatchString(id) {
		return fmt.Errorf("step id %q must contain only lowercase alphanumeric characters and underscores", id)
	}
	return nil
}

// DeriveStepId converts a step name to an identifier candidate by lowercasing
// and replacing all non-alphanumeric characters with underscores.
// The result may not pass ValidateStepId (e.g., non-ASCII names); callers must validate.
func DeriveStepId(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevUnderscore = false
		} else if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	result := strings.Trim(b.String(), "_")
	if result == "" {
		return ""
	}
	return result
}

// ResolveStepId returns the step's effective ID: explicit id > derived from name > empty.
func ResolveStepId(step testworkflowsv1.StepMeta) string {
	if step.Id != "" {
		return step.Id
	}
	return DeriveStepId(step.Name)
}

// ValidateExplicitStepIds checks that all explicitly set step IDs have a valid format
// and are unique within their scope. Parallel blocks are validated independently
// since they run as separate workflows. This is intended for API-time validation
// before the workflow is stored.
func ValidateExplicitStepIds(spec *testworkflowsv1.TestWorkflowSpec) error {
	seen := make(map[string]bool)
	for _, steps := range [][]testworkflowsv1.Step{spec.Setup, spec.Steps, spec.After} {
		for i := range steps {
			if err := validateExplicitStepId(&steps[i], seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateExplicitStepId(step *testworkflowsv1.Step, seen map[string]bool) error {
	if step.Id != "" {
		if err := ValidateStepId(step.Id); err != nil {
			return err
		}
		if seen[step.Id] {
			return fmt.Errorf("duplicate step id %q", step.Id)
		}
		seen[step.Id] = true
	}

	// Recurse into nested steps (same ID namespace)
	for _, steps := range [][]testworkflowsv1.Step{step.Setup, step.Steps} {
		for i := range steps {
			if err := validateExplicitStepId(&steps[i], seen); err != nil {
				return err
			}
		}
	}

	// Parallel blocks have their own ID namespace (separate pods)
	if step.Parallel != nil {
		parallelSeen := make(map[string]bool)
		for _, steps := range [][]testworkflowsv1.Step{step.Parallel.Setup, step.Parallel.Steps, step.Parallel.After} {
			for i := range steps {
				if err := validateExplicitStepId(&steps[i], parallelSeen); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ResolveAndValidateStepIds walks all steps in a workflow, auto-derives IDs
// from names where not explicitly set, and validates all IDs (format and uniqueness).
// Auto-derived IDs that conflict get an index suffix (e.g., build, build_1, build_2).
func ResolveAndValidateStepIds(spec *testworkflowsv1.TestWorkflowSpec) error {
	seen := make(map[string]bool)
	for _, steps := range [][]testworkflowsv1.Step{spec.Setup, spec.Steps, spec.After} {
		for i := range steps {
			if err := resolveAndValidateStep(&steps[i], seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveAndValidateStep(step *testworkflowsv1.Step, seen map[string]bool) error {
	if step.Id != "" {
		// Explicit ID: strict validation
		if err := ValidateStepId(step.Id); err != nil {
			return err
		}
		if seen[step.Id] {
			return fmt.Errorf("duplicate step id %q", step.Id)
		}
		seen[step.Id] = true
	} else if step.Name != "" {
		// Auto-derive from name, appending _1, _2, etc. on conflict
		derived := DeriveStepId(step.Name)
		if derived != "" && ValidateStepId(derived) == nil {
			candidate := derived
			for n := 1; seen[candidate]; n++ {
				candidate = fmt.Sprintf("%s_%d", derived, n)
			}
			step.Id = candidate
			seen[candidate] = true
		}
	}

	// Recurse into nested steps
	for _, steps := range [][]testworkflowsv1.Step{step.Setup, step.Steps} {
		for i := range steps {
			if err := resolveAndValidateStep(&steps[i], seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// AssignStageIds gives an id to each stage that has none, so every step of an execution has a stable
// key. The id is the id of the parent, then the name or the category of the stage, with an index
// suffix when the id is taken. The root group is the workflow itself and gets no id.
func AssignStageIds(root stage.Stage) {
	used := make(map[string]bool)
	collectStageIds(root, used)
	group, ok := root.(stage.GroupStage)
	if !ok {
		assignStageId(root, "", used)
		return
	}
	for _, child := range group.Children() {
		assignStageId(child, "", used)
	}
}

func collectStageIds(s stage.Stage, used map[string]bool) {
	if s.Id() != "" {
		used[s.Id()] = true
	}
	if group, ok := s.(stage.GroupStage); ok {
		for _, child := range group.Children() {
			collectStageIds(child, used)
		}
	}
}

func assignStageId(s stage.Stage, parentId string, used map[string]bool) {
	group, isGroup := s.(stage.GroupStage)
	if s.Id() == "" {
		s.SetId(newStageId(s, parentId, used))
		// The stage that runs the operation of a step owns the id of the step, because it writes the
		// outputs of the step. Flatten gives a group id to that stage, so a new group id goes there too.
		if isGroup && len(group.Children()) > 0 && group.Children()[0].Id() == "" {
			group.Children()[0].SetId(s.Id())
		}
	}
	if isGroup {
		for _, child := range group.Children() {
			assignStageId(child, s.Id(), used)
		}
	}
}

// newStageId returns an unused id for the stage, from the id of its parent and its name or category.
// A group without both uses the category of its first child, which runs the operation of the step.
func newStageId(s stage.Stage, parentId string, used map[string]bool) string {
	base := DeriveStepId(s.Name())
	if ValidateStepId(base) != nil {
		base = DeriveStepId(s.Category())
	}
	if group, ok := s.(stage.GroupStage); ok && ValidateStepId(base) != nil && len(group.Children()) > 0 {
		base = DeriveStepId(group.Children()[0].Category())
	}
	if ValidateStepId(base) != nil {
		base = "step"
	}
	if parentId != "" {
		base = parentId + "_" + base
	}
	id := limitStageId(base)
	for n := 1; used[id]; n++ {
		id = limitStageId(fmt.Sprintf("%s_%d", base, n))
	}
	used[id] = true
	return id
}

// maxStageIdLength is the maximum length of a generated id. The runner uses the id in the name of a
// directory, and a file name has a limit of 255 bytes. The limit leaves room for a prefix or a suffix.
const maxStageIdLength = 128

// limitStageId cuts an id that is too long and adds a hash of the full id. The hash keeps the id stable
// for the same workflow, and different long ids stay different.
func limitStageId(id string) string {
	if len(id) <= maxStageIdLength {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	hash := hex.EncodeToString(sum[:])[:8]
	return strings.TrimRight(id[:maxStageIdLength-len(hash)-1], "_") + "_" + hash
}
