// Package common holds the helpers that the toolkit commands share.
//
// # Step messages
//
// A toolkit command that fails calls Fail, Failf, or ExitOnError, not the functions of
// pkg/ui with the same names. The helper writes the cause to the file that TK_ERR_FILE
// names, and then calls the pkg/ui function, which prints the error and exits.
//
// The init process sets TK_ERR_FILE for a toolkit step. When the step fails, the init
// process sends the first line of the file as the step message. Thus write the cause in
// the first line, as plain text. The orchestration package of the init process documents
// the full contract.
//
// Fail and ExitOnError also write the reason code of the error to the file that TK_REASON_FILE
// names. An error that implements ReasonError carries its own code. WithReason gives a code to an
// error that has none. Give a code only to a failure that the code names exactly. The init process
// accepts only a known code.
//
// An error in the definition of a step, for example a count that does not resolve, is a
// DefinitionError of the commands package. It carries definition-invalid, so the user sees a
// configuration error and not a failed test.
//
// When the environment has no TK_ERR_FILE or TK_REASON_FILE, for example outside a workflow step
// in a test, the helpers do not write that file.
//
// The env and env/config packages cannot import this package, because that makes an import
// cycle. A failure in these packages, for example a Kubernetes configuration that the toolkit
// cannot load, gives no step message.
package common
