package upgrade

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/beclab/Olares/cli/pkg/utils"
	"github.com/beclab/Olares/cli/version"
)

type releaseLine string

var (
	mainLine  = releaseLine("main")
	dailyLine = releaseLine("daily")

	dailyUpgraders = []breakingUpgrader{
		upgrader_1_12_0_20250702{},
	}
	mainUpgraders = []breakingUpgrader{}
)

func getReleaseLineOfVersion(v *semver.Version) releaseLine {
	preRelease := v.Prerelease()
	if preRelease == "" || strings.HasPrefix(preRelease, "rc") {
		return mainLine
	}
	return dailyLine
}

func check(base *semver.Version, target *semver.Version) error {
	if base == nil {
		return fmt.Errorf("base version is nil")
	}

	cliVersion, err := utils.ParseOlaresVersionString(version.VERSION)
	if err != nil {
		return fmt.Errorf("invalid olares-cli version :\"%s\"", version.VERSION)
	}

	if target != nil {
		if !target.GreaterThan(base) {
			return fmt.Errorf("base version: %s, target version: %s, no need to upgrade", base, target)
		}

		targetReleaseLine := getReleaseLineOfVersion(target)
		baseReleaseLine := getReleaseLineOfVersion(base)
		if targetReleaseLine != baseReleaseLine {
			return fmt.Errorf("unable to upgrade to %s on %s release line from %s on %s release line", target, targetReleaseLine, base, baseReleaseLine)
		}
		switch baseReleaseLine {
		case mainLine:
			if !sameMajorLevelVersion(base, target) {
				return fmt.Errorf("upgrade on %s rlease line can only be performed across same major level version", baseReleaseLine)
			}
		case dailyLine:
			if !samePatchLevelVersion(base, target) {
				return fmt.Errorf("upgrade on %s rlease line can only be performed across same patch version", baseReleaseLine)
			}
		}

		if target.GreaterThan(cliVersion) {
			return fmt.Errorf("target version: %s, cli version: %s, please upgrade olares-cli first!", target, cliVersion)
		}
	}

	if base.GreaterThan(cliVersion) {
		return fmt.Errorf("base version: %s, cli version: %s, please upgrade olares-cli first!", base, cliVersion)
	}

	return nil
}

func GetUpgradePathFor(base *semver.Version, target *semver.Version) ([]*semver.Version, error) {
	if err := check(base, target); err != nil {
		return nil, err
	}
	var path []*semver.Version
	var releaseLineUpgraders []breakingUpgrader
	var versionFilter func(v *semver.Version) bool
	switch getReleaseLineOfVersion(base) {
	case mainLine:
		releaseLineUpgraders = mainUpgraders
		versionFilter = func(v *semver.Version) bool {
			if !v.GreaterThan(base) {
				return false
			}
			if !sameMajorLevelVersion(v, base) {
				return false
			}
			if target != nil && !v.LessThan(target) {
				return false
			}
			return true
		}
	case dailyLine:
		if target == nil {
			cliVersion, err := utils.ParseOlaresVersionString(version.VERSION)
			if err != nil {
				return path, fmt.Errorf("invalid olares-cli version :\"%s\"", version.VERSION)
			}
			if getReleaseLineOfVersion(cliVersion) == dailyLine && samePatchLevelVersion(cliVersion, base) && cliVersion.GreaterThan(base) {
				target = cliVersion
			}
		}
		releaseLineUpgraders = dailyUpgraders
		versionFilter = func(v *semver.Version) bool {
			if !v.GreaterThan(base) {
				return false
			}
			if !samePatchLevelVersion(v, base) {
				return false
			}
			if target != nil && !v.LessThan(target) {
				return false
			}
			return true
		}
	}

	for _, u := range releaseLineUpgraders {
		v := u.Version()
		if versionFilter(v) {
			path = append(path, v)
		}
	}

	if target != nil {
		path = append(path, target)
	}

	return path, nil
}

func GetMinVersion() (*semver.Version, error) {
	cliVersion, err := utils.ParseOlaresVersionString(version.VERSION)
	if err != nil {
		return nil, fmt.Errorf("invalid olares-cli version :\"%s\"", version.VERSION)
	}

	var releaseLineUpgraders []breakingUpgrader
	var maxBreakingVersion *semver.Version

	switch getReleaseLineOfVersion(cliVersion) {
	case mainLine:
		releaseLineUpgraders = mainUpgraders
		maxBreakingVersion, err = semver.NewVersion("1.12.0-0") // default to the first breaking version
		if err != nil {
			return nil, fmt.Errorf("invalid default breaking version: %v", err)
		}
	case dailyLine:
		releaseLineUpgraders = dailyUpgraders
		maxBreakingVersion, err = semver.NewVersion("1.12.0-alpha") // default to the first breaking version
		if err != nil {
			return nil, fmt.Errorf("invalid default breaking version: %v", err)
		}
	}

	for _, u := range releaseLineUpgraders {
		v := u.Version()
		if v.LessThan(cliVersion) && v.GreaterThan(maxBreakingVersion) {
			maxBreakingVersion = v
		}
	}

	return maxBreakingVersion, nil
}

func getUpgraderByVersion(target *semver.Version) upgrader {
	for _, upgraders := range [][]breakingUpgrader{
		dailyUpgraders,
		mainUpgraders,
	} {

		for _, u := range upgraders {
			if u.Version().Equal(target) {
				return u
			}
		}
	}
	return upgraderBase{}
}

func samePatchLevelVersion(a, b *semver.Version) bool {
	return a.Major() == b.Major() && a.Minor() == b.Minor() && a.Patch() == b.Patch()
}

func sameMajorLevelVersion(a, b *semver.Version) bool {
	return a.Major() == b.Major()
}
