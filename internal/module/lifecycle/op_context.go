// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import "slices"

type opContext struct {
	withDemo bool
	opid     string

	fromVersion map[string]string

	installStack []string
	installing   map[string]bool
	installDone  map[string]bool

	uninstallStack []string
	uninstalling   map[string]bool
	uninstallDone  map[string]bool

	upgradeStack []string
	upgrading    map[string]bool
	upgradeDone  map[string]bool
}

func newOpContext() *opContext {
	return &opContext{
		fromVersion:   map[string]string{},
		installing:    map[string]bool{},
		installDone:   map[string]bool{},
		uninstalling:  map[string]bool{},
		uninstallDone: map[string]bool{},
		upgrading:     map[string]bool{},
		upgradeDone:   map[string]bool{},
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

func (c *opContext) isInstallDone(name string) bool   { return c.installDone[name] }
func (c *opContext) markInstallDone(name string)      { c.installDone[name] = true }
func (c *opContext) isUninstallDone(name string) bool { return c.uninstallDone[name] }
func (c *opContext) markUninstallDone(name string)    { c.uninstallDone[name] = true }
func (c *opContext) isUpgradeDone(name string) bool   { return c.upgradeDone[name] }
func (c *opContext) markUpgradeDone(name string)      { c.upgradeDone[name] = true }

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
