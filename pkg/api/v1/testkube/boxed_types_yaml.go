/*
 * Testkube API
 *
 * YAML unmarshaling support for boxed types.
 * This file adds UnmarshalYAML methods to allow YAML documents to use
 * either the shorthand form (direct value) or the object form (with "value" field).
 *
 * For example, both of these are valid for BoxedString:
 *   shell: "echo hello"           # shorthand
 *   shell:
 *     value: "echo hello"         # object form
 */
package testkube

import (
	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements yaml.Unmarshaler for BoxedString.
// It accepts both direct string values and object form with "value" field.
func (b *BoxedString) UnmarshalYAML(node *yaml.Node) error {
	// Try direct string first (shorthand form)
	if node.Kind == yaml.ScalarNode {
		b.Value = node.Value
		return nil
	}

	// Otherwise, try object form with "value" field
	type boxedStringAlias BoxedString
	var alias boxedStringAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*b = BoxedString(alias)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for BoxedBoolean.
// It accepts both direct boolean values and object form with "value" field.
func (b *BoxedBoolean) UnmarshalYAML(node *yaml.Node) error {
	// Try direct boolean first (shorthand form)
	if node.Kind == yaml.ScalarNode {
		var value bool
		if err := node.Decode(&value); err != nil {
			return err
		}
		b.Value = value
		return nil
	}

	// Otherwise, try object form with "value" field
	type boxedBooleanAlias BoxedBoolean
	var alias boxedBooleanAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*b = BoxedBoolean(alias)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for BoxedInteger.
// It accepts both direct integer values and object form with "value" field.
func (b *BoxedInteger) UnmarshalYAML(node *yaml.Node) error {
	// Try direct integer first (shorthand form)
	if node.Kind == yaml.ScalarNode {
		var value int32
		if err := node.Decode(&value); err != nil {
			return err
		}
		b.Value = value
		return nil
	}

	// Otherwise, try object form with "value" field
	type boxedIntegerAlias BoxedInteger
	var alias boxedIntegerAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*b = BoxedInteger(alias)
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for BoxedStringList.
// It accepts both direct string array values and object form with "value" field.
func (b *BoxedStringList) UnmarshalYAML(node *yaml.Node) error {
	// Try direct array first (shorthand form)
	if node.Kind == yaml.SequenceNode {
		var value []string
		if err := node.Decode(&value); err != nil {
			return err
		}
		b.Value = value
		return nil
	}

	// Otherwise, try object form with "value" field
	type boxedStringListAlias BoxedStringList
	var alias boxedStringListAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}
	*b = BoxedStringList(alias)
	return nil
}
