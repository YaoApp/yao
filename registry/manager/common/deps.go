package common

import (
	"fmt"
	"strings"
)

// DepStatus represents the status of a dependency check.
type DepStatus int

const (
	DepNotInstalled DepStatus = iota // Not installed at all
	DepSatisfied                     // Installed and version satisfies requirement
	DepConflict                      // Installed but version does not satisfy requirement
)

// DepCheckResult holds the result of checking a single dependency.
type DepCheckResult struct {
	PackageID        string
	RequiredVersion  string
	InstalledVersion string
	Status           DepStatus
}

// CheckDependencies checks each dependency from the manifest against the lockfile.
// Returns missing, conflicting, and satisfied dependencies.
func CheckDependencies(deps map[string]string, lf *RegistryYao) (missing, conflicts, satisfied []DepCheckResult) {
	for pkgID, requiredVer := range deps {
		installed, ok := lf.GetPackage(pkgID)
		if !ok {
			missing = append(missing, DepCheckResult{
				PackageID:       pkgID,
				RequiredVersion: requiredVer,
				Status:          DepNotInstalled,
			})
			continue
		}

		if VersionSatisfies(installed.Version, requiredVer) {
			satisfied = append(satisfied, DepCheckResult{
				PackageID:        pkgID,
				RequiredVersion:  requiredVer,
				InstalledVersion: installed.Version,
				Status:           DepSatisfied,
			})
		} else {
			conflicts = append(conflicts, DepCheckResult{
				PackageID:        pkgID,
				RequiredVersion:  requiredVer,
				InstalledVersion: installed.Version,
				Status:           DepConflict,
			})
		}
	}
	return
}

// DetectCycle checks if adding pkgID to the installing set would cause a cycle.
// Returns true if a cycle is detected.
func DetectCycle(installing map[string]bool, pkgID string) bool {
	return installing[pkgID]
}

// VersionSatisfies checks if installedVer satisfies the constraint.
// Supports:
//   - "^X.Y.Z" — same major, >= minor.patch
//   - ">=X.Y.Z" — greater or equal
//   - "X.Y.Z" — exact match
//   - "*" — any version
func VersionSatisfies(installedVer, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}

	if strings.HasPrefix(constraint, "^") {
		return caretSatisfies(installedVer, constraint[1:])
	}
	if strings.HasPrefix(constraint, ">=") {
		return compareVersions(installedVer, strings.TrimSpace(constraint[2:])) >= 0
	}

	// Exact match
	return installedVer == constraint
}

// caretSatisfies implements ^X.Y.Z: same major version, >= the specified version.
func caretSatisfies(installed, minVer string) bool {
	iMajor, iMinor, iPatch, err := parseVersion(installed)
	if err != nil {
		return false
	}
	mMajor, mMinor, mPatch, err := parseVersion(minVer)
	if err != nil {
		return false
	}

	if iMajor != mMajor {
		return false
	}
	if iMinor > mMinor {
		return true
	}
	if iMinor == mMinor {
		return iPatch >= mPatch
	}
	return false
}

func compareVersions(a, b string) int {
	aMaj, aMin, aPat, err1 := parseVersion(a)
	bMaj, bMin, bPat, err2 := parseVersion(b)
	if err1 != nil || err2 != nil {
		if a == b {
			return 0
		}
		if a > b {
			return 1
		}
		return -1
	}

	if aMaj != bMaj {
		return aMaj - bMaj
	}
	if aMin != bMin {
		return aMin - bMin
	}
	return aPat - bPat
}

func parseVersion(v string) (major, minor, patch int, err error) {
	v = strings.TrimSpace(v)
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version %q", v)
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major in %q", v)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor in %q", v)
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch in %q", v)
	}
	return major, minor, patch, nil
}
