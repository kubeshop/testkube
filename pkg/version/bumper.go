package version

import (
	"flag"
	"fmt"
	"strings"

	"github.com/kubeshop/kubtest/pkg/process"
	"github.com/kubeshop/kubtest/pkg/ui"
)

var kind = flag.String("kind", "patch", "version kind to bump can be one of: major|minor|patch (patch is default one)")
var verbose = flag.Bool("verbose", false, "version kind to bump can be one of: major|minor|patch (patch is default one)")

func init() {
	fmt.Printf("%+v\n", *verbose)

	flag.Parse()
	ui.Verbose = *verbose
}

func Bump() {
	out, err := process.Execute("git", "tag")
	ui.ExitOnError("getting tags", err)

	versions := strings.Split(string(out), "\n")
	currentVersion := GetNewest(versions)
	nextVersion, err := Next(currentVersion, *kind)
	ui.ExitOnError("getting next version for "+*kind, err)
	nextVersion = "v" + nextVersion

	ui.Info("Generated new version", nextVersion)

	_, err = process.Execute("git", "tag", nextVersion)
	ui.ExitOnError("tagging new version", err)

	_, err = process.Execute("git", "push", "--tags")
	ui.ExitOnError("pushing new version to repository", err)
}
