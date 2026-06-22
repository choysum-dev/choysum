// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package semantics

import (
	"fmt"
	"strings"
)

const NoTestsFoundMessage = "no tests found"

func PrefixForCommand(commandName string, message string) string {
	commandName = strings.TrimSpace(commandName)
	if commandName == "" {
		commandName = "command"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return commandName
	}
	return fmt.Sprintf("%s: %s", commandName, message)
}

func InvalidRuntimeOptionsMessage(commandName string) string {
	return PrefixForCommand(commandName, "invalid runtime options")
}

func UnknownAppMessage(app string) string {
	return fmt.Sprintf("unknown app %q", strings.TrimSpace(app))
}

func UnknownModuleMessage(moduleName string, modulesPath string) string {
	return fmt.Sprintf("unknown module %q (no package.json under %s)", strings.TrimSpace(moduleName), strings.TrimSpace(modulesPath))
}

func ModuleNoE2ESpecsMessage(moduleName string) string {
	return fmt.Sprintf("module %q has no package.json choysum.e2e.specs", strings.TrimSpace(moduleName))
}

func NoRunnableE2EModulesMessage(modulesPath string) string {
	return fmt.Sprintf("e2e: no runnable modules found under %s", strings.TrimSpace(modulesPath))
}

func IsModuleNoE2ESpecsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "has no package.json choysum.e2e.specs")
}
