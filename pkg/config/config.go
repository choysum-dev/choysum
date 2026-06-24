// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

var maxProcs = runtime.GOMAXPROCS(0)

const (
	DefaultNPMRegistryURL        = "https://registry.npmjs.org"
	DefaultModuleCatalogIndexURL = "https://index.choysum.dev/v1/index.json"
	DefaultESMUpstreamURL        = "https://esm.sh"
)

type Config struct {
	ConfigPath            string `mapstructure:"-"`
	ModulesPath           string `mapstructure:"modules_path"`
	DistPath              string `mapstructure:"dist_path"`
	NPMRegistryURL        string `mapstructure:"npm_registry_url"`
	ModuleCatalogIndexURL string `mapstructure:"module_catalog_index_url"`
	ESMUpstreamURL        string `mapstructure:"esm_upstream_url"`
	DefaultChoysumPath    string `mapstructure:"default_choysum_path"`
	TmpPath               string `mapstructure:"tmp_path"`

	Log         *LogConfig      `mapstructure:"log"`
	Db          *DbConfig       `mapstructure:"db"`
	Compile     *CompileConfig  `mapstructure:"compile"`
	Server      *ServerConfig   `mapstructure:"server"`
	Auth        *AuthConfig     `mapstructure:"auth"`
	Document    *DocumentConfig `mapstructure:"document"`
	Task        *TaskConfig     `mapstructure:"task"`
	FrontendEnv map[string]any  `mapstructure:"frontendEnv"`
	BackendEnv  map[string]any  `mapstructure:"backendEnv"`
}

func ResolveDefaultChoysumPaths() (string, error) {
	hdir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(filepath.Join(hdir, ".choysum"))
	if err != nil {
		return "", err
	}
	return root, nil
}

func ResolveDefaultTmpPath() (string, error) {
	choysumRoot, err := ResolveDefaultChoysumPaths()
	if err != nil {
		return "", err
	}
	tmpPath, err := filepath.Abs(filepath.Join(choysumRoot, "tmp"))
	if err != nil {
		return "", err
	}
	return tmpPath, nil
}

func ResolveDefaultDistPath() (string, error) {
	choysumRoot, err := ResolveDefaultChoysumPaths()
	if err != nil {
		return "", err
	}
	distPath, err := filepath.Abs(filepath.Join(choysumRoot, "dist"))
	if err != nil {
		return "", err
	}
	return distPath, nil
}

func (c *Config) unmarshal(configPath string, opts ...Option) error {
	c.ConfigPath = configPath

	v := viper.New()
	// Default prefix, can be overridden by WithEnvPrefix/WithViper.
	v.SetEnvPrefix("CHOYSUM")
	if strings.TrimSpace(c.ConfigPath) != "" {
		v.SetConfigFile(c.ConfigPath)
	}

	// Pre-read: allow custom viper behavior and default overrides.
	for _, opt := range opts {
		if err := opt.applyPre(v, c); err != nil {
			return stageError(LoadStageDecode, err)
		}
	}

	if err := applyAuthViperDefaults(v); err != nil {
		return stageError(LoadStageDecode, err)
	}
	if err := applyServerViperDefaults(v); err != nil {
		return stageError(LoadStageDecode, err)
	}
	applyDocumentViperDefaults(v)
	applyTaskViperDefaults(v)
	if err := bindConfigEnv(v); err != nil {
		return stageError(LoadStageDecode, err)
	}

	// Enable environment variable loading after pre hooks and env binding setup.
	v.AutomaticEnv()

	if strings.TrimSpace(c.ConfigPath) != "" {
		if err := v.ReadInConfig(); err != nil {
			return stageError(LoadStageDecode, xfmt.Errorf("read config file failed: %v", err))
		}
	}
	if err := rejectLegacyJWTIdentityCacheKeys(v); err != nil {
		return stageError(LoadStageValidate, err)
	}
	if err := rejectLegacyModuleCatalogConfigKeys(v); err != nil {
		return stageError(LoadStageValidate, err)
	}
	if err := v.Unmarshal(c); err != nil {
		return stageError(LoadStageDecode, xfmt.Errorf("unmarshal config failed: %v", err))
	}

	// Fail-fast: validate & normalize compile.bundleMode.
	if c.Compile == nil {
		c.Compile = NewDefaultCompileConfig()
	}
	mode, err := NormalizeCompileBundleMode(c.Compile.BundleMode)
	if err != nil {
		return stageError(LoadStageValidate, err)
	}
	c.Compile.BundleMode = string(mode)
	if err := c.applyPathInvariants(); err != nil {
		return stageError(LoadStageValidate, err)
	}

	c.NPMRegistryURL = strings.TrimSpace(c.NPMRegistryURL)
	if c.NPMRegistryURL == "" {
		c.NPMRegistryURL = DefaultNPMRegistryURL
	}
	c.ModuleCatalogIndexURL = strings.TrimSpace(c.ModuleCatalogIndexURL)
	if c.ModuleCatalogIndexURL == "" {
		c.ModuleCatalogIndexURL = DefaultModuleCatalogIndexURL
	}
	if err := ValidateModuleCatalogIndexURL(c.ModuleCatalogIndexURL); err != nil {
		return stageError(LoadStageValidate, err)
	}

	if err := c.normalizeAndMergeAuthConfig(); err != nil {
		return stageError(LoadStageApply, err)
	}
	c.normalizeAndMergeTaskConfig()

	if err := c.normalizeAndValidateDocumentAttachmentConfig(v); err != nil {
		return stageError(LoadStageValidate, err)
	}

	// Merge default FrontendEnv/BackendEnv (user config takes precedence).
	c.FrontendEnv = mergeStringMap(NewDefaultFrontendEnv(c), c.FrontendEnv)
	c.BackendEnv = mergeStringMap(NewDefaultBackendEnv(), c.BackendEnv)

	// Post-read: extract custom sections or perform secondary processing.
	for _, opt := range opts {
		if err := opt.applyPost(v, c); err != nil {
			return stageError(LoadStageApply, err)
		}
	}
	if err := c.applyPathInvariants(); err != nil {
		return stageError(LoadStageApply, err)
	}

	c.normalizeJWTKeyPaths()

	if !filepath.IsAbs(c.DistPath) {
		c.DistPath, _ = filepath.Abs(c.DistPath)
	}

	return nil
}

func (c *Config) normalizeJWTKeyPaths() {
	if c.Auth == nil || c.Auth.JWT == nil {
		return
	}
	choysumRoot := strings.TrimSpace(c.DefaultChoysumPath)
	if strings.TrimSpace(c.Auth.JWT.PrivateKeyFile) == "" && choysumRoot != "" {
		c.Auth.JWT.PrivateKeyFile = filepath.Join(choysumRoot, "jwtkeys", "private.pem")
	}
	if strings.TrimSpace(c.Auth.JWT.PublicKeyFile) == "" && choysumRoot != "" {
		c.Auth.JWT.PublicKeyFile = filepath.Join(choysumRoot, "jwtkeys", "public.pem")
	}
	c.Auth.JWT.PrivateKeyFile = normalizePathRelativeToConfig(c.ConfigPath, c.Auth.JWT.PrivateKeyFile)
	c.Auth.JWT.PublicKeyFile = normalizePathRelativeToConfig(c.ConfigPath, c.Auth.JWT.PublicKeyFile)
}

func normalizePathRelativeToConfig(configPath string, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return trimmed
	}
	baseDir := filepath.Dir(configPath)
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	joined := filepath.Join(baseDir, trimmed)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return filepath.Clean(joined)
	}
	return abs
}

func isRootPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." {
		return false
	}
	return filepath.Dir(cleaned) == cleaned
}

func (c *Config) applyPathInvariants() error {
	c.DefaultChoysumPath = normalizePathRelativeToConfig(c.ConfigPath, c.DefaultChoysumPath)
	if strings.TrimSpace(c.DefaultChoysumPath) == "" {
		defaultPath, err := ResolveDefaultChoysumPaths()
		if err != nil {
			return xfmt.Errorf("resolve default_choysum_path failed: %w", err)
		}
		c.DefaultChoysumPath = defaultPath
	}
	if !filepath.IsAbs(c.DefaultChoysumPath) {
		c.DefaultChoysumPath, _ = filepath.Abs(c.DefaultChoysumPath)
	}
	c.DefaultChoysumPath = filepath.Clean(c.DefaultChoysumPath)
	if strings.TrimSpace(c.DefaultChoysumPath) == "" {
		return xfmt.Errorf("default_choysum_path must not be empty")
	}

	c.TmpPath = normalizePathRelativeToConfig(c.ConfigPath, c.TmpPath)
	if strings.TrimSpace(c.TmpPath) == "" {
		c.TmpPath = filepath.Join(c.DefaultChoysumPath, "tmp")
	}
	if !filepath.IsAbs(c.TmpPath) {
		c.TmpPath, _ = filepath.Abs(c.TmpPath)
	}
	c.TmpPath = filepath.Clean(c.TmpPath)
	if strings.TrimSpace(c.TmpPath) == "" {
		return xfmt.Errorf("tmp_path must not be empty")
	}
	if isRootPath(c.TmpPath) {
		return xfmt.Errorf("tmp_path must be a non-root directory")
	}

	c.ModulesPath = normalizePathRelativeToConfig(c.ConfigPath, c.ModulesPath)
	if strings.TrimSpace(c.ModulesPath) == "" {
		c.ModulesPath = filepath.Join(c.DefaultChoysumPath, "modules")
	}
	if !filepath.IsAbs(c.ModulesPath) {
		c.ModulesPath, _ = filepath.Abs(c.ModulesPath)
	}
	c.ModulesPath = filepath.Clean(c.ModulesPath)
	if strings.TrimSpace(c.ModulesPath) == "" {
		return xfmt.Errorf("modules_path must not be empty")
	}

	if strings.TrimSpace(c.DistPath) == "" {
		c.DistPath = filepath.Join(c.DefaultChoysumPath, "dist")
	}
	if !filepath.IsAbs(c.DistPath) {
		c.DistPath, _ = filepath.Abs(c.DistPath)
	}
	c.DistPath = filepath.Clean(c.DistPath)
	if strings.TrimSpace(c.DistPath) == "" {
		return xfmt.Errorf("dist_path must not be empty")
	}

	c.applyDatabaseInvariants()

	return nil
}

func (c *Config) applyDatabaseInvariants() {
	if c.Db == nil {
		c.Db = NewDefaultDbConfig()
	}
	c.Db.Dialect = strings.ToLower(strings.TrimSpace(c.Db.Dialect))
	if c.Db.Dialect == "" {
		c.Db.Dialect = defaultDbDialect
	}
	if c.Db.Dialect == defaultDbDialect && strings.TrimSpace(c.Db.DSN) == "" {
		c.Db.DSN = DefaultSQLiteDSN(c.DefaultChoysumPath)
	}
}

func defaultConfig() *Config {
	modulesPath := ""
	if cwd, err := os.Getwd(); err == nil {
		localModules := filepath.Join(cwd, "modules")
		if info, statErr := os.Lstat(localModules); statErr == nil && info.IsDir() {
			modulesPath = localModules
		}
	}
	if modulesPath != "" {
		var err error
		modulesPath, err = filepath.Abs(modulesPath)
		if err != nil {
			panic(err)
		}
	}

	return &Config{
		ModulesPath:           modulesPath,
		DistPath:              "",
		NPMRegistryURL:        DefaultNPMRegistryURL,
		ModuleCatalogIndexURL: DefaultModuleCatalogIndexURL,
		ESMUpstreamURL:        DefaultESMUpstreamURL,
		DefaultChoysumPath:    "",
		TmpPath:               "",
		Log:                   NewDefaultLogConfig(),
		Db:                    NewDefaultDbConfig(),
		Compile:               NewDefaultCompileConfig(),
		Server:                NewDefaultServerConfig(),
		Document:              NewDefaultDocumentConfig(),
		Task:                  NewDefaultTaskConfig(),
		FrontendEnv:           make(map[string]any),
		BackendEnv:            make(map[string]any),
	}
}

// Merge helper: base + override (override wins).
func mergeStringMap(base, override map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// opts can be empty.
func NewConfig(cfgPath string, opts ...Option) (*Config, error) {
	if err := ensureConfigRootOwnerMap(); err != nil {
		return nil, err
	}

	cfg := defaultConfig()
	// Early default overrides (optional, pre stage will run again).
	for _, opt := range opts {
		_ = opt.applyPre(nil, cfg)
	}
	if err := cfg.unmarshal(cfgPath, opts...); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DefaultConfigPath() (string, error) {
	configPath := "./config.yaml"
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", xfmt.Errorf("stat config file failed: %v", err)
	}

	configFile, err := filepath.Abs(configPath)
	if err != nil {
		return "", xfmt.Errorf("get abs path failed: %v", err)
	}

	return configFile, nil
}
