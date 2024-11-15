package diagnostics

import (
	"errors"

	"github.com/kubeshop/testkube/pkg/diagnostics/renderer"
	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

var (
	ErrGroupNotFound = errors.New("group not found")
)

func New() Diagnostics {
	return Diagnostics{
		Renderer: renderer.NewCLIRenderer(),
		Groups:   map[string]validators.ValidatorGroup{},
	}
}

type Diagnostics struct {
	Renderer renderer.Renderer
	Groups   map[string]validators.ValidatorGroup
}

func (d Diagnostics) Run() error {
	for groupName, _ := range d.Groups {
		if err := d.RunGroup(groupName); err != nil {
			return err
		}
	}
	return nil
}

func (d Diagnostics) runValidator(v validators.Validator) {
	res := v.Validate()
	d.Renderer.RenderResult(res)
}

func (d Diagnostics) RunGroup(group string) error {
	g, ok := d.Groups[group]
	if !ok {
		return ErrGroupNotFound
	}
	d.Renderer.RenderGroupStart(group)
	for _, v := range g.Validators {
		d.runValidator(v)
	}
	return nil
}

func (d *Diagnostics) AddValidator(v validators.Validator, optGroup ...string) {
	group := "default"
	if len(optGroup) > 0 {
		group = optGroup[0]
	}

	g, ok := d.Groups[group]
	if !ok {
		d.Groups[group] = validators.ValidatorGroup{}
	}
	g.Validators = append(g.Validators, v)
	d.Groups[group] = g
}
