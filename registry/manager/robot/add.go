package robot

import (
	"encoding/json"
	"fmt"

	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
)

// AddOptions configures the Add operation.
type AddOptions struct {
	Version string
	TeamID  string // required: which team to add the robot to
}

// Add installs a robot package from the registry.
// Per DESIGN-ROBOT.md:
//  1. Pull .yao.zip
//  2. Parse robot.json + pkg.yao
//  3. Install dependencies (assistants + MCPs)
//  4. Write member DB record (deferred to CLI layer which has DB access)
//  5. Write registry.yao
func (m *Manager) Add(pkgID string, opts AddOptions) (*RobotJSON, error) {
	if opts.Version == "" {
		opts.Version = "latest"
	}
	if opts.TeamID == "" {
		return nil, fmt.Errorf("--team is required for robot add")
	}

	scope, name, err := common.ParsePackageID(pkgID)
	if err != nil {
		return nil, err
	}

	lf, err := common.LoadLockfile(m.appRoot)
	if err != nil {
		return nil, err
	}

	regType := common.TypeToRegistryType(common.TypeRobot)
	zipData, digest, err := m.client.Pull(regType, "@"+scope, name, opts.Version)
	if err != nil {
		return nil, fmt.Errorf("pull %s: %w", pkgID, err)
	}

	manifest, err := common.ReadManifest(zipData)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	// Read robot.json from zip
	robotData, err := common.ExtractFile(zipData, "robot.json")
	if err != nil {
		return nil, fmt.Errorf("extract robot.json: %w", err)
	}

	var robot RobotJSON
	if err := json.Unmarshal(robotData, &robot); err != nil {
		return nil, fmt.Errorf("parse robot.json: %w", err)
	}

	// Analyze dependencies from robot configuration
	analyzedDeps := AnalyzeDeps(&robot)

	// Merge with pkg.yao declared dependencies
	allDeps := map[string]string{}
	for _, dep := range analyzedDeps {
		allDeps[dep.PackageID] = "*"
	}
	for depID, ver := range manifest.Dependencies {
		allDeps[depID] = ver
	}

	// Install dependencies
	if len(allDeps) > 0 {
		missing, _, _ := common.CheckDependencies(allDeps, lf)
		if len(missing) > 0 {
			var summary string
			for _, dep := range missing {
				summary += fmt.Sprintf("  %s %s\n", dep.PackageID, dep.RequiredVersion)
			}
			if !m.prompter.Confirm(fmt.Sprintf("The following dependencies need to be installed:\n%sInstall?", summary)) {
				return nil, fmt.Errorf("dependency installation declined, aborting")
			}

			for _, dep := range missing {
				depScope, depName, err := common.ParsePackageID(dep.PackageID)
				if err != nil {
					return nil, err
				}

				// Reload lockfile before each dep install — a previous dep may
				// have recursively installed this one already.
				freshLF, _ := common.LoadLockfile(m.appRoot)
				if _, already := freshLF.GetPackage(dep.PackageID); already {
					fmt.Printf("  ✓ Dependency %s already installed (transitive)\n", dep.PackageID)
					continue
				}

				depType := depTypeFor(dep.PackageID, analyzedDeps)

				switch depType {
				case "mcp":
					err = m.mcpMgr.Add(dep.PackageID, mcpmgr.AddOptions{})
				default:
					err = m.agentMgr.Add(dep.PackageID, agentmgr.AddOptions{})
				}
				if err != nil {
					return nil, fmt.Errorf("failed to install dependency %s (%s/%s): %w", dep.PackageID, depScope, depName, err)
				}
				fmt.Printf("  ✓ Dependency %s installed\n", dep.PackageID)
			}

			// Reload lockfile after dependency installation
			lf, err = common.LoadLockfile(m.appRoot)
			if err != nil {
				return nil, err
			}
		}
	}

	// Write to registry.yao (member record writing is done by CLI layer)
	info := common.PackageInfo{
		Type:         common.TypeRobot,
		Version:      manifest.Version,
		Integrity:    digest,
		Dependencies: allDeps,
		TeamID:       opts.TeamID,
	}
	lf.SetPackage(pkgID, info)

	// Add required_by references
	for depID := range allDeps {
		lf.AddRequiredBy(depID, pkgID)
	}

	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return nil, err
	}

	fmt.Printf("✓ Robot %s@%s installed (dependencies ready, team: %s)\n", pkgID, manifest.Version, opts.TeamID)
	fmt.Printf("  The member record needs to be created in the database.\n")
	return &robot, nil
}

// depTypeFor finds the type of a dependency from the analyzed deps list.
func depTypeFor(pkgID string, analyzedDeps []RobotDep) string {
	for _, d := range analyzedDeps {
		if d.PackageID == pkgID {
			return d.Type
		}
	}
	return "assistant"
}
