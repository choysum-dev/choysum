// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"

	semver "github.com/Masterminds/semver/v3"
	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

func CheckExternalDependencies(module *meta.IrModule) error {
	isBinaryDependencyInstalled := func(cmd, version string) error {
		cmd = regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(cmd, "")
		if cmd == "" {
			return fmt.Errorf("binary dependency command is empty")
		}

		command := exec.Command(cmd, "--version")
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to execute %s --version: %w", cmd, err)
		}

		versionStr := string(output)
		re := regexp.MustCompile(`\d+\.\d+\.\d+`)
		match := re.FindString(versionStr)
		if match == "" {
			return fmt.Errorf("invalid version format from %s: %s", cmd, versionStr)
		}

		v1, err := semver.NewVersion(match)
		if err != nil {
			return fmt.Errorf("failed to parse version %s: %w", match, err)
		}

		constraint, err := semver.NewConstraint(version)
		if err != nil {
			return fmt.Errorf("invalid version constraint %s: %w", version, err)
		}

		if !constraint.Check(v1) {
			return fmt.Errorf("%s version %s does not satisfy requirement %s", cmd, v1, version)
		}

		return nil
	}

	if module == nil || len(module.ExternalDependencies) == 0 {
		return nil
	}
	externalDependencies := make(map[string]map[string]string)
	if err := json.Unmarshal(module.ExternalDependencies, &externalDependencies); err != nil {
		return xfmt.Errorf("error unmarshal external dependencies: %w", err)
	}

	for name, version := range externalDependencies["binary"] {
		if err := isBinaryDependencyInstalled(name, version); err != nil {
			return xfmt.Errorf("binary: %s not installed: %w", name, err)
		}
	}
	return nil
}
