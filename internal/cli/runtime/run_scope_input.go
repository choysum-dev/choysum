// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	xfmt "golang.org/x/exp/errors/fmt"
)

type RunServerOptions struct {
	BindAddress string
	Port        int
	EnabledTLS  bool
}

func NewRunServerOptions(serverCfg *config.ServerConfig) RunServerOptions {
	defaults := config.NewDefaultServerConfig()
	options := RunServerOptions{
		BindAddress: defaults.BindAddress,
		Port:        defaults.Port,
		EnabledTLS:  defaults.EnabledTLS,
	}
	if serverCfg == nil {
		return options
	}
	if strings.TrimSpace(serverCfg.BindAddress) != "" {
		options.BindAddress = serverCfg.BindAddress
	}
	if serverCfg.Port > 0 {
		options.Port = serverCfg.Port
	}
	options.EnabledTLS = serverCfg.EnabledTLS
	return options
}

func (o RunServerOptions) Validate() error {
	if strings.TrimSpace(o.BindAddress) == "" {
		return xfmt.Errorf("run server options: bindAddress is required")
	}
	if o.Port <= 0 {
		return xfmt.Errorf("run server options: port must be positive")
	}
	return nil
}

type RunDBOptions struct {
	Dialect     string
	DSN         string
	AllowCreate bool
}

func (o RunDBOptions) Validate() error {
	if strings.TrimSpace(o.Dialect) == "" {
		return xfmt.Errorf("run db options: dialect is required")
	}
	return nil
}

type RunScopeInput struct {
	options       *ScopeInputConfigOptions
	cliOptions    Options
	serverOptions RunServerOptions
	dbOptions     RunDBOptions
}

func NewRunScopeInput(options *ScopeInputConfigOptions, cliOptions Options, serverOptions RunServerOptions, dbOptions RunDBOptions) RunScopeInput {
	return RunScopeInput{
		options:       options,
		cliOptions:    cliOptions,
		serverOptions: serverOptions,
		dbOptions:     dbOptions,
	}
}

func (i RunScopeInput) ConfigOptions() *ScopeInputConfigOptions {
	return i.options
}

func (i RunScopeInput) CLIOptions() Options {
	return i.cliOptions
}

func (i RunScopeInput) ServerOptions() RunServerOptions {
	return i.serverOptions
}

func (i RunScopeInput) DBOptions() RunDBOptions {
	return i.dbOptions
}

func (i RunScopeInput) Environment() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.Environment
}

func (i RunScopeInput) ModulesPath() string {
	if strings.TrimSpace(i.cliOptions.ModulesPath) != "" {
		return i.cliOptions.ModulesPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModulesPath
}

func (i RunScopeInput) DistPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.DistPath
}

func (i RunScopeInput) TmpPath() string {
	if strings.TrimSpace(i.cliOptions.TmpPath) != "" {
		return i.cliOptions.TmpPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.TmpPath
}

func (i RunScopeInput) DefaultChoysumPath() string {
	if strings.TrimSpace(i.cliOptions.DefaultChoysumPath) != "" {
		return i.cliOptions.DefaultChoysumPath
	}
	if i.options == nil {
		return ""
	}
	return i.options.DefaultChoysumPath
}

func (i RunScopeInput) ConfigPath() string {
	if i.options == nil {
		return ""
	}
	return i.options.ConfigPath
}

func (i RunScopeInput) ESMUpstreamURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.ESMUpstreamURL
}

func (i RunScopeInput) NpmRegistryURL() string {
	if i.options == nil {
		return ""
	}
	return i.options.NPMRegistryURL
}

func (i RunScopeInput) ModuleCatalogIndexURL() string {
	if strings.TrimSpace(i.cliOptions.ModuleCatalogIndexURL) != "" {
		return i.cliOptions.ModuleCatalogIndexURL
	}
	if i.options == nil {
		return ""
	}
	return i.options.ModuleCatalogIndexURL
}

func (i RunScopeInput) BootstrapModuleInstallTimeoutSeconds() int {
	if i.options == nil {
		return 0
	}
	return i.options.BootstrapModuleInstallTimeoutSeconds
}

func (i RunScopeInput) CompileBundleMode() string {
	if i.options == nil || i.options.Compile == nil {
		return ""
	}
	return i.options.Compile.BundleMode
}

func (i RunScopeInput) CompileConfig() *config.CompileConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyCompileConfig()
}

func (i RunScopeInput) AuthEnabled() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.Enabled
}

func (i RunScopeInput) AuthConfig() *config.AuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthConfig()
}

func (i RunScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthHttpAuth()
}

func (i RunScopeInput) AuthGrpcAuthentication() bool {
	if i.options == nil || i.options.Auth == nil {
		return false
	}
	return i.options.Auth.GrpcAuthentication
}

func (i RunScopeInput) AuthInternalKey() string {
	if i.options == nil || i.options.Auth == nil {
		return ""
	}
	return i.options.Auth.InternalKey
}

func (i RunScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthJobTokenAllowedSANs()
}

func (i RunScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyAuthGrpcEntryPolicy()
}

func (i RunScopeInput) ServerEnabledTLS() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnabledTLS
}

func (i RunScopeInput) ServerConfig() *config.ServerConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerConfig()
}

func (i RunScopeInput) ServerBindAddress() string {
	if strings.TrimSpace(i.serverOptions.BindAddress) != "" {
		return i.serverOptions.BindAddress
	}
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.BindAddress
}

func (i RunScopeInput) ServerPort() int {
	if i.serverOptions.Port > 0 {
		return i.serverOptions.Port
	}
	if i.options == nil || i.options.Server == nil {
		return 0
	}
	return i.options.Server.Port
}

func (i RunScopeInput) ServerEnableGzip() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGzip
}

func (i RunScopeInput) ServerTLSCaFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCaFile
}

func (i RunScopeInput) ServerTLSServerName() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSServerName
}

func (i RunScopeInput) ServerTLSCertFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSCertFile
}

func (i RunScopeInput) ServerTLSKeyFile() string {
	if i.options == nil || i.options.Server == nil {
		return ""
	}
	return i.options.Server.TLSKeyFile
}

func (i RunScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.EnableGrpcWebProxy
}

func (i RunScopeInput) ServerHotReload() bool {
	if i.options == nil || i.options.Server == nil {
		return false
	}
	return i.options.Server.HotReload
}

func (i RunScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.options == nil || i.options.Server == nil || i.options.Server.GrpcClient == nil {
		return 0
	}
	return i.options.Server.GrpcClient.MaxCachedConns
}

func (i RunScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSPConfig()
}

func (i RunScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerHSTSConfig()
}

func (i RunScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyServerCSRFConfig()
}

func (i RunScopeInput) ServerSecurityMissing() bool {
	return i.options == nil || i.options.Server == nil || i.options.Server.Security == nil
}

func (i RunScopeInput) TaskConfig() *config.TaskConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyTaskConfig()
}

func (i RunScopeInput) LogConfig() *config.LogConfig {
	if i.options == nil {
		return nil
	}
	return i.options.CopyLogConfig()
}

func (i RunScopeInput) FrontendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyFrontendEnv()
}

func (i RunScopeInput) BackendEnv() map[string]any {
	if i.options == nil {
		return nil
	}
	return i.options.CopyBackendEnv()
}

func (i RunScopeInput) DatabaseDialect() string {
	if strings.TrimSpace(i.dbOptions.Dialect) != "" {
		return i.dbOptions.Dialect
	}
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.Dialect
}

func (i RunScopeInput) DatabaseDSN() string {
	if strings.TrimSpace(i.dbOptions.DSN) != "" {
		return i.dbOptions.DSN
	}
	if i.options == nil || i.options.Db == nil {
		return ""
	}
	return i.options.Db.DSN
}

func (i RunScopeInput) DatabaseMaxOpenConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxOpenConns
}

func (i RunScopeInput) DatabaseMaxIdleConns() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.MaxIdleConns
}

func (i RunScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.options == nil || i.options.Db == nil {
		return 0
	}
	return i.options.Db.ConnMaxLifetime
}
