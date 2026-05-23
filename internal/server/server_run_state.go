// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/server/runplan"
)

type runState struct {
	distManifest      *distmanifest.DistManifestV2
	runMode           runplan.RunMode
	runModeReason     string
	compileBundleMode string
	applicationNames  []string
}

type runStateSnapshot struct {
	distManifest      *distmanifest.DistManifestV2
	runMode           runplan.RunMode
	runModeReason     string
	compileBundleMode string
	applicationNames  []string
}

func (r *runState) applyPlannedDecision(manifest *distmanifest.DistManifestV2, decision runplan.RunDecision) {
	r.distManifest = manifest
	r.runMode = decision.RunMode
	r.runModeReason = decision.Reason
	r.compileBundleMode = decision.CompileBundleMode
	r.applicationNames = append([]string{}, decision.ServeTargets...)
}

func (r *runState) applyBootstrapDecision(manifest *distmanifest.DistManifestV2, decision runplan.RunDecision) {
	r.applyPlannedDecision(manifest, decision)
	r.prefixReason("bootstrap switch")
}

func (r *runState) prefixReason(prefix string) {
	if strings.TrimSpace(prefix) == "" {
		return
	}
	if strings.TrimSpace(r.runModeReason) == "" {
		r.runModeReason = prefix
		return
	}
	r.runModeReason = prefix + ": " + r.runModeReason
}

func (r *runState) ensureStartupDefaults(opts runtimeOptions) {
	if r.runMode == "" {
		r.runMode = runplan.RunModeApplication
	}
	if strings.TrimSpace(r.compileBundleMode) == "" {
		r.compileBundleMode = opts.compileBundleMode
	}
}

func (r *runState) isBootstrapMode() bool {
	return r.runMode == runplan.RunModeBootstrap
}

func (r *runState) mode() runplan.RunMode {
	return r.runMode
}

func (r *runState) reason() string {
	return r.runModeReason
}

func (r *runState) bundleMode() string {
	return r.compileBundleMode
}

func (r *runState) serviceTargets() []string {
	return append([]string{}, r.applicationNames...)
}

func (r *runState) serves(target string) bool {
	for _, name := range r.applicationNames {
		if name == target {
			return true
		}
	}
	return false
}

func (r *runState) manifestApp(name string) (distmanifest.DistManifestApp, bool) {
	if r.distManifest == nil {
		return distmanifest.DistManifestApp{}, false
	}
	app, ok := r.distManifest.Apps[name]
	return app, ok
}

func (r *runState) switchToBootstrapService(name string) {
	r.applicationNames = []string{name}
}

func (r *runState) snapshot() runStateSnapshot {
	return runStateSnapshot{
		distManifest:      r.distManifest,
		runMode:           r.runMode,
		runModeReason:     r.runModeReason,
		compileBundleMode: r.compileBundleMode,
		applicationNames:  append([]string{}, r.applicationNames...),
	}
}

func (r *runState) restore(snapshot runStateSnapshot) {
	r.distManifest = snapshot.distManifest
	r.runMode = snapshot.runMode
	r.runModeReason = snapshot.runModeReason
	r.compileBundleMode = snapshot.compileBundleMode
	r.applicationNames = append([]string{}, snapshot.applicationNames...)
}
