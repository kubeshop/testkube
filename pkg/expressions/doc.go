// Package expressions parses, simplifies and resolves the expressions of a test workflow, for
// example `{{ config.message }}`.
//
// # Errors
//
// A finalizer that cannot resolve a variable returns ErrUnknownVariable. The accessor wraps it in
// UndefinedError, which names the variable for the user: `<name> is not defined`. A caller that
// needs the kind of the error uses errors.Is with ErrUnknownVariable. A caller that needs the name
// uses errors.As with UndefinedError. So no caller reads the text of the error.
package expressions
