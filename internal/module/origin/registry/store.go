// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"os"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	DefaultRegistryAlias     = "official"
	DefaultRegistryIndexURL  = "https://index.choysum.dev/v1/index.json"
	legacyDefaultRegistryURL = "https://github.com/project-choysum/registry"
)

type Entry struct {
	IndexURL string `yaml:"indexURL"`
	URL      string `yaml:"url,omitempty"`
	AuthRef  string `yaml:"authRef,omitempty"`
}

type Config struct {
	Version    int              `yaml:"version"`
	Registries map[string]Entry `yaml:"registries"`
}

func defaultConfig() *Config {
	return &Config{
		Version: 1,
		Registries: map[string]Entry{
			DefaultRegistryAlias: {
				IndexURL: DefaultRegistryIndexURL,
			},
		},
	}
}

type Store struct {
	homeDir            string
	defaultChoysumPath string
}

type Option func(*Store)

func WithHomeDir(homeDir string) Option {
	return func(s *Store) {
		homeDir = strings.TrimSpace(homeDir)
		if homeDir != "" {
			s.homeDir = homeDir
		}
	}
}

func WithDefaultChoysumPath(defaultChoysumPath string) Option {
	return func(s *Store) {
		defaultChoysumPath = strings.TrimSpace(defaultChoysumPath)
		if defaultChoysumPath == "" {
			s.defaultChoysumPath = ""
			return
		}
		if absDefaultChoysumPath, err := filepath.Abs(defaultChoysumPath); err == nil {
			defaultChoysumPath = absDefaultChoysumPath
		}
		s.defaultChoysumPath = defaultChoysumPath
	}
}

func NewStore(opts ...Option) *Store {
	s := &Store{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Store) filePath() (string, error) {
	home := strings.TrimSpace(s.homeDir)
	choysumRoot := strings.TrimSpace(s.defaultChoysumPath)
	if choysumRoot == "" {
		return "", xfmt.Errorf("defaultChoysumPath is required")
	}
	if absChoysumRoot, err := filepath.Abs(choysumRoot); err == nil {
		choysumRoot = absChoysumRoot
	}
	choysumRoot = filepath.Clean(choysumRoot)
	if choysumRoot == "." || choysumRoot == string(filepath.Separator) {
		return "", xfmt.Errorf("defaultChoysumPath must be a non-root directory")
	}
	if home == "" {
		return filepath.Join(choysumRoot, "registries.yaml"), nil
	}
	return filepath.Join(home, filepath.Base(choysumRoot), "registries.yaml"), nil
}

func cloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return defaultConfig()
	}
	out := &Config{Version: cfg.Version, Registries: map[string]Entry{}}
	for k, v := range cfg.Registries {
		entry := v
		entry.IndexURL = strings.TrimSpace(entry.IndexURL)
		if entry.IndexURL == "" {
			entry.IndexURL = strings.TrimSpace(entry.URL)
		}
		if strings.TrimSpace(k) == DefaultRegistryAlias && isLegacyDefaultRegistryURL(entry.IndexURL) {
			entry.IndexURL = DefaultRegistryIndexURL
		}
		entry.URL = ""
		out.Registries[k] = entry
	}
	if out.Version == 0 {
		out.Version = 1
	}
	if out.Registries == nil {
		out.Registries = map[string]Entry{}
	}
	defaultEntry, ok := out.Registries[DefaultRegistryAlias]
	if !ok {
		out.Registries[DefaultRegistryAlias] = Entry{IndexURL: DefaultRegistryIndexURL}
	} else if strings.TrimSpace(defaultEntry.IndexURL) == "" {
		defaultEntry.IndexURL = DefaultRegistryIndexURL
		out.Registries[DefaultRegistryAlias] = defaultEntry
	}
	return out
}

func isLegacyDefaultRegistryURL(raw string) bool {
	normalized := strings.TrimSuffix(strings.TrimSpace(raw), "/")
	return normalized == legacyDefaultRegistryURL
}

func (s *Store) Load() (*Config, error) {
	path, err := s.filePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, xfmt.Errorf("read registries config failed: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, xfmt.Errorf("decode registries config failed: %w", err)
	}
	return cloneConfig(cfg), nil
}

func (s *Store) Save(cfg *Config) error {
	path, err := s.filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return xfmt.Errorf("create registries config dir failed: %w", err)
	}
	payload, err := yaml.Marshal(cloneConfig(cfg))
	if err != nil {
		return xfmt.Errorf("encode registries config failed: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return xfmt.Errorf("write registries config failed: %w", err)
	}
	return nil
}

func (s *Store) Resolve(alias string) (Entry, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return Entry{}, xfmt.Errorf("registry alias is empty")
	}
	cfg, err := s.Load()
	if err != nil {
		return Entry{}, err
	}
	entry, ok := cfg.Registries[alias]
	if !ok {
		return Entry{}, xfmt.Errorf("registry alias %s not found", alias)
	}
	entry.IndexURL = strings.TrimSpace(entry.IndexURL)
	if entry.IndexURL == "" {
		return Entry{}, xfmt.Errorf("registry alias %s has empty indexURL", alias)
	}
	return entry, nil
}
