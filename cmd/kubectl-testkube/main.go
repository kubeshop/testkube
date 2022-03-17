package main

import (
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands"
)

var (
	commit  string
	version string = "999.0.0-dev" // simple bypass of upgrading cluster if coming from dev build or go run
	builtBy string
	date    string
)

func init() {
	// pass data from goreleaser to commands package
	commands.Version = version
	commands.BuiltBy = builtBy
	commands.Commit = commit
	commands.Date = date
}

func main() {
	commands.Execute()
}
