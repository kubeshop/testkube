package marketplace

import (
	"fmt"
	"io"

	"github.com/pterm/pterm"

	"github.com/kubeshop/testkube/pkg/marketplace"
)

// Prompter is the narrow interface used to collect input from the user
// during `testkube marketplace install`. It is implemented by the pterm-backed
// prompter at runtime and by stubs in tests.
type Prompter interface {
	// Prompt is called once per parameter. The returned string is the raw
	// user input. An empty string means "keep the current value" (which is
	// either the parameter's YAML default or a prior --set override).
	Prompt(p marketplace.Parameter) (string, error)

	// Confirm asks a yes/no question and returns the user's answer.
	// defaultYes controls which option is highlighted as the default when
	// the user simply presses enter.
	Confirm(message string, defaultYes bool) (bool, error)
}

// promptForParameters walks the parameters and asks the supplied prompter
// for a value for each one. Empty input preserves the current Value.
//
// The function mutates the Value fields in place so downstream callers
// (ApplyParameters) receive the merged result.
func promptForParameters(w io.Writer, params []marketplace.Parameter, prompter Prompter) ([]marketplace.Parameter, error) {
	if len(params) == 0 {
		return params, nil
	}
	fmt.Fprintf(w, "\nThe workflow exposes %d parameter(s). Press enter to accept the shown default.\n\n", len(params))
	for i := range params {
		p := params[i]
		typeLabel := p.Type
		if typeLabel == "" {
			typeLabel = "string"
		}
		header := fmt.Sprintf("%d/%d  %s (%s)", i+1, len(params), p.Key, typeLabel)
		if p.Sensitive {
			header += "  [sensitive]"
		}
		fmt.Fprintln(w, header)
		if p.Description != "" {
			fmt.Fprintf(w, "     %s\n", p.Description)
		}

		in, err := prompter.Prompt(p)
		if err != nil {
			return nil, fmt.Errorf("reading value for %q: %w", p.Key, err)
		}
		if in != "" {
			params[i].Value = in
		}
		fmt.Fprintln(w)
	}
	return params, nil
}

// ptermPrompter is the default runtime Prompter. It uses pterm for both
// plain and masked (sensitive) text input.
type ptermPrompter struct{}

func (ptermPrompter) Prompt(p marketplace.Parameter) (string, error) {
	// WithMultiLine(false) normalises the builder to a value (other
	// builders on this type return pointers), mirroring pkg/ui/textinput.go.
	t := pterm.DefaultInteractiveTextInput.WithMultiLine(false)
	if p.Sensitive {
		// Masked input deliberately does not display the current value so
		// we don't leak a --set secret back onto the terminal. Empty input
		// keeps whatever the caller already had.
		return t.WithMask("*").Show("value (leave empty to keep current)")
	}
	if p.Value != "" {
		t = t.WithDefaultValue(p.Value)
	}
	return t.Show("value")
}

func (ptermPrompter) Confirm(message string, defaultYes bool) (bool, error) {
	return pterm.DefaultInteractiveConfirm.WithDefaultValue(defaultYes).Show(message)
}
