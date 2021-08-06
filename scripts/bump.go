package main

import (
	"flag"
	"strings"

	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/ui"
	"github.com/kubeshop/kubetest/pkg/version"
)

var kind = flag.String("kind", "patch", "version kind to bump can be one of: major|minor|patch (patch is default one)")

func main() {

	out, err := process.Execute("git", "tag")
	ui.ExitOnError("getting tags", err)

	versions := strings.Split(string(out), "\n")
	currentVersion := version.GetNewest(versions)
	nextVersion, err := version.Next(currentVersion, *kind)
	ui.ExitOnError("getting next version for "+*kind, err)

	ui.Info("Generated new version", nextVersion)

	_, err = process.Execute("git", "tag", nextVersion)
	ui.ExitOnError("tagging new version", err)

	_, err = process.Execute("git", "push", "--tags")
	ui.ExitOnError("pushing new version to repository", err)
}
