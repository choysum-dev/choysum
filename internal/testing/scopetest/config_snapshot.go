// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scopetest

import (
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

// ConfigFromSnapshot bridges a config snapshot into a deep-copied config fixture for tests.
func ConfigFromSnapshot(snap *snapshot.ConfigSnapshot) *config.Config {
	if snap == nil {
		return nil
	}
	return &config.Config{
		ModulesPath:        snap.ModulesPath,
		DistPath:           snap.DistPath,
		TmpPath:            snap.TmpPath,
		DefaultChoysumPath: snap.DefaultChoysumPath,
		ConfigPath:         snap.ConfigPath,
		NpmPath:            snap.NpmPath,
		Compile:            snap.CopyCompileConfig(),
		Auth:               snap.CopyAuthConfig(),
		Server:             snap.CopyServerConfig(),
		Task:               snap.CopyTaskConfig(),
		FrontendEnv:        snap.CopyFrontendEnv(),
		BackendEnv:         snap.CopyBackendEnv(),
		Db:                 cloneConfigSnapshotDbConfig(snap.Db),
	}
}

func cloneConfigSnapshotDbConfig(dbCfg *config.DbConfig) *config.DbConfig {
	if dbCfg == nil {
		return nil
	}
	cloned := *dbCfg
	return &cloned
}
