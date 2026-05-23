// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultcache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/choysum-dev/choysum/pkg/cache"
)

// ErrInvalidMaxEntries reports that the cache size limit is not positive.
var ErrInvalidMaxEntries = errors.New("invalid cache max entries")

// Options configures the default in-memory cache implementation.
type Options struct {
	// MaxEntries caps the number of live entries retained by the cache.
	MaxEntries int
}

type storedEntry struct {
	value  []byte
	expiry time.Time
}

type inMemoryCache struct {
	mu      sync.Mutex
	entries *lru.Cache[string, *storedEntry]
}

// NewInMemory builds an LRU-backed in-memory cache with per-entry TTL enforcement.
func NewInMemory(opts Options) (cache.Cache, error) {
	if opts.MaxEntries <= 0 {
		return nil, ErrInvalidMaxEntries
	}

	entries, err := lru.New[string, *storedEntry](opts.MaxEntries)
	if err != nil {
		return nil, err
	}

	return &inMemoryCache{entries: entries}, nil
}

func (c *inMemoryCache) Get(_ context.Context, namespace string, key string) ([]byte, bool, error) {
	cacheKey, err := normalizeCacheKey(namespace, key)
	if err != nil {
		return nil, false, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries.Get(cacheKey)
	if !found {
		return nil, false, nil
	}
	if time.Now().After(entry.expiry) {
		c.entries.Remove(cacheKey)
		return nil, false, nil
	}

	return append([]byte(nil), entry.value...), true, nil
}

func (c *inMemoryCache) Set(_ context.Context, namespace string, key string, value []byte, ttl time.Duration) error {
	cacheKey, err := normalizeCacheKey(namespace, key)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return cache.ErrInvalidTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries.Add(cacheKey, &storedEntry{
		value:  append([]byte(nil), value...),
		expiry: time.Now().Add(ttl),
	})
	return nil
}

func (c *inMemoryCache) Delete(_ context.Context, namespace string, key string) error {
	cacheKey, err := normalizeCacheKey(namespace, key)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries.Remove(cacheKey)
	return nil
}

func (c *inMemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries.Purge()
	return nil
}

func normalizeCacheKey(namespace string, key string) (string, error) {
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		return "", cache.ErrInvalidNamespace
	}
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", cache.ErrInvalidKey
	}
	return trimmedNamespace + "\x00" + trimmedKey, nil
}
