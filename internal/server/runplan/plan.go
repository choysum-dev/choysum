// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"log/slog"
	"strings"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	xfmt "golang.org/x/exp/errors/fmt"
)

type RunMode string

const (
	RunModeApplication RunMode = "application"
	RunModeBootstrap   RunMode = "bootstrap"
)

type RunDecision struct {
	RunMode           RunMode
	CompileBundleMode string
	ServeTargets      []string
	Reason            string
}

func Plan(distRoot string, configBundleMode string, logger *slog.Logger, serviceNames []string) (*distmanifest.DistManifestV2, RunDecision, error) {
	manifest, err := LoadDistManifest(distRoot)
	if err != nil {
		return nil, RunDecision{}, err
	}

	decision, err := buildRunDecision(distRoot, configBundleMode, logger, manifest, serviceNames)
	if err != nil {
		return nil, RunDecision{}, err
	}

	return manifest, decision, nil
}

func buildRunDecision(distRoot string, configBundleMode string, logger *slog.Logger, manifest *distmanifest.DistManifestV2, serviceNames []string) (RunDecision, error) {
	explicitTargets := len(serviceNames) > 0

	compileBundleMode, serveTargets, err := resolveServeTargets(distRoot, configBundleMode, logger, manifest, serviceNames)
	if err != nil {
		if !explicitTargets && isBootstrapFallbackError(err) {
			return RunDecision{
				RunMode:           RunModeBootstrap,
				CompileBundleMode: compileBundleMode,
				Reason:            "required app assets are not ready yet",
			}, nil
		}
		return RunDecision{}, err
	}

	if len(serveTargets) == 0 {
		if !explicitTargets {
			return RunDecision{
				RunMode:           RunModeBootstrap,
				CompileBundleMode: compileBundleMode,
				Reason:            "no app is ready to serve yet",
			}, nil
		}
		return RunDecision{}, xfmt.Errorf("no runnable targets resolved from explicit run arguments")
	}

	if err := ValidateDistForTargets(compileBundleMode, distRoot, serveTargets); err != nil {
		if !explicitTargets && isBootstrapFallbackError(err) {
			return RunDecision{
				RunMode:           RunModeBootstrap,
				CompileBundleMode: compileBundleMode,
				Reason:            "required app assets are not ready yet",
			}, nil
		}
		return RunDecision{}, err
	}

	return RunDecision{
		RunMode:           RunModeApplication,
		CompileBundleMode: compileBundleMode,
		ServeTargets:      serveTargets,
		Reason:            "dist validation passed",
	}, nil
}

func isBootstrapFallbackError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	markers := []string{
		"web dist missing",
		"bundles dir missing",
		"bundles index missing",
		"app index missing",
		"api proto assets missing",
		"app proto assets missing",
	}

	for _, marker := range markers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
