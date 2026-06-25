// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
)

type scopeInputConfigOptions struct {
	ModulesPath                          string
	DistPath                             string
	TmpPath                              string
	DefaultChoysumPath                   string
	ConfigPath                           string
	NPMRegistryURL                       string
	ModuleCatalogIndexURL                string
	ModuleInstallRegistryFallbackEnabled bool
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

func newScopeInputConfigOptions(snap *snapshot.ConfigSnapshot) *scopeInputConfigOptions {
	if snap == nil {
		return nil
	}
	return &scopeInputConfigOptions{
		ModulesPath:                          snap.ModulesPath,
		DistPath:                             snap.DistPath,
		TmpPath:                              snap.TmpPath,
		DefaultChoysumPath:                   snap.DefaultChoysumPath,
		ConfigPath:                           snap.ConfigPath,
		NPMRegistryURL:                       snap.NPMRegistryURL,
		ModuleCatalogIndexURL:                snap.ModuleCatalogIndexURL,
		ModuleInstallRegistryFallbackEnabled: snap.ModuleInstallRegistryFallbackEnabled,
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

func (o *scopeInputConfigOptions) CopyCompileConfig() *config.CompileConfig {
	if o == nil || o.Compile == nil {
		return nil
	}
	cloned := *o.Compile
	return &cloned
}

func (o *scopeInputConfigOptions) CopyAuthConfig() *config.AuthConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputAuthConfig(o.Auth)
}

func (o *scopeInputConfigOptions) CopyAuthHttpAuth() *config.HttpAuthConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.HttpAuth
}

func (o *scopeInputConfigOptions) CopyAuthJobTokenAllowedSANs() []string {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.JobTokenAllowedSANs
}

func (o *scopeInputConfigOptions) CopyAuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	authCfg := o.CopyAuthConfig()
	if authCfg == nil {
		return nil
	}
	return authCfg.GrpcEntryPolicy
}

func (o *scopeInputConfigOptions) CopyServerConfig() *config.ServerConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputServerConfig(o.Server)
}

func (o *scopeInputConfigOptions) CopyServerCSPConfig() *config.CSPConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSP
}

func (o *scopeInputConfigOptions) CopyServerHSTSConfig() *config.HSTSConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.HSTS
}

func (o *scopeInputConfigOptions) CopyServerCSRFConfig() *config.CSRFConfig {
	serverCfg := o.CopyServerConfig()
	if serverCfg == nil || serverCfg.Security == nil {
		return nil
	}
	return serverCfg.Security.CSRF
}

func (o *scopeInputConfigOptions) CopyTaskConfig() *config.TaskConfig {
	if o == nil {
		return nil
	}
	return cloneScopeInputTaskConfig(o.Task)
}

func (o *scopeInputConfigOptions) CopyLogConfig() *config.LogConfig {
	if o == nil || o.Log == nil {
		return nil
	}
	cloned := *o.Log
	return &cloned
}

func (o *scopeInputConfigOptions) CopyFrontendEnv() map[string]any {
	if o == nil {
		return nil
	}
	return cloneScopeInputFrontendEnv(o.FrontendEnv)
}

func (o *scopeInputConfigOptions) CopyBackendEnv() map[string]any {
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

// commandRuntimeScopeInput binds validated CLI runtime options to scope creation.
type commandRuntimeScopeInput struct {
	options        *scopeInputConfigOptions
	runtimeOptions cliRuntimeOptions
}

func newCommandRuntimeScopeInput(options *scopeInputConfigOptions, runtimeOptions cliRuntimeOptions) commandRuntimeScopeInput {
	return commandRuntimeScopeInput{options: options, runtimeOptions: runtimeOptions}
}

func (i commandRuntimeScopeInput) Environment() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.Environment
}

func (i commandRuntimeScopeInput) ModulesPath() string {
	if strings.TrimSpace(i.runtimeOptions.modulesPath) != "" {
		return i.runtimeOptions.modulesPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModulesPath
}

func (i commandRuntimeScopeInput) DistPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DistPath
}

func (i commandRuntimeScopeInput) TmpPath() string {
	if strings.TrimSpace(i.runtimeOptions.tmpPath) != "" {
		return i.runtimeOptions.tmpPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.TmpPath
}

func (i commandRuntimeScopeInput) DefaultChoysumPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DefaultChoysumPath
}

func (i commandRuntimeScopeInput) ConfigPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.ConfigPath
}

func (i commandRuntimeScopeInput) ESMUpstreamURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ESMUpstreamURL
}

func (i commandRuntimeScopeInput) NpmRegistryURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.NPMRegistryURL
}

func (i commandRuntimeScopeInput) ModuleCatalogIndexURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ModuleCatalogIndexURL
}

func (i commandRuntimeScopeInput) ModuleInstallRegistryFallbackEnabled() bool {
	if i.options == nil {
		return true
	}
	return i.options.ModuleInstallRegistryFallbackEnabled
}

func (i commandRuntimeScopeInput) CompileBundleMode() string {
	if i.options == nil || i.options.Compile == nil {
		return ""
	}
	return i.options.Compile.BundleMode
}

func (i commandRuntimeScopeInput) CompileConfig() *config.CompileConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyCompileConfig()
}

func (i commandRuntimeScopeInput) AuthEnabled() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.Enabled
}

func (i commandRuntimeScopeInput) AuthConfig() *config.AuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthConfig()
}

func (i commandRuntimeScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthHttpAuth()
}

func (i commandRuntimeScopeInput) AuthGrpcAuthentication() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.GrpcAuthentication
}

func (i commandRuntimeScopeInput) AuthInternalKey() string {
	if i.options == nil || i.options.Auth == nil {
		return ""
	}
	return i.options.Auth.InternalKey
}

func (i commandRuntimeScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthJobTokenAllowedSANs()
}

func (i commandRuntimeScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthGrpcEntryPolicy()
}

func (i commandRuntimeScopeInput) ServerEnabledTLS() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnabledTLS
}

func (i commandRuntimeScopeInput) ServerConfig() *config.ServerConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerConfig()
}

func (i commandRuntimeScopeInput) ServerBindAddress() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.BindAddress
}

func (i commandRuntimeScopeInput) ServerPort() int {
	if i.options == nil || i.options.Server == nil {
		return 0
	}
	return i.options.Server.Port
}

func (i commandRuntimeScopeInput) ServerEnableGzip() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGzip
}

func (i commandRuntimeScopeInput) ServerTLSCaFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCaFile
}

func (i commandRuntimeScopeInput) ServerTLSServerName() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSServerName
}

func (i commandRuntimeScopeInput) ServerTLSCertFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCertFile
}

func (i commandRuntimeScopeInput) ServerTLSKeyFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSKeyFile
}

func (i commandRuntimeScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGrpcWebProxy
}

func (i commandRuntimeScopeInput) ServerHotReload() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.HotReload
}

func (i commandRuntimeScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.options == nil || i.options.Server == nil || i.options.Server.GrpcClient == nil {
		return 0
	}
	return i.options.Server.GrpcClient.MaxCachedConns
}

func (i commandRuntimeScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSPConfig()
}

func (i commandRuntimeScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerHSTSConfig()
}

func (i commandRuntimeScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSRFConfig()
}

func (i commandRuntimeScopeInput) ServerSecurityMissing() bool {
	return i.options == nil || i.options.Server == nil || i.options.Server.Security == nil
}

func (i commandRuntimeScopeInput) TaskConfig() *config.TaskConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyTaskConfig()
}

func (i commandRuntimeScopeInput) LogConfig() *config.LogConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyLogConfig()
}

func (i commandRuntimeScopeInput) FrontendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyFrontendEnv()
}

func (i commandRuntimeScopeInput) BackendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyBackendEnv()
}

func (i commandRuntimeScopeInput) DatabaseDialect() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.Dialect
}

func (i commandRuntimeScopeInput) DatabaseDSN() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.DSN
}

func (i commandRuntimeScopeInput) DatabaseMaxOpenConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxOpenConns
}

func (i commandRuntimeScopeInput) DatabaseMaxIdleConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxIdleConns
}

func (i commandRuntimeScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.ConnMaxLifetime
}

// runRuntimeScopeInput carries run startup options into scope creation.
type runRuntimeScopeInput struct {
	options       *scopeInputConfigOptions
	cliOptions    cliRuntimeOptions
	serverOptions runServerRuntimeOptions
	dbOptions     runDBRuntimeOptions
}

func newRunRuntimeScopeInput(options *scopeInputConfigOptions, cliOptions cliRuntimeOptions, serverOptions runServerRuntimeOptions, dbOptions runDBRuntimeOptions) runRuntimeScopeInput {
	return runRuntimeScopeInput{
		options:       options,
		cliOptions:    cliOptions,
		serverOptions: serverOptions,
		dbOptions:     dbOptions,
	}
}

func (i runRuntimeScopeInput) Environment() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.Environment
}

func (i runRuntimeScopeInput) ModulesPath() string {
	if strings.TrimSpace(i.cliOptions.modulesPath) != "" {
		return i.cliOptions.modulesPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModulesPath
}

func (i runRuntimeScopeInput) DistPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DistPath
}

func (i runRuntimeScopeInput) TmpPath() string {
	if strings.TrimSpace(i.cliOptions.tmpPath) != "" {
		return i.cliOptions.tmpPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.TmpPath
}

func (i runRuntimeScopeInput) DefaultChoysumPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DefaultChoysumPath
}

func (i runRuntimeScopeInput) ConfigPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.ConfigPath
}

func (i runRuntimeScopeInput) ESMUpstreamURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ESMUpstreamURL
}

func (i runRuntimeScopeInput) NpmRegistryURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.NPMRegistryURL
}

func (i runRuntimeScopeInput) ModuleCatalogIndexURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ModuleCatalogIndexURL
}

func (i runRuntimeScopeInput) ModuleInstallRegistryFallbackEnabled() bool {
	if i.options == nil {
		return true
	}
	return i.options.ModuleInstallRegistryFallbackEnabled
}

func (i runRuntimeScopeInput) CompileBundleMode() string {
	if i.options == nil || i.options.Compile == nil {
		return ""
	}
	return i.options.Compile.BundleMode
}

func (i runRuntimeScopeInput) CompileConfig() *config.CompileConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyCompileConfig()
}

func (i runRuntimeScopeInput) AuthEnabled() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.Enabled
}

func (i runRuntimeScopeInput) AuthConfig() *config.AuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthConfig()
}

func (i runRuntimeScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthHttpAuth()
}

func (i runRuntimeScopeInput) AuthGrpcAuthentication() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.GrpcAuthentication
}

func (i runRuntimeScopeInput) AuthInternalKey() string {
	if i.options == nil || i.options.Auth == nil {
		return ""
	}
	return i.options.Auth.InternalKey
}

func (i runRuntimeScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthJobTokenAllowedSANs()
}

func (i runRuntimeScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthGrpcEntryPolicy()
}

func (i runRuntimeScopeInput) ServerEnabledTLS() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnabledTLS
}

func (i runRuntimeScopeInput) ServerConfig() *config.ServerConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerConfig()
}

func (i runRuntimeScopeInput) ServerBindAddress() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.BindAddress
}

func (i runRuntimeScopeInput) ServerPort() int {
	if i.options == nil || i.options.Server == nil {
		return 0
	}
	return i.options.Server.Port
}

func (i runRuntimeScopeInput) ServerEnableGzip() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGzip
}

func (i runRuntimeScopeInput) ServerTLSCaFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCaFile
}

func (i runRuntimeScopeInput) ServerTLSServerName() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSServerName
}

func (i runRuntimeScopeInput) ServerTLSCertFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCertFile
}

func (i runRuntimeScopeInput) ServerTLSKeyFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSKeyFile
}

func (i runRuntimeScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGrpcWebProxy
}

func (i runRuntimeScopeInput) ServerHotReload() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.HotReload
}

func (i runRuntimeScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.options == nil || i.options.Server == nil || i.options.Server.GrpcClient == nil {
		return 0
	}
	return i.options.Server.GrpcClient.MaxCachedConns
}

func (i runRuntimeScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSPConfig()
}

func (i runRuntimeScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerHSTSConfig()
}

func (i runRuntimeScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSRFConfig()
}

func (i runRuntimeScopeInput) ServerSecurityMissing() bool {
	return i.options == nil || i.options.Server == nil || i.options.Server.Security == nil
}

func (i runRuntimeScopeInput) TaskConfig() *config.TaskConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyTaskConfig()
}

func (i runRuntimeScopeInput) LogConfig() *config.LogConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyLogConfig()
}

func (i runRuntimeScopeInput) FrontendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyFrontendEnv()
}

func (i runRuntimeScopeInput) BackendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyBackendEnv()
}

func (i runRuntimeScopeInput) DatabaseDialect() string {
	if strings.TrimSpace(i.dbOptions.dialect) != "" {
		return i.dbOptions.dialect
	}
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.Dialect
}

func (i runRuntimeScopeInput) DatabaseDSN() string {
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.DSN
}

func (i runRuntimeScopeInput) DatabaseMaxOpenConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxOpenConns
}

func (i runRuntimeScopeInput) DatabaseMaxIdleConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxIdleConns
}

func (i runRuntimeScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.ConnMaxLifetime
}
