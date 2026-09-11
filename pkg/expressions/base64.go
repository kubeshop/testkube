package expressions

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

// EncodeBase64JSON hides a value inside a single opaque command argument.
//
// Every container argument is resolved by testworkflow-init with
// expressions.FinalizerFail, which aborts the step when anything fails to resolve.
// An argument that legitimately carries an unresolved expression - a cache key
// holding hash(file(...)), say, which can only be evaluated once the repository is
// checked out - must therefore not reach that pass as plain text. Encoding it keeps
// it inert until the toolkit command decodes it and resolves it against the
// context that can actually satisfy it.
func EncodeBase64JSON(data interface{}) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "encoding argument to JSON")
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

// DecodeBase64JSON reads back what EncodeBase64JSON wrote.
func DecodeBase64JSON(encoded string, target interface{}) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return errors.Wrapf(err, "decoding base64 argument (length=%d)", len(encoded))
	}
	if err := json.Unmarshal(decoded, target); err != nil {
		return errors.Wrap(err, "parsing JSON in base64 argument")
	}
	return nil
}
