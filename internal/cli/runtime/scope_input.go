// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

type ScopeInputConfigOptions struct {
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

func NewScopeInputConfigOptions(snap *snapshot.ConfigSnapshot) *ScopeInputConfigOptions {
	if snap == nil {
		return nil
	}
	return &ScopeInputConfigOptions{
		ModulesPath:                          snap.ModulesPath,
		DistPath:                             snap.DistPath,
		TmpPath:                              snap.TmpPath,
		DefaultChoysumPath:                   snap.DefaultChoysumPath,
		ConfigPath:                           snap.ConfigPath,
		NPMRegistryURL:                       snap.NPMRegistryURL,
		ModuleCatalogIndexURL:                snap.ModuleCatalogIndexURL,
		BootstrapModuleInstallTimeoutSeconds: snap.BootstrapModuleInstallTimeoutSeconds,
		ESMUpstreamURL:                       snap.ESMUpstreamURL,
		Log:                                  snap.CopyLogConfig(),
		Compile:                              snap.CopyCompileConfig(),
		Auth:                                 snap.CopyAuthConfig(),
		Server:                               snap.CopyServerConfig(),
		Task:                                 snap.CopyTaskConfig(),
		FrontendEnv:                          snap.CopyFrontendEnv(),
		BackendEnv:                           snap.CopyBackendEnv(),
		Db:                                   cloneScopeInputDbConfig(snap.Db),
	}
}

func (o *ScopeInputConfigOptions) CopyCompileConfig() *config.CompileConfig {
	if o == nil || o.Compile == nil {
		return nil
	}
	cloned := *o.Compile
	return &cloned
}

func (o *ScopeInputConfigOptions) CopyAuthConfig() *config.AuthConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputAuthConfig(o.Auth)
}

func (o *ScopeInputConfigOptions) CopyAuthHttpAuth() *config.HttpAuthConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.HttpAuth
}

func (o *ScopeInputConfigOptions) CopyAuthJobTokenAllowedSANs() []string {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.JobTokenAllowedSANs
}

func (o *ScopeInputConfigOptions) CopyAuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.GrpcEntryPolicy
}

func (o *ScopeInputConfigOptions) CopyServerConfig() *config.ServerConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputServerConfig(o.Server)
}

func (o *ScopeInputConfigOptions) CopyServerCSPConfig() *config.CSPConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSP
}

func (o *ScopeInputConfigOptions) CopyServerHSTSConfig() *config.HSTSConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.HSTS
}

func (o *ScopeInputConfigOptions) CopyServerCSRFConfig() *config.CSRFConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSRF
}

func (o *ScopeInputConfigOptions) CopyTaskConfig() *config.TaskConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputTaskConfig(o.Task)
}

func (o *ScopeInputConfigOptions) CopyLogConfig() *config.LogConfig {
	if o == nil || o.Log == nil {
		return nil
	}
	cloned := *o.Log
	return &cloned
}

func (o *ScopeInputConfigOptions) CopyFrontendEnv() map[string]any {
	if o == nil {
		return nil
	}
	return cloneScopeInputFrontendEnv(o.FrontendEnv)
}

func (o *ScopeInputConfigOptions) CopyBackendEnv() map[string]any {
	if o == nil {
		return nil
	}
	return cloneScopeInputBackendEnv(o.BackendEnv)
}

func cloneScopeInputAuthConfig(authCfg *config.AuthConfig) *config.AuthConfig {
	if authCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Auth: authCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyAuthConfig()
}

func cloneScopeInputServerConfig(serverCfg *config.ServerConfig) *config.ServerConfig {
	if serverCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Server: serverCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyServerConfig()
}

func cloneScopeInputTaskConfig(taskCfg *config.TaskConfig) *config.TaskConfig {
	if taskCfg == nil {
		return nil
	}
	snap := snapshot.New(&config.Config{Task: taskCfg})
	if snap == nil {
		return nil
	}
	return snap.CopyTaskConfig()
}

func cloneScopeInputFrontendEnv(frontendEnv map[string]any) map[string]any {
	snap := snapshot.New(&config.Config{FrontendEnv: frontendEnv})
	if snap == nil {
		return nil
	}
	return snap.CopyFrontendEnv()
}

func cloneScopeInputBackendEnv(backendEnv map[string]any) map[string]any {
	snap := snapshot.New(&config.Config{BackendEnv: backendEnv})
	if snap == nil {
		return nil
	}
	return snap.CopyBackendEnv()
}

func cloneScopeInputDbConfig(dbCfg *config.DbConfig) *config.DbConfig {
	if dbCfg == nil {
		return nil
	}
	cloned := *dbCfg
	return &cloned
}

type CommandScopeInput struct {
	options        *ScopeInputConfigOptions
	runtimeOptions Options
}

func NewCommandScopeInput(options *ScopeInputConfigOptions, runtimeOptions Options) CommandScopeInput {
	return CommandScopeInput{options: options, runtimeOptions: runtimeOptions}
}

func (i CommandScopeInput) Environment() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.Environment
}

func (i CommandScopeInput) ModulesPath() string {
	if strings.TrimSpace(i.runtimeOptions.ModulesPath) != "" {
		return i.runtimeOptions.ModulesPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModulesPath
}

func (i CommandScopeInput) DistPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DistPath
}

func (i CommandScopeInput) TmpPath() string {
	if strings.TrimSpace(i.runtimeOptions.TmpPath) != "" {
		return i.runtimeOptions.TmpPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.TmpPath
}

func (i CommandScopeInput) DefaultChoysumPath() string {
	if strings.TrimSpace(i.runtimeOptions.DefaultChoysumPath) != "" {
		return i.runtimeOptions.DefaultChoysumPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.DefaultChoysumPath
}

func (i CommandScopeInput) ConfigPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.ConfigPath
}

func (i CommandScopeInput) ESMUpstreamURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ESMUpstreamURL
}

func (i CommandScopeInput) NpmRegistryURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.NPMRegistryURL
}

func (i CommandScopeInput) ModuleCatalogIndexURL() string {
	if strings.TrimSpace(i.runtimeOptions.ModuleCatalogIndexURL) != "" {
		return i.runtimeOptions.ModuleCatalogIndexURL
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModuleCatalogIndexURL
}

func (i CommandScopeInput) BootstrapModuleInstallTimeoutSeconds() int {
	if i.options == nil {
		return 0
	}
	return i.options.BootstrapModuleInstallTimeoutSeconds
}

func (i CommandScopeInput) CompileBundleMode() string {
	if i.options == nil || i.options.Compile == nil {
		return ""
	}
	return i.options.Compile.BundleMode
}

func (i CommandScopeInput) CompileConfig() *config.CompileConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyCompileConfig()
}

func (i CommandScopeInput) AuthEnabled() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.Enabled
}

func (i CommandScopeInput) AuthConfig() *config.AuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthConfig()
}

func (i CommandScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthHttpAuth()
}

func (i CommandScopeInput) AuthGrpcAuthentication() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.GrpcAuthentication
}

func (i CommandScopeInput) AuthInternalKey() string {
	if i.options == nil || i.options.Auth == nil {
		return ""
	}
	return i.options.Auth.InternalKey
}

func (i CommandScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthJobTokenAllowedSANs()
}

func (i CommandScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthGrpcEntryPolicy()
}

func (i CommandScopeInput) ServerEnabledTLS() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnabledTLS
}

func (i CommandScopeInput) ServerConfig() *config.ServerConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerConfig()
}

func (i CommandScopeInput) ServerBindAddress() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.BindAddress
}

func (i CommandScopeInput) ServerPort() int {
	if i.options == nil || i.options.Server == nil {
		return 0
	}
	return i.options.Server.Port
}

func (i CommandScopeInput) ServerEnableGzip() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGzip
}

func (i CommandScopeInput) ServerTLSCaFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCaFile
}

func (i CommandScopeInput) ServerTLSServerName() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSServerName
}

func (i CommandScopeInput) ServerTLSCertFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCertFile
}

func (i CommandScopeInput) ServerTLSKeyFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSKeyFile
}

func (i CommandScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGrpcWebProxy
}

func (i CommandScopeInput) ServerHotReload() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.HotReload
}

func (i CommandScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.options == nil || i.options.Server == nil || i.options.Server.GrpcClient == nil {
		return 0
	}
	return i.options.Server.GrpcClient.MaxCachedConns
}

func (i CommandScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSPConfig()
}

func (i CommandScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerHSTSConfig()
}

func (i CommandScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSRFConfig()
}

func (i CommandScopeInput) ServerSecurityMissing() bool {
	return i.options == nil || i.options.Server == nil || i.options.Server.Security == nil
}

func (i CommandScopeInput) TaskConfig() *config.TaskConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyTaskConfig()
}

func (i CommandScopeInput) LogConfig() *config.LogConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyLogConfig()
}

func (i CommandScopeInput) FrontendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyFrontendEnv()
}

func (i CommandScopeInput) BackendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyBackendEnv()
}

func (i CommandScopeInput) DatabaseDialect() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.Dialect
}

func (i CommandScopeInput) DatabaseDSN() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.DSN
}

func (i CommandScopeInput) DatabaseMaxOpenConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxOpenConns
}

func (i CommandScopeInput) DatabaseMaxIdleConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxIdleConns
}

func (i CommandScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.ConnMaxLifetime
}
