package diagnostics

import (
	"errors"
	"sync"

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
		Groups:   map[string]*validators.ValidatorGroup{},
	}
}

// Diagnostics Top level diagnostics service to organize validators in groups
type Diagnostics struct {
	Renderer renderer.Renderer
	Groups   map[string]*validators.ValidatorGroup
}

// Run executes all validators in all groups and renders the results
func (d Diagnostics) Run() error {

	// for now we'll make validators concurrent
	for groupName := range d.Groups {
		ch, err := d.RunGroup(groupName)
		if err != nil {
			return err
		}

		d.Renderer.RenderGroupStart(groupName)
		for r := range ch {
			d.Renderer.RenderResult(r)
			if r.BreakValidationChain {
				break
			}
		}
	}

	return nil
}

// RunGroup tries to locate group and run it
func (d Diagnostics) RunGroup(group string) (chan validators.ValidationResult, error) {
	ch := make(chan validators.ValidationResult)
	g, ok := d.Groups[group]
	if !ok {
		return ch, ErrGroupNotFound
	}

	go func() {
		var wg sync.WaitGroup

		defer close(ch)

		if len(g.Validators) > 0 {
			for _, v := range g.Validators {
				wg.Add(1)
				go func(v validators.Validator) {
					defer wg.Done()
					ch <- v.Validate(g.Subject).WithValidator(v.Name())
				}(v)
			}
			wg.Wait()
		}
	}()

	return ch, nil
}

func (d *Diagnostics) AddValidatorGroup(group string, subject any) *validators.ValidatorGroup {
	d.Groups[group] = &validators.ValidatorGroup{
		Subject: subject,
		Name:    group,
	}
	return d.Groups[group]
}
