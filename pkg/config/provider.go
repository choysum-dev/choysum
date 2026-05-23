// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

// Provider is the minimal config infrastructure seam for loading Config.
// It intentionally focuses on static loading only; watch/reload concerns are out of scope.
// Implementations should return fail-fast errors when any decode/validate/apply stage fails.
type Provider interface {
	Load(configPath string, opts ...Option) (*Config, error)
}

// FileProvider is the default provider backed by local file loading.
type FileProvider struct{}

func (FileProvider) Load(configPath string, opts ...Option) (*Config, error) {
	return NewConfig(configPath, opts...)
}

// LoadWithProvider loads config through the given provider.
// When provider is nil, FileProvider is used.
func LoadWithProvider(provider Provider, configPath string, opts ...Option) (*Config, error) {
	if provider == nil {
		provider = FileProvider{}
	}
	return provider.Load(configPath, opts...)
}
