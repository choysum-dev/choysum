// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/config"
)

type FactoryInput interface {
	Environment() string
}

type PathsInput interface {
	ModulesPath() string
	DistPath() string
	TmpPath() string
}

type PathsDefaultInput interface {
	DefaultChoysumPath() string
}

type PathsConfigInput interface {
	ConfigPath() string
}

type NpmPathInput interface {
	NpmPath() string
}

type NpmRegistryURLInput interface {
	NpmRegistryURL() string
}

type ModuleCatalogIndexURLInput interface {
	ModuleCatalogIndexURL() string
}

type ESMUpstreamURLInput interface {
	ESMUpstreamURL() string
}

type CompileInput interface {
	CompileBundleMode() string
}

type CompileConfigInput interface {
	CompileConfig() *config.CompileConfig
}

type AuthInput interface {
	AuthEnabled() bool
}

type AuthDetailsInput interface {
	AuthHttpAuth() *config.HttpAuthConfig
	AuthGrpcAuthentication() bool
	AuthInternalKey() string
	AuthJobTokenAllowedSANs() []string
	AuthGrpcEntryPolicy() map[string]*config.EntryMethodConfig
}

type AuthConfigInput interface {
	AuthConfig() *config.AuthConfig
}

type ServerInput interface {
	ServerBindAddress() string
	ServerPort() int
	ServerEnableGzip() bool
	ServerEnabledTLS() bool
	ServerTLSCaFile() string
	ServerTLSServerName() string
	ServerTLSCertFile() string
	ServerTLSKeyFile() string
	ServerEnableGrpcWebProxy() bool
	ServerHotReload() bool
	ServerGrpcClientMaxCachedConns() int
}

type ServerConfigInput interface {
	ServerConfig() *config.ServerConfig
}

type ServerSecurityInput interface {
	ServerCSPConfig() *config.CSPConfig
	ServerHSTSConfig() *config.HSTSConfig
	ServerCSRFConfig() *config.CSRFConfig
	ServerSecurityMissing() bool
}

type TaskConfigInput interface {
	TaskConfig() *config.TaskConfig
}

type LogConfigInput interface {
	LogConfig() *config.LogConfig
}

type RuntimeEnvironmentInput interface {
	FrontendEnv() map[string]any
	BackendEnv() map[string]any
}

type DocumentConfigInput interface {
	DocumentConfig() *config.DocumentConfig
}

type DatabaseInput interface {
	DatabaseDialect() string
	DatabaseDSN() string
	DatabaseMaxOpenConns() int
	DatabaseMaxIdleConns() int
	DatabaseConnMaxLifetimeSeconds() int
}

type PathsRuntimeOptions struct {
	ModulesPath           string
	DistPath              string
	TmpPath               string
	DefaultChoysumPath    string
	ConfigPath            string
	NpmPath               string
	NpmRegistryURL        string
	ModuleCatalogIndexURL string
	ESMUpstreamURL        string
}

type CompileRuntimeOptions struct {
	BundleMode  string
	SourceMap   bool
	Minify      bool
	TreeShaking bool
}

type AuthRuntimeOptions struct {
	Enabled             bool
	Type                string
	JWT                 *config.JWTConfig
	HttpAuth            *config.HttpAuthConfig
	GrpcAuthentication  bool
	GrpcMethodAccess    bool
	GrpcRecordRule      bool
	GrpcCompanyFilter   bool
	GrpcFieldRule       bool
	InternalKey         string
	JobTokenAllowedSANs []string
	GrpcEntryPolicy     map[string]*config.EntryMethodConfig
	AuthzDecisionLog    string
	AuthzDecisionAudit  bool
}

type ServerRuntimeOptions struct {
	BindAddress              string
	Port                     int
	EnableGzip               bool
	Register                 string
	Environment              string
	EnabledTLS               bool
	TLSCaFile                string
	TLSServerName            string
	TLSCertFile              string
	TLSKeyFile               string
	EnableGrpcWebProxy       bool
	HotReload                bool
	HotReloadQueueSize       int
	GrpcClientMaxCachedConns int
	WebBaseURL               string
	RootRedirectURL          string
	JsEngineFactory          string
	JsExecutorFactory        string
	CSP                      *config.CSPConfig
	HSTS                     *config.HSTSConfig
	CSRF                     *config.CSRFConfig
	SecurityMissing          bool
}

type TaskRuntimeOptions struct {
	Task *config.TaskConfig
}

type RuntimeEnvironmentOptions struct {
	FrontendEnv map[string]any
	BackendEnv  map[string]any
}

type DocumentRuntimeOptions struct {
	Attachment *config.AttachmentConfig
}

type DatabaseRuntimeOptions struct {
	Dialect                string
	DSN                    string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int
}

func PathsRuntimeOptionsFromInput(input FactoryInput) (PathsRuntimeOptions, bool) {
	if input == nil {
		return PathsRuntimeOptions{}, false
	}
	pathsInput, ok := input.(PathsInput)
	if !ok {
		return PathsRuntimeOptions{}, false
	}
	options := PathsRuntimeOptions{
		ModulesPath: pathsInput.ModulesPath(),
		DistPath:    pathsInput.DistPath(),
		TmpPath:     pathsInput.TmpPath(),
	}
	if defaultsInput, ok := input.(PathsDefaultInput); ok {
		options.DefaultChoysumPath = defaultsInput.DefaultChoysumPath()
	}
	if configInput, ok := input.(PathsConfigInput); ok {
		options.ConfigPath = configInput.ConfigPath()
	}
	if npmPathInput, ok := input.(NpmPathInput); ok {
		options.NpmPath = npmPathInput.NpmPath()
	}
	if npmRegistryURLInput, ok := input.(NpmRegistryURLInput); ok {
		options.NpmRegistryURL = npmRegistryURLInput.NpmRegistryURL()
	}
	if moduleCatalogIndexURLInput, ok := input.(ModuleCatalogIndexURLInput); ok {
		options.ModuleCatalogIndexURL = moduleCatalogIndexURLInput.ModuleCatalogIndexURL()
	}
	if esmUpstreamInput, ok := input.(ESMUpstreamURLInput); ok {
		options.ESMUpstreamURL = esmUpstreamInput.ESMUpstreamURL()
	}
	return options, true
}

func CompileRuntimeOptionsFromInput(input FactoryInput) (CompileRuntimeOptions, bool) {
	if input == nil {
		return CompileRuntimeOptions{}, false
	}

	options := CompileRuntimeOptions{}
	hasOptions := false

	if compileConfigInput, ok := input.(CompileConfigInput); ok {
		if compileCfg := compileConfigInput.CompileConfig(); compileCfg != nil {
			options.BundleMode = compileCfg.BundleMode
			options.SourceMap = compileCfg.SourceMap
			options.Minify = compileCfg.Minify
			options.TreeShaking = compileCfg.TreeShaking
			hasOptions = true
		}
	}
	if compileInput, ok := input.(CompileInput); ok {
		options.BundleMode = compileInput.CompileBundleMode()
		hasOptions = true
	}

	if !hasOptions {
		return CompileRuntimeOptions{}, false
	}

	return options, true
}

func AuthRuntimeOptionsFromInput(input FactoryInput) (AuthRuntimeOptions, bool) {
	if input == nil {
		return AuthRuntimeOptions{}, false
	}

	options := AuthRuntimeOptions{}
	hasOptions := false

	if authInput, ok := input.(AuthInput); ok {
		options.Enabled = authInput.AuthEnabled()
		hasOptions = true
	}
	if authConfigInput, ok := input.(AuthConfigInput); ok {
		if authCfg := cloneAuthConfig(authConfigInput.AuthConfig()); authCfg != nil {
			options.Enabled = authCfg.Enabled
			options.Type = authCfg.Type
			options.JWT = cloneJWTConfig(authCfg.JWT)
			options.HttpAuth = cloneHttpAuthConfig(authCfg.HttpAuth)
			options.GrpcAuthentication = authCfg.GrpcAuthentication
			options.GrpcMethodAccess = authCfg.GrpcMethodAccess
			options.GrpcRecordRule = authCfg.GrpcRecordRule
			options.GrpcCompanyFilter = authCfg.GrpcCompanyFilter
			options.GrpcFieldRule = authCfg.GrpcFieldRule
			options.InternalKey = authCfg.InternalKey
			options.JobTokenAllowedSANs = cloneStringSlice(authCfg.JobTokenAllowedSANs)
			options.GrpcEntryPolicy = cloneEntryMethodConfigMap(authCfg.GrpcEntryPolicy)
			options.AuthzDecisionLog = authCfg.AuthzDecisionLog
			options.AuthzDecisionAudit = authCfg.AuthzDecisionAudit
			hasOptions = true
		}
	}
	if authDetailsInput, ok := input.(AuthDetailsInput); ok {
		options.HttpAuth = cloneHttpAuthConfig(authDetailsInput.AuthHttpAuth())
		options.GrpcAuthentication = authDetailsInput.AuthGrpcAuthentication()
		options.InternalKey = authDetailsInput.AuthInternalKey()
		options.JobTokenAllowedSANs = cloneStringSlice(authDetailsInput.AuthJobTokenAllowedSANs())
		options.GrpcEntryPolicy = cloneEntryMethodConfigMap(authDetailsInput.AuthGrpcEntryPolicy())
		hasOptions = true
	}

	if !hasOptions {
		return AuthRuntimeOptions{}, false
	}

	return options, true
}

func ServerRuntimeOptionsFromInput(input FactoryInput) (ServerRuntimeOptions, bool) {
	if input == nil {
		return ServerRuntimeOptions{}, false
	}
	options := ServerRuntimeOptions{}
	hasOptions := false

	if serverConfigInput, ok := input.(ServerConfigInput); ok {
		if serverCfg := cloneServerConfig(serverConfigInput.ServerConfig()); serverCfg != nil {
			options.BindAddress = serverCfg.BindAddress
			options.Port = serverCfg.Port
			options.EnableGzip = serverCfg.EnableGzip
			options.Register = serverCfg.Register
			options.Environment = serverCfg.Environment
			options.EnabledTLS = serverCfg.EnabledTLS
			options.TLSCaFile = serverCfg.TLSCaFile
			options.TLSServerName = serverCfg.TLSServerName
			options.TLSCertFile = serverCfg.TLSCertFile
			options.TLSKeyFile = serverCfg.TLSKeyFile
			options.EnableGrpcWebProxy = serverCfg.EnableGrpcWebProxy
			options.HotReload = serverCfg.HotReload
			options.HotReloadQueueSize = serverCfg.HotReloadQueueSize
			options.WebBaseURL = serverCfg.WebBaseURL
			options.RootRedirectURL = serverCfg.RootRedirectURL
			options.JsEngineFactory = serverCfg.JsEngineFactory
			options.JsExecutorFactory = serverCfg.JsExecutorFactory
			if serverCfg.GrpcClient != nil {
				options.GrpcClientMaxCachedConns = serverCfg.GrpcClient.MaxCachedConns
			}
			if serverCfg.Security != nil {
				options.CSP = cloneCSPConfig(serverCfg.Security.CSP)
				options.HSTS = cloneHSTSConfig(serverCfg.Security.HSTS)
				options.CSRF = cloneCSRFConfig(serverCfg.Security.CSRF)
			} else {
				options.SecurityMissing = true
			}
			hasOptions = true
		}
	}

	if serverInput, ok := input.(ServerInput); ok {
		options.BindAddress = serverInput.ServerBindAddress()
		options.Port = serverInput.ServerPort()
		options.EnableGzip = serverInput.ServerEnableGzip()
		options.EnabledTLS = serverInput.ServerEnabledTLS()
		options.TLSCaFile = serverInput.ServerTLSCaFile()
		options.TLSServerName = serverInput.ServerTLSServerName()
		options.TLSCertFile = serverInput.ServerTLSCertFile()
		options.TLSKeyFile = serverInput.ServerTLSKeyFile()
		options.EnableGrpcWebProxy = serverInput.ServerEnableGrpcWebProxy()
		options.HotReload = serverInput.ServerHotReload()
		options.GrpcClientMaxCachedConns = serverInput.ServerGrpcClientMaxCachedConns()
		hasOptions = true
	}
	if strings.TrimSpace(options.Environment) == "" {
		options.Environment = input.Environment()
	}
	if securityInput, ok := input.(ServerSecurityInput); ok {
		options.CSP = cloneCSPConfig(securityInput.ServerCSPConfig())
		options.HSTS = cloneHSTSConfig(securityInput.ServerHSTSConfig())
		options.CSRF = cloneCSRFConfig(securityInput.ServerCSRFConfig())
		options.SecurityMissing = securityInput.ServerSecurityMissing()
		hasOptions = true
	}

	if !hasOptions {
		return ServerRuntimeOptions{}, false
	}
	return options, true
}

func TaskRuntimeOptionsFromInput(input FactoryInput) (TaskRuntimeOptions, bool) {
	if input == nil {
		return TaskRuntimeOptions{}, false
	}
	taskInput, ok := input.(TaskConfigInput)
	if !ok {
		return TaskRuntimeOptions{}, false
	}
	taskCfg := cloneTaskConfig(taskInput.TaskConfig())
	if taskCfg == nil {
		return TaskRuntimeOptions{}, false
	}
	return TaskRuntimeOptions{Task: taskCfg}, true
}

func RuntimeEnvironmentOptionsFromInput(input FactoryInput) (RuntimeEnvironmentOptions, bool) {
	if input == nil {
		return RuntimeEnvironmentOptions{}, false
	}
	environmentInput, ok := input.(RuntimeEnvironmentInput)
	if !ok {
		return RuntimeEnvironmentOptions{}, false
	}
	return RuntimeEnvironmentOptions{
		FrontendEnv: cloneAnyMap(environmentInput.FrontendEnv()),
		BackendEnv:  cloneAnyMap(environmentInput.BackendEnv()),
	}, true
}

func DocumentRuntimeOptionsFromInput(input FactoryInput) (DocumentRuntimeOptions, bool) {
	if input == nil {
		return DocumentRuntimeOptions{}, false
	}
	documentInput, ok := input.(DocumentConfigInput)
	if !ok {
		return DocumentRuntimeOptions{}, false
	}
	documentCfg := cloneDocumentConfig(documentInput.DocumentConfig())
	if documentCfg == nil {
		return DocumentRuntimeOptions{}, false
	}
	return DocumentRuntimeOptions{Attachment: cloneAttachmentConfig(documentCfg.Attachment)}, true
}

func DatabaseRuntimeOptionsFromInput(input FactoryInput) (DatabaseRuntimeOptions, bool) {
	if input == nil {
		return DatabaseRuntimeOptions{}, false
	}
	dbInput, ok := input.(DatabaseInput)
	if !ok {
		return DatabaseRuntimeOptions{}, false
	}
	return DatabaseRuntimeOptions{
		Dialect:                dbInput.DatabaseDialect(),
		DSN:                    dbInput.DatabaseDSN(),
		MaxOpenConns:           dbInput.DatabaseMaxOpenConns(),
		MaxIdleConns:           dbInput.DatabaseMaxIdleConns(),
		ConnMaxLifetimeSeconds: dbInput.DatabaseConnMaxLifetimeSeconds(),
	}, true
}
func LogConfigFromInput(input FactoryInput) *config.LogConfig {
	if input == nil {
		return nil
	}
	logInput, ok := input.(LogConfigInput)
	if !ok {
		return nil
	}
	logCfg := logInput.LogConfig()
	if logCfg == nil {
		return nil
	}
	cloned := *logCfg
	return &cloned
}

func PathsRuntimeOptionsFromScope(runtimeScope Scope) (PathsRuntimeOptions, bool) {
	if runtimeScope == nil {
		return PathsRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := PathsRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return PathsRuntimeOptions{}, false
}

func CompileRuntimeOptionsFromScope(runtimeScope Scope) (CompileRuntimeOptions, bool) {
	if runtimeScope == nil {
		return CompileRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := CompileRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return CompileRuntimeOptions{}, false
}

func AuthRuntimeOptionsFromScope(runtimeScope Scope) (AuthRuntimeOptions, bool) {
	if runtimeScope == nil {
		return AuthRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := AuthRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return AuthRuntimeOptions{}, false
}

func ServerRuntimeOptionsFromScope(runtimeScope Scope) (ServerRuntimeOptions, bool) {
	if runtimeScope == nil {
		return ServerRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := ServerRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return ServerRuntimeOptions{}, false
}

func TaskRuntimeOptionsFromScope(runtimeScope Scope) (TaskRuntimeOptions, bool) {
	if runtimeScope == nil {
		return TaskRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := TaskRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return TaskRuntimeOptions{}, false
}

func RuntimeEnvironmentOptionsFromScope(runtimeScope Scope) (RuntimeEnvironmentOptions, bool) {
	if runtimeScope == nil {
		return RuntimeEnvironmentOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := RuntimeEnvironmentOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return RuntimeEnvironmentOptions{}, false
}

func DocumentRuntimeOptionsFromScope(runtimeScope Scope) (DocumentRuntimeOptions, bool) {
	if runtimeScope == nil {
		return DocumentRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := DocumentRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return DocumentRuntimeOptions{}, false
}

func DatabaseRuntimeOptionsFromScope(runtimeScope Scope) (DatabaseRuntimeOptions, bool) {
	if runtimeScope == nil {
		return DatabaseRuntimeOptions{}, false
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		if opts, ok := DatabaseRuntimeOptionsFromInput(input); ok {
			return opts, true
		}
	}
	return DatabaseRuntimeOptions{}, false
}
func LogConfigFromScope(runtimeScope Scope) *config.LogConfig {
	if runtimeScope == nil {
		return nil
	}
	if input := FactoryInputFromScope(runtimeScope); input != nil {
		return LogConfigFromInput(input)
	}
	return nil
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneEntryMethodConfigMap(values map[string]*config.EntryMethodConfig) map[string]*config.EntryMethodConfig {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]*config.EntryMethodConfig, len(values))
	for key, value := range values {
		if value == nil {
			cloned[key] = nil
			continue
		}
		valueCopy := *value
		if len(value.RecordRuleAllow) > 0 {
			recordRules := make([]config.EntryRecordRuleAllow, len(value.RecordRuleAllow))
			for idx, rule := range value.RecordRuleAllow {
				ruleCopy := rule
				ruleCopy.Ops = cloneStringSlice(rule.Ops)
				recordRules[idx] = ruleCopy
			}
			valueCopy.RecordRuleAllow = recordRules
		}
		cloned[key] = &valueCopy
	}
	return cloned
}

func cloneHttpAuthConfig(cfg *config.HttpAuthConfig) *config.HttpAuthConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	cloned.ExcludedRegex = cloneStringSlice(cfg.ExcludedRegex)
	cloned.TokenExtractors = cloneStringSlice(cfg.TokenExtractors)
	return &cloned
}

func cloneJWTConfig(cfg *config.JWTConfig) *config.JWTConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.IdentityCache != nil {
		identityCache := *cfg.IdentityCache
		cloned.IdentityCache = &identityCache
	}
	return &cloned
}

func cloneAuthConfig(cfg *config.AuthConfig) *config.AuthConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.JWT = cloneJWTConfig(cfg.JWT)
	cloned.HttpAuth = cloneHttpAuthConfig(cfg.HttpAuth)
	cloned.JobTokenAllowedSANs = cloneStringSlice(cfg.JobTokenAllowedSANs)
	cloned.GrpcEntryPolicy = cloneEntryMethodConfigMap(cfg.GrpcEntryPolicy)
	return &cloned
}

func cloneServerConfig(cfg *config.ServerConfig) *config.ServerConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.GrpcClient != nil {
		grpcClient := *cfg.GrpcClient
		cloned.GrpcClient = &grpcClient
	}
	if cfg.Security != nil {
		security := *cfg.Security
		security.CSP = cloneCSPConfig(cfg.Security.CSP)
		security.HSTS = cloneHSTSConfig(cfg.Security.HSTS)
		security.CSRF = cloneCSRFConfig(cfg.Security.CSRF)
		cloned.Security = &security
	}
	return &cloned
}

func cloneTaskConfig(cfg *config.TaskConfig) *config.TaskConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.Dispatch != nil {
		dispatch := *cfg.Dispatch
		cloned.Dispatch = &dispatch
	}
	if cfg.Schedule != nil {
		schedule := *cfg.Schedule
		cloned.Schedule = &schedule
	}
	if cfg.Worker != nil {
		worker := *cfg.Worker
		cloned.Worker = &worker
	}
	if cfg.Sanitize != nil {
		sanitize := *cfg.Sanitize
		cloned.Sanitize = &sanitize
	}
	if cfg.Retention != nil {
		retention := *cfg.Retention
		if cfg.Retention.TaskJob != nil {
			taskJob := *cfg.Retention.TaskJob
			taskJob.Overrides = cloneTaskRetentionOverrides(cfg.Retention.TaskJob.Overrides)
			retention.TaskJob = &taskJob
		}
		if cfg.Retention.TaskExecution != nil {
			taskExecution := *cfg.Retention.TaskExecution
			taskExecution.Overrides = cloneTaskRetentionOverrides(cfg.Retention.TaskExecution.Overrides)
			retention.TaskExecution = &taskExecution
		}
		cloned.Retention = &retention
	}
	return &cloned
}

func cloneDocumentConfig(cfg *config.DocumentConfig) *config.DocumentConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.Attachment = cloneAttachmentConfig(cfg.Attachment)
	return &cloned
}

func cloneAttachmentConfig(cfg *config.AttachmentConfig) *config.AttachmentConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.S3 != nil {
		s3 := *cfg.S3
		cloned.S3 = &s3
	}
	return &cloned
}

func cloneTaskRetentionOverrides(overrides map[string]*config.TaskRetentionPolicy) map[string]*config.TaskRetentionPolicy {
	if len(overrides) == 0 {
		return map[string]*config.TaskRetentionPolicy{}
	}
	cloned := make(map[string]*config.TaskRetentionPolicy, len(overrides))
	for method, policy := range overrides {
		if policy == nil {
			cloned[method] = nil
			continue
		}
		policyCopy := *policy
		cloned[method] = &policyCopy
	}
	return cloned
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneCSPConfig(cfg *config.CSPConfig) *config.CSPConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	cloned.Development = cloneCSPDirectives(cfg.Development)
	cloned.Production = cloneCSPDirectives(cfg.Production)
	return &cloned
}

func cloneCSPDirectives(directives config.CSPDirectives) config.CSPDirectives {
	cloned := directives
	cloned.DefaultSrc = cloneStringSlice(directives.DefaultSrc)
	cloned.ScriptSrc = cloneStringSlice(directives.ScriptSrc)
	cloned.StyleSrc = cloneStringSlice(directives.StyleSrc)
	cloned.ImgSrc = cloneStringSlice(directives.ImgSrc)
	cloned.ConnectSrc = cloneStringSlice(directives.ConnectSrc)
	cloned.FontSrc = cloneStringSlice(directives.FontSrc)
	cloned.ObjectSrc = cloneStringSlice(directives.ObjectSrc)
	cloned.MediaSrc = cloneStringSlice(directives.MediaSrc)
	cloned.FrameSrc = cloneStringSlice(directives.FrameSrc)
	cloned.WorkerSrc = cloneStringSlice(directives.WorkerSrc)
	cloned.FrameAncestors = cloneStringSlice(directives.FrameAncestors)
	cloned.FormAction = cloneStringSlice(directives.FormAction)
	cloned.BaseURI = cloneStringSlice(directives.BaseURI)
	cloned.ChildSrc = cloneStringSlice(directives.ChildSrc)
	cloned.ManifestSrc = cloneStringSlice(directives.ManifestSrc)
	return cloned
}

func cloneHSTSConfig(cfg *config.HSTSConfig) *config.HSTSConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

func cloneCSRFConfig(cfg *config.CSRFConfig) *config.CSRFConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	return &cloned
}

// Factory constructs a scope from runtime configuration.
type Factory func(ctx context.Context, input FactoryInput, logger *slog.Logger) Scope

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register registers a named scope factory.
func Register(name string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

// Exists reports whether name is registered.
func Exists(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[name]
	return ok
}

// Keys returns the registered names.
func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

// NewScopeByName constructs a scope by registered name.
func NewScopeByName(name string, ctx context.Context, input FactoryInput, logger *slog.Logger) Scope {
	mu.RLock()
	factory, ok := factories[name]
	mu.RUnlock()
	if !ok {
		if logger != nil {
			logger.Error("scope factory not registered", "scope", name)
		}
		return nil
	}
	return factory(ctx, input, logger)
}

// NewScope constructs a scope from factory input.
func NewScope(ctx context.Context, input FactoryInput, logger *slog.Logger) Scope {
	if input == nil {
		if logger != nil {
			logger.Error("scope input invalid", "reason", "missing environment")
		}
		return nil
	}
	environment := strings.TrimSpace(input.Environment())
	if environment == "" {
		if logger != nil {
			logger.Error("scope input invalid", "reason", "missing environment")
		}
		return nil
	}
	return NewScopeByName(environment, ctx, input, logger)
}
