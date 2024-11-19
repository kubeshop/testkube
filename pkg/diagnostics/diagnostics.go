package diagnostics

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/diagnostics/renderer"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

const (
	DefaultValidatorGroupName = "default "
)

var (
	ErrGroupNotFound = errors.New("group not found")
)

func New() Diagnostics {
	return Diagnostics{
		Renderer: renderer.NewCLIRenderer(),
		Groups: map[string]*validators.ValidatorGroup{
			DefaultValidatorGroupName: &validators.ValidatorGroup{},
		},
	}
}

// Diagnostics Top level diagnostics service to organize validators in groups
type Diagnostics struct {
	Renderer renderer.Renderer
	Groups   map[string]*validators.ValidatorGroup
}

// TODO make it parallel
// Run executes all validators in all groups and renders the results
func (d Diagnostics) Run() error {
	for groupName, _ := range d.Groups {
		if err := d.RunGroup(groupName); err != nil {
			return err
		}
	}
	return nil
}

func (d Diagnostics) runValidator(v validators.Validator, subject any) {
	res := v.Validate(subject)
	d.Renderer.RenderResult(res)
}

func (d Diagnostics) RunGroup(group string) error {
	g, ok := d.Groups[group]
	if !ok {
		return ErrGroupNotFound
	}
	if len(g.Validators) > 0 {
		d.Renderer.RenderGroupStart(group)
		for _, v := range g.Validators {
			d.runValidator(v, g.Subject)
		}
	}
	return nil
}

func (d *Diagnostics) AddValidatorGroup(group string, subject any) *validators.ValidatorGroup {
	d.Groups[group] = &validators.ValidatorGroup{
		Subject: subject,
		Name:    group,
	}
	return d.Groups[group]
}
