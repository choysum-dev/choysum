package policy

import (
	"encoding/json"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

type InstalledModuleLoader func(name string) (*meta.IrModule, error)

func ResolveInstalledDependencies(load InstalledModuleLoader, module *meta.IrModule) ([]*meta.IrModule, error) {
	if module == nil {
		return nil, nil
	}
	if load == nil {
		return nil, xfmt.Errorf("dependency loader is nil")
	}

	dependsArr := make([]string, 0)
	if err := json.Unmarshal(module.DependsStr, &dependsArr); err != nil {
		return nil, xfmt.Errorf("error unmarshal depends: %w", err)
	}
	seen := map[string]bool{}
	deps := make([]*meta.IrModule, 0, len(dependsArr))
	for _, dep := range dependsArr {
		dep = strings.TrimSpace(dep)
		if dep == "" || seen[dep] {
			continue
		}
		seen[dep] = true

		installedDep, err := load(dep)
		if err != nil {
			return nil, xfmt.Errorf("error loading installed dependency %s: %w", dep, err)
		}
		if installedDep == nil || installedDep.Status != meta.Installed {
			return nil, xfmt.Errorf("dependency %s is not installed; expected plan to install dependencies before %s", dep, module.Name)
		}
		deps = append(deps, installedDep)
	}
	return deps, nil
}
