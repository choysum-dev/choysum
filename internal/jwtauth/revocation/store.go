// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"context"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Store defines the token revocation storage interface.
type Store interface {
	// IsRevoked reports whether a token has been revoked.
	IsRevoked(ctx context.Context, tokenID string) (bool, error)

	// RevokeToken revokes a token.
	//
	// tokenID: token ID to revoke
	// userID: user ID that owns the token
	// tokenType: token type
	// expiresAt: token expiration time
	// reason: revocation reason recorded in storage
	RevokeToken(ctx context.Context, tokenID string, userID string, tokenType auth.TokenType, expiresAt time.Time, reason string) error

	// RevokeAllUserTokens revokes all tokens for a user except exceptTokenID.
	//
	// userID: user ID
	// exceptTokenID: token ID to exclude, if any
	// reason: revocation reason recorded in storage
	// Returns the number of revoked tokens and any error.
	RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error)

	// CleanExpired removes expired revocation records.
	// Returns the number of cleaned records and any error.
	CleanExpired(ctx context.Context) (int, error)

	// Close closes the store and releases resources.
	Close() error
}

// StoreFactory defines the function type used to create stores.
type StoreFactory func(runtimeScope scope.Scope) (Store, error)

// StoreOptions defines store options.
type StoreOptions struct {
	// TableName is the storage table name.
	TableName string

	// CleanupInterval controls how often expired entries are removed.
	CleanupInterval time.Duration
}

// DefaultOptions returns the default store options.
func DefaultOptions() StoreOptions {
	return StoreOptions{
		TableName:       "token",
		CleanupInterval: 60 * time.Minute,
	}
}

// Driver registry and lock.
var (
	driversMu sync.RWMutex
	drivers   = make(map[string]StoreFactory)
)

// RegisterDriver registers a token revocation store driver.
func RegisterDriver(name string, factory StoreFactory) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if factory == nil {
		panic("revocation: RegisterDriver factory is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("revocation: RegisterDriver called twice for driver " + name)
	}
	drivers[name] = factory
}

// GetDriver returns the driver factory for the given name.
func GetDriver(name string) (StoreFactory, bool) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	factory, ok := drivers[name]
	return factory, ok
}

// GetDriverWithError returns a driver factory or a descriptive error.
func GetDriverWithError(name string) (StoreFactory, error) {
	factory, ok := GetDriver(name)
	if !ok {
		return nil, autherrors.NewAuthErrorf(autherrors.ErrDriverNotRegistered,
			"revocation store driver '%s' is not registered", name)
	}
	return factory, nil
}

// GetDrivers returns all registered driver names.
func GetDrivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var names []string
	for name := range drivers {
		names = append(names, name)
	}
	return names
}
