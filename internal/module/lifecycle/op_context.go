// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"slices"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/plan"
)

type opContext struct {
	withDemo bool
	opid     string

	fromVersion map[string]string

	schemaIntents schema.IntentBag

	installStack []string
	installing   map[string]bool
	installDone  map[string]bool
	// installTouched records modules that performed real install work (not already_installed).
	installTouched map[string]bool

	uninstallStack []string
	uninstalling   map[string]bool
	uninstallDone  map[string]bool

	upgradeStack []string
	upgrading    map[string]bool
	upgradeDone  map[string]bool
	// upgradeTouched records modules that performed real upgrade work.
	upgradeTouched map[string]bool
}

func newOpContext() *opContext {
	return &opContext{
		fromVersion:    map[string]string{},
		schemaIntents:  schema.NewMemoryIntentBag(),
		installing:     map[string]bool{},
		installDone:    map[string]bool{},
		installTouched: map[string]bool{},
		uninstalling:   map[string]bool{},
		uninstallDone:  map[string]bool{},
		upgrading:      map[string]bool{},
		upgradeDone:    map[string]bool{},
		upgradeTouched: map[string]bool{},
	}
}

func cyclePath(stack []string, name string) []string {
	idx := slices.Index(stack, name)
	if idx < 0 {
		return nil
	}
	path := append([]string{}, stack[idx:]...)
	path = append(path, name)
	return path
}

func (c *opContext) isInstallDone(name string) bool    { return c.installDone[name] }
func (c *opContext) markInstallDone(name string)       { c.installDone[name] = true }
func (c *opContext) isInstallTouched(name string) bool { return c != nil && c.installTouched[name] }
func (c *opContext) markInstallTouched(name string) {
	if c == nil || name == "" {
		return
	}
	if c.installTouched == nil {
		c.installTouched = map[string]bool{}
	}
	c.installTouched[name] = true
}
func (c *opContext) isUninstallDone(name string) bool  { return c.uninstallDone[name] }
func (c *opContext) markUninstallDone(name string)     { c.uninstallDone[name] = true }
func (c *opContext) isUpgradeDone(name string) bool    { return c.upgradeDone[name] }
func (c *opContext) markUpgradeDone(name string)       { c.upgradeDone[name] = true }
func (c *opContext) isUpgradeTouched(name string) bool { return c != nil && c.upgradeTouched[name] }
func (c *opContext) markUpgradeTouched(name string) {
	if c == nil || name == "" {
		return
	}
	if c.upgradeTouched == nil {
		c.upgradeTouched = map[string]bool{}
	}
	c.upgradeTouched[name] = true
}

// phaseEndCandidates returns modules that should run PhaseEnd for this op.
// Install: only modules that actually installed. Upgrade: upgraded targets plus
// EnsureOrder modules that were newly installed in this op.
func phaseEndCandidates(op plan.OpType, moduleOrder, ensureOrder []string, ctx *opContext) []string {
	switch op {
	case plan.OpInstall:
		out := make([]string, 0, len(moduleOrder))
		for _, name := range moduleOrder {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if ctx == nil || ctx.isInstallTouched(name) {
				out = append(out, name)
			}
		}
		return out
	case plan.OpUpgrade:
		// Capacity is a hint only; avoid len+len so static analyzers do not flag overflow.
		out := make([]string, 0, len(moduleOrder))
		for _, name := range moduleOrder {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if ctx == nil || ctx.isUpgradeTouched(name) {
				out = append(out, name)
			}
		}
		if ctx != nil {
			for _, name := range ensureOrder {
				name = strings.TrimSpace(name)
				if name != "" && ctx.isInstallTouched(name) {
					out = append(out, name)
				}
			}
		}
		return mergeUniqueModuleNames(out)
	default:
		return mergeUniqueModuleNames(moduleOrder)
	}
}

func (c *opContext) setFromVersion(name string, version string) {
	if c == nil || name == "" {
		return
	}
	c.fromVersion[name] = version
}

func (c *opContext) getFromVersion(name string) string {
	if c == nil || name == "" {
		return ""
	}
	return c.fromVersion[name]
}

func (c *opContext) pushInstall(name string) []string {
	if c.installing[name] {
		return cyclePath(c.installStack, name)
	}
	c.installing[name] = true
	c.installStack = append(c.installStack, name)
	return nil
}

func (c *opContext) popInstall(name string) {
	if len(c.installStack) > 0 {
		c.installStack = c.installStack[:len(c.installStack)-1]
	}
	c.installing[name] = false
}

func (c *opContext) pushUninstall(name string) []string {
	if c.uninstalling[name] {
		return cyclePath(c.uninstallStack, name)
	}
	c.uninstalling[name] = true
	c.uninstallStack = append(c.uninstallStack, name)
	return nil
}

func (c *opContext) popUninstall(name string) {
	if len(c.uninstallStack) > 0 {
		c.uninstallStack = c.uninstallStack[:len(c.uninstallStack)-1]
	}
	c.uninstalling[name] = false
}

func (c *opContext) pushUpgrade(name string) []string {
	if c.upgrading[name] {
		return cyclePath(c.upgradeStack, name)
	}
	c.upgrading[name] = true
	c.upgradeStack = append(c.upgradeStack, name)
	return nil
}

func (c *opContext) popUpgrade(name string) {
	if len(c.upgradeStack) > 0 {
		c.upgradeStack = c.upgradeStack[:len(c.upgradeStack)-1]
	}
	c.upgrading[name] = false
}
