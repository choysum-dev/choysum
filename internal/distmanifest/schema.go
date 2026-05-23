// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package distmanifest

// DistManifestFileName is the filename used for generated dist manifest snapshots.
const DistManifestFileName = "choysum.dist.json"

// SchemaVersion is the current dist manifest schema version.
const SchemaVersion = 2

// DistManifestV2 defines the on-disk dist manifest schema shared by build and runtime owners.
type DistManifestV2 struct {
	SchemaVersion     int                        `json:"schemaVersion"`
	GeneratedAt       string                     `json:"generatedAt"`
	CompileBundleMode string                     `json:"compileBundleMode"`
	HasWeb            bool                       `json:"hasWeb"`
	BackendTopoOrder  []string                   `json:"backendTopoOrder"`
	Apps              map[string]DistManifestApp `json:"apps"`
}

// DistManifestApp stores per-application dependency metadata in the dist manifest.
type DistManifestApp struct {
	Deps DistManifestAppDeps `json:"deps"`
	Dev  DistManifestAppDev  `json:"dev"`
}

// DistManifestAppDeps stores runtime dependency data for an application entry.
type DistManifestAppDeps struct {
	Apps []string `json:"apps"`
}

// DistManifestAppDev stores development-only metadata for an application entry.
type DistManifestAppDev struct {
	Modules []string `json:"modules"`
}
