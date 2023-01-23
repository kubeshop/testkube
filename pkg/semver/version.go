package semver

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

// NextPrerelease returns pre-release version e.g. current -> 0.10.1; nextPrerelease -> 0.10.2-beta001
func NextPrerelease(currentVersion string) (string, error) {
	version, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", err
	}

	if version.Prerelease() != "" {
		version = bumpPrerelease(version)
		return version.String(), nil
	}

	return "", nil

}

// IsPrerelease detects if release is prerelease
func IsPrerelease(currentVersion string) bool {
	version, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false
	}

	return version.Prerelease() != ""

}

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

// bumpPrerelease bumps number in versions like 0.0.1-alpha2 or 0.0.3-omega4
func bumpPrerelease(version *semver.Version) *semver.Version {
	prerelease := version.Prerelease()
	r := regexp.MustCompile("[0-9]+$")

	matches := r.FindStringSubmatch(prerelease)
	if len(matches) == 1 {
		num, err := strconv.Atoi(matches[0])
		if err == nil {
			num = num + 1
			prerelease = strings.Replace(prerelease, matches[0], fmt.Sprintf("%03d", num), -1)
			v, _ := version.SetPrerelease(prerelease)
			return &v

		}
	}

	return version
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

// Lte checks if version1 is less-than or equal version2, returns error in case of invalid version string
func Lte(version1, version2 string) (bool, error) {
	ok, err := Lt(version1, version2)
	if err != nil {
		return false, err
	}

	return ok || version1 == version2, nil
}

func validateVersionPostion(kind string) error {
	if kind == Major || kind == Minor || kind == Patch {
		return nil
	}

	return fmt.Errorf("invalid version kind: %s: use one of major|minor|patch", kind)
}

// GetNewest returns greatest version from passed versions list
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
