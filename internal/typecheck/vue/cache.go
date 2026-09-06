// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// CachedCoder wraps a Coder and memoizes CreateServiceScript by
// path + source + CurrentDirectory.
type CachedCoder struct {
	inner Coder
	mu    sync.Mutex
	cache map[string]ServiceScript
}

// NewCachedCoder returns a Coder that caches successful codegen results.
func NewCachedCoder(inner Coder) *CachedCoder {
	return &CachedCoder{
		inner: inner,
		cache: make(map[string]ServiceScript),
	}
}

func cacheKey(path, source string, opts CodegenOptions) string {
	sum := sha256.Sum256([]byte(path + "\x00" + source + "\x00" + opts.CurrentDirectory))
	return hex.EncodeToString(sum[:])
}

// CreateServiceScript returns a cached ServiceScript when path, source, and
// CurrentDirectory match a prior call.
func (c *CachedCoder) CreateServiceScript(path, source string, opts CodegenOptions) (ServiceScript, error) {
	if c == nil || c.inner == nil {
		return ServiceScript{}, fmt.Errorf("vue: CachedCoder is nil or missing inner Coder")
	}
	key := cacheKey(path, source, opts)
	c.mu.Lock()
	if hit, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return hit, nil
	}
	c.mu.Unlock()

	script, err := c.inner.CreateServiceScript(path, source, opts)
	if err != nil {
		return ServiceScript{}, err
	}

	c.mu.Lock()
	c.cache[key] = script
	c.mu.Unlock()
	return script, nil
}

// Close closes the inner Coder when it implements Close.
func (c *CachedCoder) Close() error {
	if c == nil {
		return nil
	}
	if cl, ok := c.inner.(interface{ Close() error }); ok {
		return cl.Close()
	}
	return nil
}
