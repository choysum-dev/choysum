// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type testScopeConfigSnapshot = snapshot.ConfigSnapshot

func newTestScopeConfigSnapshot(cfg *testScopeConfigSnapshot) *testScopeConfigSnapshot {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

// testRuntimeScopeInput keeps backend runtime options attached to scope construction.
type testRuntimeScopeInput struct {
	cfg            *testScopeConfigSnapshot
	runtimeOptions runtimeOptions
}

func newTestRuntimeScopeInput(cfg *testScopeConfigSnapshot) testRuntimeScopeInput {
	pathOpts := scope.PathsRuntimeOptions{}
	compileOpts := scope.CompileRuntimeOptions{}
	serverOpts := scope.ServerRuntimeOptions{}
	authOpts := scope.AuthRuntimeOptions{}
	hasPathOpts := false
	hasCompileOpts := false
	hasServerOpts := false
	hasAuthOpts := false

	if cfg != nil {
		pathOpts = scope.PathsRuntimeOptions{AddonsPath: cfg.AddonsPath, DistPath: cfg.DistPath, TmpPath: cfg.TmpPath, NpmRegistryURL: cfg.NPMRegistryURL}
		hasPathOpts = true
		if cfg.Compile != nil {
			compileOpts = scope.CompileRuntimeOptions{BundleMode: cfg.Compile.BundleMode}
			hasCompileOpts = true
		}
		if cfg.Server != nil {
			serverOpts = scope.ServerRuntimeOptions{
				JsEngineFactory:   cfg.Server.JsEngineFactory,
				JsExecutorFactory: cfg.Server.JsExecutorFactory,
			}
			hasServerOpts = true
		}
		if cfg.Auth != nil {
			authOpts = scope.AuthRuntimeOptions{Enabled: cfg.Auth.Enabled}
			hasAuthOpts = true
		}
	}

	return testRuntimeScopeInput{
		cfg:            newTestScopeConfigSnapshot(cfg),
		runtimeOptions: newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts),
	}
}

func newTestRuntimeScopeInputFromScope(runtimeScope scope.Scope, dbOpts scope.DatabaseRuntimeOptions) testRuntimeScopeInput {
	if runtimeScope == nil {
		return testRuntimeScopeInput{}
	}

	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)

	cfg := &config.Config{}
	if hasPathOpts {
		cfg.AddonsPath = pathOpts.AddonsPath
		cfg.DistPath = pathOpts.DistPath
		cfg.TmpPath = pathOpts.TmpPath
		cfg.DefaultChoysumPath = pathOpts.DefaultChoysumPath
		cfg.ConfigPath = pathOpts.ConfigPath
		cfg.NpmPath = pathOpts.NpmPath
		cfg.NPMRegistryURL = pathOpts.NpmRegistryURL
	}
	cfg.Log = scope.LogConfigFromScope(runtimeScope)

	if hasCompileOpts {
		cfg.Compile = &config.CompileConfig{
			BundleMode:  compileOpts.BundleMode,
			SourceMap:   compileOpts.SourceMap,
			Minify:      compileOpts.Minify,
			TreeShaking: compileOpts.TreeShaking,
		}
	}
	if cfg.Compile == nil {
		cfg.Compile = config.NewDefaultCompileConfig()
	}
	// Unit backend tests rely on sourcemap remapping for coverage artifacts and on
	// stable model/constructor labels in diagnostics, so keep sourcemaps on and
	// minification off in the derived test scope.
	cfg.Compile.SourceMap = true
	cfg.Compile.Minify = false

	if hasAuthOpts {
		cfg.Auth = &config.AuthConfig{
			Enabled:             authOpts.Enabled,
			Type:                authOpts.Type,
			JWT:                 authOpts.JWT,
			HttpAuth:            authOpts.HttpAuth,
			GrpcAuthentication:  authOpts.GrpcAuthentication,
			GrpcMethodAccess:    authOpts.GrpcMethodAccess,
			GrpcRecordRule:      authOpts.GrpcRecordRule,
			GrpcCompanyFilter:   authOpts.GrpcCompanyFilter,
			GrpcFieldRule:       authOpts.GrpcFieldRule,
			InternalKey:         authOpts.InternalKey,
			JobTokenAllowedSANs: authOpts.JobTokenAllowedSANs,
			GrpcEntryPolicy:     authOpts.GrpcEntryPolicy,
			AuthzDecisionLog:    authOpts.AuthzDecisionLog,
			AuthzDecisionAudit:  authOpts.AuthzDecisionAudit,
		}
	}

	if hasServerOpts {
		serverCfg := &config.ServerConfig{
			BindAddress:        serverOpts.BindAddress,
			Port:               serverOpts.Port,
			EnableGzip:         serverOpts.EnableGzip,
			Register:           serverOpts.Register,
			Environment:        serverOpts.Environment,
			EnabledTLS:         serverOpts.EnabledTLS,
			TLSCaFile:          serverOpts.TLSCaFile,
			TLSServerName:      serverOpts.TLSServerName,
			TLSCertFile:        serverOpts.TLSCertFile,
			TLSKeyFile:         serverOpts.TLSKeyFile,
			EnableGrpcWebProxy: serverOpts.EnableGrpcWebProxy,
			HotReload:          serverOpts.HotReload,
			WebBaseURL:         serverOpts.WebBaseURL,
			RootRedirectURL:    serverOpts.RootRedirectURL,
			JsEngineFactory:    serverOpts.JsEngineFactory,
			JsExecutorFactory:  serverOpts.JsExecutorFactory,
		}
		if serverOpts.GrpcClientMaxCachedConns > 0 {
			serverCfg.GrpcClient = &config.GrpcClientConfig{MaxCachedConns: serverOpts.GrpcClientMaxCachedConns}
		}
		if !serverOpts.SecurityMissing || serverOpts.CSP != nil || serverOpts.HSTS != nil || serverOpts.CSRF != nil {
			serverCfg.Security = &config.SecurityConfig{
				CSP:  serverOpts.CSP,
				HSTS: serverOpts.HSTS,
				CSRF: serverOpts.CSRF,
			}
		}
		cfg.Server = serverCfg
	}

	if taskOpts, ok := scope.TaskRuntimeOptionsFromScope(runtimeScope); ok {
		cfg.Task = taskOpts.Task
	}

	if envOpts, ok := scope.RuntimeEnvironmentOptionsFromScope(runtimeScope); ok {
		cfg.FrontendEnv = envOpts.FrontendEnv
		cfg.BackendEnv = envOpts.BackendEnv
	}

	if strings.TrimSpace(dbOpts.Dialect) != "" || strings.TrimSpace(dbOpts.DSN) != "" || dbOpts.MaxOpenConns > 0 || dbOpts.MaxIdleConns > 0 || dbOpts.ConnMaxLifetimeSeconds > 0 {
		cfg.Db = &config.DbConfig{
			Dialect:         dbOpts.Dialect,
			DSN:             dbOpts.DSN,
			MaxOpenConns:    dbOpts.MaxOpenConns,
			MaxIdleConns:    dbOpts.MaxIdleConns,
			ConnMaxLifetime: dbOpts.ConnMaxLifetimeSeconds,
		}
	}

	return testRuntimeScopeInput{
		cfg:            newTestScopeConfigSnapshot(snapshot.New(cfg)),
		runtimeOptions: newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts),
	}
}

func (i testRuntimeScopeInput) Environment() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.Environment
}

func (i testRuntimeScopeInput) AddonsPath() string {
	if i.runtimeOptions.addonsPath != "" {
		return i.runtimeOptions.addonsPath
	}
	if i.cfg == nil {
		return ""
	}
	return i.cfg.AddonsPath
}

func (i testRuntimeScopeInput) DistPath() string {
	if i.runtimeOptions.distPath != "" {
		return i.runtimeOptions.distPath
	}
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DistPath
}

func (i testRuntimeScopeInput) TmpPath() string {
	if i.runtimeOptions.tmpPath != "" {
		return i.runtimeOptions.tmpPath
	}
	if i.cfg == nil {
		return ""
	}
	return i.cfg.TmpPath
}

func (i testRuntimeScopeInput) DefaultChoysumPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DefaultChoysumPath
}

func (i testRuntimeScopeInput) ConfigPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ConfigPath
}

func (i testRuntimeScopeInput) NpmPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.NpmPath
}

func (i testRuntimeScopeInput) NpmRegistryURL() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.NPMRegistryURL
}

func (i testRuntimeScopeInput) CompileBundleMode() string {
	if i.runtimeOptions.compileBundleMode != "" {
		return i.runtimeOptions.compileBundleMode
	}
	if i.cfg == nil || i.cfg.Compile == nil {
		return ""
	}
	return i.cfg.Compile.BundleMode
}

func (i testRuntimeScopeInput) CompileConfig() *config.CompileConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyCompileConfig()
}

func (i testRuntimeScopeInput) AuthEnabled() bool {
	return i.runtimeOptions.authEnabled
}

func (i testRuntimeScopeInput) AuthConfig() *config.AuthConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyAuthConfig()
}

func (i testRuntimeScopeInput) AuthHttpAuth() *config.HttpAuthConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyAuthHttpAuth()
}

func (i testRuntimeScopeInput) AuthGrpcAuthentication() bool {
	if i.cfg == nil || i.cfg.Auth == nil {
		return false
	}
	return i.cfg.Auth.GrpcAuthentication
}

func (i testRuntimeScopeInput) AuthInternalKey() string {
	if i.cfg == nil || i.cfg.Auth == nil {
		return ""
	}
	return i.cfg.Auth.InternalKey
}

func (i testRuntimeScopeInput) AuthJobTokenAllowedSANs() []string {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyAuthJobTokenAllowedSANs()
}

func (i testRuntimeScopeInput) AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyAuthGrpcEntryPolicy()
}

func (i testRuntimeScopeInput) ServerEnabledTLS() bool {
	if i.cfg == nil || i.cfg.Server == nil {
		return false
	}
	return i.cfg.Server.EnabledTLS
}

func (i testRuntimeScopeInput) ServerConfig() *config.ServerConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyServerConfig()
}

func (i testRuntimeScopeInput) ServerBindAddress() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.BindAddress
}

func (i testRuntimeScopeInput) ServerPort() int {
	if i.cfg == nil || i.cfg.Server == nil {
		return 0
	}
	return i.cfg.Server.Port
}

func (i testRuntimeScopeInput) ServerEnableGzip() bool {
	if i.cfg == nil || i.cfg.Server == nil {
		return false
	}
	return i.cfg.Server.EnableGzip
}

func (i testRuntimeScopeInput) ServerTLSCaFile() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.TLSCaFile
}

func (i testRuntimeScopeInput) ServerTLSServerName() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.TLSServerName
}

func (i testRuntimeScopeInput) ServerTLSCertFile() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.TLSCertFile
}

func (i testRuntimeScopeInput) ServerTLSKeyFile() string {
	if i.cfg == nil || i.cfg.Server == nil {
		return ""
	}
	return i.cfg.Server.TLSKeyFile
}

func (i testRuntimeScopeInput) ServerEnableGrpcWebProxy() bool {
	if i.cfg == nil || i.cfg.Server == nil {
		return false
	}
	return i.cfg.Server.EnableGrpcWebProxy
}

func (i testRuntimeScopeInput) ServerHotReload() bool {
	if i.cfg == nil || i.cfg.Server == nil {
		return false
	}
	return i.cfg.Server.HotReload
}

func (i testRuntimeScopeInput) ServerGrpcClientMaxCachedConns() int {
	if i.cfg == nil || i.cfg.Server == nil || i.cfg.Server.GrpcClient == nil {
		return 0
	}
	return i.cfg.Server.GrpcClient.MaxCachedConns
}

func (i testRuntimeScopeInput) ServerCSPConfig() *config.CSPConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyServerCSPConfig()
}

func (i testRuntimeScopeInput) ServerHSTSConfig() *config.HSTSConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyServerHSTSConfig()
}

func (i testRuntimeScopeInput) ServerCSRFConfig() *config.CSRFConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyServerCSRFConfig()
}

func (i testRuntimeScopeInput) ServerSecurityMissing() bool {
	return i.cfg == nil || i.cfg.Server == nil || i.cfg.Server.Security == nil
}

func (i testRuntimeScopeInput) TaskConfig() *config.TaskConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyTaskConfig()
}

func (i testRuntimeScopeInput) LogConfig() *config.LogConfig {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyLogConfig()
}

func (i testRuntimeScopeInput) FrontendEnv() map[string]any {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyFrontendEnv()
}

func (i testRuntimeScopeInput) BackendEnv() map[string]any {
	if i.cfg == nil {
		return nil
	}
	return i.cfg.CopyBackendEnv()
}

func (i testRuntimeScopeInput) DatabaseDialect() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.Dialect
}

func (i testRuntimeScopeInput) DatabaseDSN() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.DSN
}

func (i testRuntimeScopeInput) DatabaseMaxOpenConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxOpenConns
}

func (i testRuntimeScopeInput) DatabaseMaxIdleConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxIdleConns
}

func (i testRuntimeScopeInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.ConnMaxLifetime
}
