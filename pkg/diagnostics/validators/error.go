package validators

import "errors"

func IsError(err error) bool {
	return errors.Is(err, &Error{})
}

func Err(e string, kind ErrorKind, suggestions ...string) Error {
	err := Error{Message: e, Suggestions: suggestions}
	return err
}

type ErrorKind string

type Error struct {
	Kind        ErrorKind
	Message     string
	Details     string
	Suggestions []string
	DocsURI     string
}

func (e Error) Error() string {
	s := ""
	if e.Message != "" {
		s += e.Message
	}
	if e.Details != "" {
		s += " - " + e.Details
	}
	return s
}

func (e Error) WithSuggestion(s string) Error {
	e.Suggestions = append(e.Suggestions, s)
	return e
}

func (e Error) WithDetails(d string) Error {
	e.Details = d
	return e
}

func (e Error) WithDocsURI(d string) Error {
	e.DocsURI = d
	return e
}
