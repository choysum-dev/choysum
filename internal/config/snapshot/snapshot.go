// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "github.com/choysum-dev/choysum/pkg/config"

// ConfigSnapshot stores a deep-copied runtime config projection for scope input adapters.
type ConfigSnapshot struct {
	ModulesPath                          string
	DistPath                             string
	TmpPath                              string
	DefaultChoysumPath                   string
	ConfigPath                           string
	NPMRegistryURL                       string
	ModuleCatalogIndexURL                string
	BootstrapModuleInstallTimeoutSeconds int
	ESMUpstreamURL                       string
	Log                                  *config.LogConfig
	Compile                              *config.CompileConfig
	Auth                                 *config.AuthConfig
	Server                               *config.ServerConfig
	Task                                 *config.TaskConfig
	FrontendEnv                          map[string]any
	BackendEnv                           map[string]any
	Db                                   *config.DbConfig
}

// New deep-copies the provided config into a runtime-safe snapshot.
func New(cfg *config.Config) *ConfigSnapshot {
	if cfg == nil {
		return nil
	}
	return &ConfigSnapshot{
		ModulesPath:                          cfg.ModulesPath,
		DistPath:                             cfg.DistPath,
		TmpPath:                              cfg.TmpPath,
		DefaultChoysumPath:                   cfg.DefaultChoysumPath,
		ConfigPath:                           cfg.ConfigPath,
		NPMRegistryURL:                       cfg.NPMRegistryURL,
		ModuleCatalogIndexURL:                cfg.ModuleCatalogIndexURL,
		BootstrapModuleInstallTimeoutSeconds: cfg.BootstrapModuleInstallTimeoutSeconds,
		ESMUpstreamURL:                       cfg.ESMUpstreamURL,
		Log:                                  cloneLogConfig(cfg.Log),
		Compile:                              cloneCompileConfig(cfg.Compile),
		Auth:                                 cloneAuthConfig(cfg.Auth),
		Server:                               cloneServerConfig(cfg.Server),
		Task:                                 cloneTaskConfig(cfg.Task),
		FrontendEnv:                          cloneAnyMap(cfg.FrontendEnv),
		BackendEnv:                           cloneAnyMap(cfg.BackendEnv),
		Db:                                   cloneDbConfig(cfg.Db),
	}
}

// CopyLogConfig returns a deep copy of the log config.
func (s *ConfigSnapshot) CopyLogConfig() *config.LogConfig {
	if s == nil {
		return nil
	}
	return cloneLogConfig(s.Log)
}

// CopyCompileConfig returns a deep copy of the compile config.
func (s *ConfigSnapshot) CopyCompileConfig() *config.CompileConfig {
	if s == nil {
		return nil
	}
	return cloneCompileConfig(s.Compile)
}

func cloneLogConfig(cfg *config.LogConfig) *config.LogConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

// CopyAuthConfig returns a deep copy of the auth config.
func (s *ConfigSnapshot) CopyAuthConfig() *config.AuthConfig {
	if s == nil {
		return nil
	}
	return cloneAuthConfig(s.Auth)
}

// CopyAuthHttpAuth returns a deep copy of the nested HTTP auth config.
func (s *ConfigSnapshot) CopyAuthHttpAuth() *config.HttpAuthConfig {
	if s == nil || s.Auth == nil {
		return nil
	}
	return cloneHttpAuthConfig(s.Auth.HttpAuth)
}

// CopyAuthJobTokenAllowedSANs returns a copy of the configured job-token SAN allowlist.
func (s *ConfigSnapshot) CopyAuthJobTokenAllowedSANs() []string {
	if s == nil || s.Auth == nil {
		return nil
	}
	return cloneStringSlice(s.Auth.JobTokenAllowedSANs)
}

// CopyAuthGrpcEntryPolicy returns a deep copy of the gRPC entry policy map.
func (s *ConfigSnapshot) CopyAuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if s == nil || s.Auth == nil {
		return nil
	}
	return cloneEntryMethodConfigMap(s.Auth.GrpcEntryPolicy)
}

// CopyServerConfig returns a deep copy of the server config.
func (s *ConfigSnapshot) CopyServerConfig() *config.ServerConfig {
	if s == nil {
		return nil
	}
	return cloneServerConfig(s.Server)
}

// CopyServerCSPConfig returns a deep copy of the server CSP config.
func (s *ConfigSnapshot) CopyServerCSPConfig() *config.CSPConfig {
	if s == nil || s.Server == nil || s.Server.Security == nil {
		return nil
	}
	return cloneCSPConfig(s.Server.Security.CSP)
}

// CopyServerHSTSConfig returns a deep copy of the server HSTS config.
func (s *ConfigSnapshot) CopyServerHSTSConfig() *config.HSTSConfig {
	if s == nil || s.Server == nil || s.Server.Security == nil {
		return nil
	}
	return cloneHSTSConfig(s.Server.Security.HSTS)
}

// CopyServerCSRFConfig returns a deep copy of the server CSRF config.
func (s *ConfigSnapshot) CopyServerCSRFConfig() *config.CSRFConfig {
	if s == nil || s.Server == nil || s.Server.Security == nil {
		return nil
	}
	return cloneCSRFConfig(s.Server.Security.CSRF)
}

// CopyTaskConfig returns a deep copy of the task config.
func (s *ConfigSnapshot) CopyTaskConfig() *config.TaskConfig {
	if s == nil {
		return nil
	}
	return cloneTaskConfig(s.Task)
}

// CopyFrontendEnv returns a deep copy of the frontend env map.
func (s *ConfigSnapshot) CopyFrontendEnv() map[string]any {
	if s == nil {
		return nil
	}
	return cloneAnyMap(s.FrontendEnv)
}

// CopyBackendEnv returns a deep copy of the backend env map.
func (s *ConfigSnapshot) CopyBackendEnv() map[string]any {
	if s == nil {
		return nil
	}
	return cloneAnyMap(s.BackendEnv)
}
