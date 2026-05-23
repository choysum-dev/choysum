// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cache

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidNamespace = errors.New("invalid cache namespace")
	ErrInvalidKey       = errors.New("invalid cache key")
	ErrInvalidTTL       = errors.New("invalid cache ttl")
)

// Cache freezes the minimal namespaced expiring key/value semantics for the
// first coordinated cache adopters. Pure in-process optimization caches should
// stay in implementation packages instead of depending on this seam.
type Cache interface {
	Get(ctx context.Context, namespace string, key string) ([]byte, bool, error)
	Set(ctx context.Context, namespace string, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, namespace string, key string) error
	Close() error
}
