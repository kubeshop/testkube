package main

import (
	"github.com/kubeshop/testkube/cmd/diagnostics/commands"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
)

var (
	commit  string
	version string = "999.0.0-dev"
	builtBy string
	date    string
)

func init() {
	// pass data from goreleaser to commands package
	common.Version = version
	common.BuiltBy = builtBy
	common.Commit = commit
	common.Date = date
}

func main() {
	commands.Execute()
}
