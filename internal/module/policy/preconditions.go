package policy

import (
	"slices"

	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

func RequireInstalledForUpgrade(module *meta.IrModule) error {
	if module == nil {
		return xfmt.Errorf("module is nil")
	}
	if !slices.Contains([]meta.Status{meta.Installed, meta.ToUpgrade}, module.Status) {
		return xfmt.Errorf("module %s is not installed", module.Name)
	}
	return nil
}
