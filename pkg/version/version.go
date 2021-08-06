package version

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver"
)

const (
	// Major version
	Major = "major"
	// Minor version
	Minor = "minor"
	// Patch version
	Patch = "patch"
)

// Next returns next generated semver based on version position
func Next(currentVersion, kind string) (string, error) {
	err := validateVersionPostion(kind)
	if err != nil {
		return "", err
	}

	version, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", err
	}
	var inc semver.Version

	switch kind {
	case Major:
		inc = version.IncMajor()
	case Minor:
		inc = version.IncMinor()
	case Patch:
		inc = version.IncPatch()
	default:
		return "", fmt.Errorf("invalid position" + kind)
	}

	return inc.String(), nil
}

// Lt checks if version1 is less-than version2, returns error in case of invalid version string
func Lt(version1, version2 string) (bool, error) {
	v1, err := semver.NewVersion(version1)
	if err != nil {
		return false, err
	}
	v2, err := semver.NewVersion(version2)
	if err != nil {
		return false, err
	}

	return v1.LessThan(v2), nil
}

func validateVersionPostion(kind string) error {
	if kind == Major || kind == Minor || kind == Patch {
		return nil
	}

	return fmt.Errorf("invalid version kind: %s: use one of major|minor|patch", kind)
}

func GetNewest(versions []string) string {
	semversions := []*semver.Version{}
	for _, ver := range versions {
		v, err := semver.NewVersion(ver)

		if err == nil {
			semversions = append(semversions, v)
		}
	}

	sort.Slice(semversions, func(i, j int) bool {
		return semversions[j].LessThan(semversions[i])
	})

	return semversions[0].String()
}
