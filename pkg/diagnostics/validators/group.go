package validators

// ValidatorGroup to organize validators around given subject e.g. "License Key"
type ValidatorGroup struct {
	Subject    any
	Name       string
	Validators []Validator
}

// AddValidator adds a new validator to the group
func (vg *ValidatorGroup) AddValidator(v Validator) {
	vg.Validators = append(vg.Validators, v)
}
