// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package storage

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/config"
)

type ProviderFactory func(att *config.AttachmentConfig) (StoredContentDriver, error)

type StoredContentDriverFactory interface {
	NewDriver(provider string, att *config.AttachmentConfig) (StoredContentDriver, error)
}

type registryBackedDriverFactory struct{}

var (
	driverFactoriesMu sync.RWMutex
	driverFactories   = make(map[string]ProviderFactory)
)

func Register(provider string, factory ProviderFactory) {
	name := normalizeProvider(provider)
	if name == "" {
		panic("storage provider name is required")
	}
	if factory == nil {
		panic(fmt.Sprintf("storage provider %q factory is required", name))
	}

	driverFactoriesMu.Lock()
	defer driverFactoriesMu.Unlock()
	if _, exists := driverFactories[name]; exists {
		panic(fmt.Sprintf("storage provider %q is already registered", name))
	}
	driverFactories[name] = factory
}

func Exists(provider string) bool {
	name := normalizeProvider(provider)
	if name == "" {
		return false
	}

	driverFactoriesMu.RLock()
	defer driverFactoriesMu.RUnlock()
	_, exists := driverFactories[name]
	return exists
}

func Providers() []string {
	driverFactoriesMu.RLock()
	defer driverFactoriesMu.RUnlock()

	out := make([]string, 0, len(driverFactories))
	for provider := range driverFactories {
		out = append(out, provider)
	}
	sort.Strings(out)
	return out
}

func NewFactory() StoredContentDriverFactory {
	return registryBackedDriverFactory{}
}

func (registryBackedDriverFactory) NewDriver(provider string, att *config.AttachmentConfig) (StoredContentDriver, error) {
	name := normalizeProvider(provider)
	if name == "" {
		return nil, fmt.Errorf("storage provider name is required")
	}

	driverFactoriesMu.RLock()
	factory := driverFactories[name]
	driverFactoriesMu.RUnlock()
	if factory == nil {
		return nil, fmt.Errorf("stored content driver provider %q is not registered", name)
	}

	return factory(att)
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
