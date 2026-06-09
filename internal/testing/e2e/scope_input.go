// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

type runtimeScopeInputOptions struct {
	ModulesPath        string
	DistPath           string
	TmpPath            string
	DefaultChoysumPath string
	ConfigPath         string
	NpmPath            string
	NPMRegistryURL     string
	Log                *config.LogConfig
	Compile            *config.CompileConfig
	Auth               *config.AuthConfig
	Server             *config.ServerConfig
	Task               *config.TaskConfig
	FrontendEnv        map[string]any
	BackendEnv         map[string]any
	Db                 *config.DbConfig
}

func newRuntimeScopeInputOptions(cfg *snapshot.ConfigSnapshot) *runtimeScopeInputOptions {
	snap := cfg
	if snap == nil {
		return nil
	}
	return &runtimeScopeInputOptions{
		ModulesPath:        snap.ModulesPath,
		DistPath:           snap.DistPath,
		TmpPath:            snap.TmpPath,
		DefaultChoysumPath: snap.DefaultChoysumPath,
		ConfigPath:         snap.ConfigPath,
		NpmPath:            snap.NpmPath,
		NPMRegistryURL:     snap.NPMRegistryURL,
		Log:                snap.CopyLogConfig(),
		Compile:            snap.CopyCompileConfig(),
		Auth:               snap.CopyAuthConfig(),
		Server:             snap.CopyServerConfig(),
		Task:               snap.CopyTaskConfig(),
		FrontendEnv:        snap.CopyFrontendEnv(),
		BackendEnv:         snap.CopyBackendEnv(),
		Db:                 cloneRuntimeScopeDbConfig(snap.Db),
	}
}

func (o *runtimeScopeInputOptions) CopyCompileConfig() *config.CompileConfig {
	if o == nil || o.Compile == nil {
		return nil
	}
	cloned := *o.Compile
	return &cloned
}

func (o *runtimeScopeInputOptions) CopyAuthConfig() *config.AuthConfig {
	if o == nil {
		return nil
	}
	return cloneRuntimeScopeAuthConfig(o.Auth)
}

func (o *runtimeScopeInputOptions) CopyAuthHttpAuth() *config.HttpAuthConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.HttpAuth
}

func (o *runtimeScopeInputOptions) CopyAuthJobTokenAllowedSANs() []string {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.JobTokenAllowedSANs
}

func (o *runtimeScopeInputOptions) CopyAuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.GrpcEntryPolicy
}

func (o *runtimeScopeInputOptions) CopyServerConfig() *config.ServerConfig {
	if o == nil {
		return nil
	}
	return cloneRuntimeScopeServerConfig(o.Server)
}

func (o *runtimeScopeInputOptions) CopyServerCSPConfig() *config.CSPConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSP
}

func (o *runtimeScopeInputOptions) CopyServerHSTSConfig() *config.HSTSConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.HSTS
}

func (o *runtimeScopeInputOptions) CopyServerCSRFConfig() *config.CSRFConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSRF
}

func (o *runtimeScopeInputOptions) CopyTaskConfig() *config.TaskConfig {
	if o == nil {
		return nil
	}
	return cloneRuntimeScopeTaskConfig(o.Task)
}

func (o *runtimeScopeInputOptions) CopyLogConfig() *config.LogConfig {
	if o == nil || o.Log == nil {
		return nil
	}
	cloned := *o.Log
	return &cloned
}

func (o *runtimeScopeInputOptions) CopyFrontendEnv() map[string]any {
	if o == nil {
		return nil
	}
	return cloneRuntimeScopeFrontendEnv(o.FrontendEnv)
}

func (o *runtimeScopeInputOptions) CopyBackendEnv() map[string]any {
	if o == nil {
		return nil
	}
	return cloneRuntimeScopeBackendEnv(o.BackendEnv)
}

func cloneRuntimeScopeAuthConfig(authCfg *config.AuthConfig) *config.AuthConfig {
	if authCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Auth: authCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyAuthConfig()
}

func cloneRuntimeScopeServerConfig(serverCfg *config.ServerConfig) *config.ServerConfig {
	if serverCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Server: serverCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyServerConfig()
}

func cloneRuntimeScopeTaskConfig(taskCfg *config.TaskConfig) *config.TaskConfig {
	if taskCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Task: taskCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyTaskConfig()
}

func cloneRuntimeScopeFrontendEnv(frontendEnv map[string]any) map[string]any {
	snap := snapshot.New(&config.Config{FrontendEnv: frontendEnv})
	if snap == nil {
		return nil
	}
	return snap.CopyFrontendEnv()
}

func cloneRuntimeScopeBackendEnv(backendEnv map[string]any) map[string]any {
	snap := snapshot.New(&config.Config{BackendEnv: backendEnv})
	if snap == nil {
		return nil
	}
	return snap.CopyBackendEnv()
}

func cloneRuntimeScopeDbConfig(dbCfg *config.DbConfig) *config.DbConfig {
	if dbCfg == nil {
		return nil
	}
	cloned := *dbCfg
	return &cloned
}

// runtimeScopeInput keeps e2e runtime options attached when constructing scope input.
type runtimeScopeInput struct {
	options        *runtimeScopeInputOptions
	runtimeOptions e2eRuntimeOptions
}

func newRuntimeScopeInput(cfg *snapshot.ConfigSnapshot, runtimeOptions e2eRuntimeOptions) runtimeScopeInput {
	return runtimeScopeInput{options: newRuntimeScopeInputOptions(cfg), runtimeOptions: runtimeOptions}
}

func (i runtimeScopeInput) Environment() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.Environment
}

func (i runtimeScopeInput) ModulesPath() string {
	if i.runtimeOptions.modulesPath != "" {
		return i.runtimeOptions.modulesPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModulesPath
}

func (i runtimeScopeInput) DistPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DistPath
}

func (i runtimeScopeInput) TmpPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.TmpPath
}

func (i runtimeScopeInput) DefaultChoysumPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DefaultChoysumPath
}

func (i runtimeScopeInput) ConfigPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.ConfigPath
}

func (i runtimeScopeInput) NpmPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.NpmPath
}

func (i runtimeScopeInput) NpmRegistryURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.NPMRegistryURL
}

func (i runtimeScopeInput) CompileBundleMode() string {
	if i.options == nil || i.options.Compile == nil {
		return ""
	}
	return i.options.Compile.BundleMode
}

func (i runtimeScopeInput) CompileConfig() *config.CompileConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyCompileConfig()
}

func (i runtimeScopeInput) AuthEnabled() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.Enabled
}

func (i runtimeScopeInput) AuthConfig() *config.AuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthConfig()
}

func (i runtimeScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthHttpAuth()
}

func (i runtimeScopeInput) AuthGrpcAuthentication() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.GrpcAuthentication
}

func (i runtimeScopeInput) AuthInternalKey() string {
	if i.options == nil || i.options.Auth == nil {
		return ""
	}
	return i.options.Auth.InternalKey
}

func (i runtimeScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthJobTokenAllowedSANs()
}

func (i runtimeScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthGrpcEntryPolicy()
}

func (i runtimeScopeInput) ServerEnabledTLS() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnabledTLS
}

func (i runtimeScopeInput) ServerConfig() *config.ServerConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerConfig()
}

func (i runtimeScopeInput) ServerBindAddress() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.BindAddress
}

func (i runtimeScopeInput) ServerPort() int {
	if i.options == nil || i.options.Server == nil {
		return 0
	}
	return i.options.Server.Port
}

func (i runtimeScopeInput) ServerEnableGzip() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGzip
}

func (i runtimeScopeInput) ServerTLSCaFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCaFile
}

func (i runtimeScopeInput) ServerTLSServerName() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSServerName
}

func (i runtimeScopeInput) ServerTLSCertFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCertFile
}

func (i runtimeScopeInput) ServerTLSKeyFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSKeyFile
}

func (i runtimeScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGrpcWebProxy
}

func (i runtimeScopeInput) ServerHotReload() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.HotReload
}

func (i runtimeScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.options == nil || i.options.Server == nil || i.options.Server.GrpcClient == nil {
		return 0
	}
	return i.options.Server.GrpcClient.MaxCachedConns
}

func (i runtimeScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSPConfig()
}

func (i runtimeScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerHSTSConfig()
}

func (i runtimeScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSRFConfig()
}

func (i runtimeScopeInput) ServerSecurityMissing() bool {
	return i.options == nil || i.options.Server == nil || i.options.Server.Security == nil
}

func (i runtimeScopeInput) TaskConfig() *config.TaskConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyTaskConfig()
}

func (i runtimeScopeInput) LogConfig() *config.LogConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyLogConfig()
}

func (i runtimeScopeInput) FrontendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyFrontendEnv()
}

func (i runtimeScopeInput) BackendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyBackendEnv()
}

func (i runtimeScopeInput) DatabaseDialect() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.Dialect
}

func (i runtimeScopeInput) DatabaseDSN() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.DSN
}

func (i runtimeScopeInput) DatabaseMaxOpenConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxOpenConns
}

func (i runtimeScopeInput) DatabaseMaxIdleConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxIdleConns
}

func (i runtimeScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.ConnMaxLifetime
}
