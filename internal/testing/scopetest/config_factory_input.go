// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scopetest

import (
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type configFactoryInput struct {
	cfg *config.Config
}

// FactoryInputFromConfig bridges a config fixture into scope.FactoryInput for tests.
func FactoryInputFromConfig(cfg *config.Config) scope.FactoryInput {
	if cfg == nil {
		return nil
	}
	return configFactoryInput{cfg: cfg}
}

func (i configFactoryInput) Environment() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.Environment
}

func (i configFactoryInput) ModulesPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ModulesPath
}

func (i configFactoryInput) DistPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DistPath
}

func (i configFactoryInput) TmpPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.TmpPath
}

func (i configFactoryInput) DefaultChoysumPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DefaultChoysumPath
}

func (i configFactoryInput) ConfigPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ConfigPath
}

func (i configFactoryInput) ESMUpstreamURL() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ESMUpstreamURL
}

func (i configFactoryInput) NpmRegistryURL() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.NPMRegistryURL
}

func (i configFactoryInput) ModuleCatalogIndexURL() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ModuleCatalogIndexURL
}

func (i configFactoryInput) CompileConfig() *config.CompileConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Compile
}

func (i configFactoryInput) AuthConfig() *config.AuthConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Auth
}

func (i configFactoryInput) ServerConfig() *config.ServerConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Server
}

func (i configFactoryInput) TaskConfig() *config.TaskConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Task
}

func (i configFactoryInput) LogConfig() *config.LogConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Log
}

func (i configFactoryInput) DocumentConfig() *config.DocumentConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.Document
}

func (i configFactoryInput) FrontendEnv() map[string]any {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.FrontendEnv
}

func (i configFactoryInput) BackendEnv() map[string]any {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.BackendEnv
}

func (i configFactoryInput) DatabaseDialect() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.Dialect
}

func (i configFactoryInput) DatabaseDSN() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.DSN
}

func (i configFactoryInput) DatabaseMaxOpenConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxOpenConns
}

func (i configFactoryInput) DatabaseMaxIdleConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxIdleConns
}

func (i configFactoryInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.ConnMaxLifetime
}
