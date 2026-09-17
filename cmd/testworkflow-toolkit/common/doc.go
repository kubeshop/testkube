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
// When the environment has no TK_ERR_FILE, for example outside a workflow step in a test,
// the helpers only call pkg/ui.
//
// The env and env/config packages cannot import this package, because that makes an import
// cycle. A failure in these packages, for example a Kubernetes configuration that the toolkit
// cannot load, gives no step message.
package common
