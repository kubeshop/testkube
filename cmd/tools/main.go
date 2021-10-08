package main

import "github.com/kubeshop/testkube/cmd/tools/commands"

var (
	commit  string
	version string
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
