// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type revocationTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

func (f *revocationTestScope) Run(fn func(scope.Scope) error) error { return fn(f) }
func (f *revocationTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(f)
}
func (f *revocationTestScope) Session() *scope.Session { return nil }
func (f *revocationTestScope) WithContext(ctx context.Context) scope.Scope {
	return &revocationTestScope{ctx: ctx, cfg: f.cfg, logger: f.logger}
}
func (f *revocationTestScope) Context() context.Context { return f.ctx }
func (f *revocationTestScope) Logger() *slog.Logger     { return f.logger }
func (f *revocationTestScope) Config() *config.Config   { return f.cfg }
func (f *revocationTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(f.cfg)
}

type fakeStore struct{}

func (fakeStore) IsRevoked(context.Context, string) (bool, error) { return false, nil }
func (fakeStore) RevokeToken(context.Context, string, string, auth.TokenType, time.Time, string) error {
	return nil
}
func (fakeStore) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 0, nil
}
func (fakeStore) CleanExpired(context.Context) (int, error) { return 0, nil }
func (fakeStore) Close() error                              { return nil }

func newRevocationScope() *revocationTestScope {
	return &revocationTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth: config.NewDefaultAuthConfig(),
			Log:  config.NewDefaultLogConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func withDriversSnapshot(t *testing.T) {
	t.Helper()
	driversMu.Lock()
	orig := make(map[string]StoreFactory, len(drivers))
	for k, v := range drivers {
		orig[k] = v
	}
	drivers = map[string]StoreFactory{}
	driversMu.Unlock()
	t.Cleanup(func() {
		driversMu.Lock()
		drivers = orig
		driversMu.Unlock()
	})
}

func TestDefaultOptionsTokenAndUtils(t *testing.T) {
	options := DefaultOptions()
	if options.TableName != "token" || options.CleanupInterval != 60*time.Minute {
		t.Fatalf("unexpected default options: %#v", options)
	}
	if !IsValidTokenID("token") || IsValidTokenID("") {
		t.Fatal("token id validation mismatch")
	}
	if !IsValidUserID("user") || IsValidUserID("") {
		t.Fatal("user id validation mismatch")
	}
	if !IsExpired(time.Now().Add(-time.Second)) || IsExpired(time.Now().Add(time.Hour)) {
		t.Fatal("expiry helper mismatch")
	}

	token := NewToken("token-1", "user-1", auth.RefreshToken, time.Now().Add(time.Hour), "reason")
	if token.GetTokenID() != "token-1" || token.GetUserID() != "user-1" || token.GetTokenType() != auth.RefreshToken || token.GetReason() != "reason" {
		t.Fatalf("unexpected token getters: %#v", token)
	}
}

func TestDriverRegistrationLookupAndPanics(t *testing.T) {
	withDriversSnapshot(t)
	RegisterDriver("fake", func(scope.Scope) (Store, error) { return fakeStore{}, nil })
	if _, ok := GetDriver("fake"); !ok {
		t.Fatal("expected fake driver to be registered")
	}
	if _, err := GetDriverWithError("missing"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrDriverNotRegistered) {
		t.Fatalf("expected missing driver error, got %v", err)
	}
	names := GetDrivers()
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"fake"}) {
		t.Fatalf("drivers = %#v, want fake", names)
	}

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic for %s", name)
			}
		}()
		fn()
	}
	assertPanics("nil factory", func() { RegisterDriver("nil", nil) })
	assertPanics("duplicate factory", func() {
		RegisterDriver("dup", func(scope.Scope) (Store, error) { return fakeStore{}, nil })
		RegisterDriver("dup", func(scope.Scope) (Store, error) { return fakeStore{}, nil })
	})
}

func TestNewStoreHandlesMissingConfigDriverErrorsAndSuccess(t *testing.T) {
	withDriversSnapshot(t)
	runtimeScope := newRevocationScope()
	runtimeScope.cfg.Auth.JWT = nil
	if _, err := NewStore(runtimeScope); err == nil || !autherrors.IsAuthError(err, autherrors.ErrMissingConfiguration) {
		t.Fatalf("expected missing jwt config error, got %v", err)
	}

	runtimeScope = newRevocationScope()
	runtimeScope.cfg.Auth.JWT.RevokeStore = "missing"
	if _, err := NewStore(runtimeScope); err == nil || !autherrors.IsAuthError(err, autherrors.ErrDriverNotRegistered) {
		t.Fatalf("expected missing driver error, got %v", err)
	}

	RegisterDriver("defaulted-memory", func(scope.Scope) (Store, error) { return fakeStore{}, nil })
	runtimeScope.cfg.Auth.JWT.RevokeStore = "defaulted-memory"
	if store, err := NewStore(runtimeScope); err != nil || store == nil {
		t.Fatalf("expected store from registered driver, got store=%#v err=%v", store, err)
	}

	RegisterDriver("broken", func(scope.Scope) (Store, error) { return nil, errors.New("boom") })
	runtimeScope.cfg.Auth.JWT.RevokeStore = "broken"
	if _, err := NewStore(runtimeScope); err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
		t.Fatalf("expected wrapped revocation store error, got %v", err)
	}
}
