// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

type stubScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

type factoryTestTransactor struct {
	rootScope Scope
}

type factoryTestTransaction struct {
	ctx     context.Context
	session *Session
}

type factoryTestInput struct {
	environment string
	cfg         *config.Config
}

func (i factoryTestInput) Environment() string {
	return i.environment
}

func factoryTestConfigFromInput(input FactoryInput) *config.Config {
	testInput, ok := input.(factoryTestInput)
	if !ok {
		return nil
	}
	return testInput.cfg
}

func (i factoryTestInput) ModulesPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.ModulesPath
}

func (i factoryTestInput) DistPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.DistPath
}

func (i factoryTestInput) TmpPath() string {
	if i.cfg == nil {
		return ""
	}
	return i.cfg.TmpPath
}

func (i factoryTestInput) CompileBundleMode() string {
	if i.cfg == nil || i.cfg.Compile == nil {
		return ""
	}
	return i.cfg.Compile.BundleMode
}

func (i factoryTestInput) AuthEnabled() bool {
	if i.cfg == nil || i.cfg.Auth == nil {
		return false
	}
	return i.cfg.Auth.Enabled
}

func (i factoryTestInput) DatabaseDialect() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.Dialect
}

func (i factoryTestInput) DatabaseDSN() string {
	if i.cfg == nil || i.cfg.Db == nil {
		return ""
	}
	return i.cfg.Db.DSN
}

func (i factoryTestInput) DatabaseMaxOpenConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxOpenConns
}

func (i factoryTestInput) DatabaseMaxIdleConns() int {
	if i.cfg == nil || i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.MaxIdleConns
}

func (i factoryTestInput) DatabaseConnMaxLifetimeSeconds() int {
	if i.cfg == nil {
		return 0
	}
	if i.cfg.Db == nil {
		return 0
	}
	return i.cfg.Db.ConnMaxLifetime
}

func (e *stubScope) Run(func(Scope) error) error { return nil }

func (e *stubScope) Transactor() Transactor { return factoryTestTransactor{rootScope: e} }

func (e *stubScope) Session() *Session { return nil }

func (e *stubScope) WithContext(ctx context.Context) Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &stubScope{ctx: ctx, cfg: e.cfg, logger: e.logger}
}

func (e *stubScope) Context() context.Context { return e.ctx }

func (e *stubScope) Logger() *slog.Logger { return e.logger }

func (e *stubScope) Config() *config.Config { return e.cfg }

func (e *stubScope) FactoryInput() FactoryInput {
	return factoryTestInput{cfg: e.cfg}
}

func (t factoryTestTransactor) Do(ctx context.Context, opts TransactionOptions, fn TxFunc) error {
	propagation := opts.Propagation
	if propagation == "" {
		propagation = PropagationRequired
	}

	switch propagation {
	case PropagationRequired:
		if existingTx, ok := TransactionFromContext(ctx); ok && existingTx != nil {
			return fn(t.rootScope.WithContext(ctx), existingTx)
		}

		txScope := t.rootScope
		if ctx != nil {
			txScope = t.rootScope.WithContext(ctx)
		}

		tx := &factoryTestTransaction{session: txScope.Session()}
		tx.ctx = ContextWithTransaction(txScope.Context(), tx)
		return fn(txScope.WithContext(tx.ctx), tx)
	case PropagationRequiresNew:
		return ErrRequiresNewUnsupported
	case PropagationNested:
		return ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", ErrInvalidTransactionPropagation, propagation)
	}
}

func (t factoryTestTransactor) Required(ctx context.Context, fn TxFunc) error {
	return t.Do(ctx, TransactionOptions{Propagation: PropagationRequired}, fn)
}

func (t factoryTestTransactor) RequiresNew(ctx context.Context, fn TxFunc) error {
	return t.Do(ctx, TransactionOptions{Propagation: PropagationRequiresNew}, fn)
}

func (t factoryTestTransactor) Nested(ctx context.Context, fn TxFunc) error {
	return t.Do(ctx, TransactionOptions{Propagation: PropagationNested}, fn)
}

func (tx *factoryTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *factoryTestTransaction) Session() *Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *factoryTestTransaction) Savepoint(name string) error {
	if tx == nil || tx.session == nil {
		return ErrSessionUnavailable
	}
	return tx.session.Savepoint(name)
}

func (tx *factoryTestTransaction) RollbackToSavepoint(name string) error {
	if tx == nil || tx.session == nil {
		return ErrSessionUnavailable
	}
	return tx.session.RollbackToSavepoint(name)
}

func (tx *factoryTestTransaction) ReleaseSavepoint(name string) error {
	if tx == nil || tx.session == nil {
		return ErrSessionUnavailable
	}
	return tx.session.ReleaseSavepoint(name)
}

func snapshotFactories() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()
	clone := make(map[string]Factory, len(factories))
	for name, factory := range factories {
		clone[name] = factory
	}
	return clone
}

func restoreFactories(snapshot map[string]Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories = snapshot
}

func testLogger(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buffer, nil))
}

func TestSessionFromContextWithoutTransactionOrScope(t *testing.T) {
	var nilCtx context.Context
	if loaded, ok := SessionFromContext(nilCtx); ok || loaded != nil {
		t.Fatalf("SessionFromContext(nil) = %#v, %v", loaded, ok)
	}
	if loaded, ok := SessionFromContext(context.Background()); ok || loaded != nil {
		t.Fatalf("SessionFromContext(background) = %#v, %v", loaded, ok)
	}
}

func TestContextWithScopeAndScopeFromContext(t *testing.T) {
	scope := &stubScope{ctx: context.Background()}
	var nilCtx context.Context
	ctx := ContextWithScope(nilCtx, scope)

	loaded, ok := ScopeFromContext(ctx)
	if !ok || loaded != scope {
		t.Fatalf("ScopeFromContext() = %#v, %v", loaded, ok)
	}

	if loaded, ok = ScopeFromContext(nilCtx); ok || loaded != nil {
		t.Fatalf("ScopeFromContext(nil) = %#v, %v", loaded, ok)
	}

	ctx = ContextWithScope(context.Background(), nil)
	if loaded, ok = ScopeFromContext(ctx); ok || loaded != nil {
		t.Fatalf("ScopeFromContext(nil scope) = %#v, %v", loaded, ok)
	}
}

func TestFactoryRegisterExistsKeysAndNewByName(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	alphaFactory := func(ctx context.Context, input FactoryInput, logger *slog.Logger) Scope {
		return &stubScope{ctx: ctx, cfg: factoryTestConfigFromInput(input), logger: logger}
	}
	betaFactory := func(ctx context.Context, input FactoryInput, logger *slog.Logger) Scope {
		return &stubScope{ctx: ctx, cfg: factoryTestConfigFromInput(input), logger: logger}
	}
	Register("alpha", alphaFactory)
	Register("beta", betaFactory)

	if !Exists("alpha") || !Exists("beta") {
		t.Fatal("expected registered factories to exist")
	}
	if Exists("missing") {
		t.Fatal("did not expect missing factory to exist")
	}

	keys := Keys()
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"alpha", "beta"}) {
		t.Fatalf("Keys() = %#v, want alpha/beta", keys)
	}

	ctx := context.WithValue(context.Background(), "trace", "alpha")
	cfg := &config.Config{}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	created := NewScopeByName("alpha", ctx, factoryTestInput{environment: "alpha", cfg: cfg}, logger)
	if created == nil {
		t.Fatal("expected NewScopeByName to create scope")
	}
	stub := created.(*stubScope)
	if stub.Context() != ctx || stub.Config() != cfg || stub.Logger() != logger {
		t.Fatalf("unexpected created scope: %#v", stub)
	}
}

func TestNewScopeByNameAndNewScopeLogOnMissingConfig(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	var buffer bytes.Buffer
	logger := testLogger(&buffer)
	if scope := NewScopeByName("missing", context.Background(), factoryTestInput{environment: "missing", cfg: &config.Config{}}, logger); scope != nil {
		t.Fatalf("expected nil scope for missing factory, got %#v", scope)
	}
	if !strings.Contains(buffer.String(), "scope factory not registered") || !strings.Contains(buffer.String(), "scope=missing") {
		t.Fatalf("expected missing factory log, got %q", buffer.String())
	}

	buffer.Reset()
	if scope := NewScope(context.Background(), nil, logger); scope != nil {
		t.Fatalf("expected nil scope for nil config, got %#v", scope)
	}
	if !strings.Contains(buffer.String(), "scope input invalid") || !strings.Contains(buffer.String(), "reason=\"missing environment\"") {
		t.Fatalf("expected invalid input log, got %q", buffer.String())
	}

	buffer.Reset()
	if scope := NewScope(context.Background(), factoryTestInput{cfg: &config.Config{}}, logger); scope != nil {
		t.Fatalf("expected nil scope for missing environment, got %#v", scope)
	}
	if !strings.Contains(buffer.String(), "scope input invalid") || !strings.Contains(buffer.String(), "reason=\"missing environment\"") {
		t.Fatalf("expected invalid input log for missing environment, got %q", buffer.String())
	}
}

func TestNewScopeUsesRegisteredFactory(t *testing.T) {
	snapshot := snapshotFactories()
	t.Cleanup(func() { restoreFactories(snapshot) })
	factories = make(map[string]Factory)

	ctx := context.WithValue(context.Background(), "trace", "scope")
	cfg := &config.Config{Server: &config.ServerConfig{Environment: "chosen"}}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	Register("chosen", func(ctx context.Context, input FactoryInput, logger *slog.Logger) Scope {
		return &stubScope{ctx: ctx, cfg: factoryTestConfigFromInput(input), logger: logger}
	})

	created := NewScope(ctx, factoryTestInput{environment: "chosen", cfg: cfg}, logger)
	if created == nil {
		t.Fatal("expected scope to be created")
	}
	stub := created.(*stubScope)
	if stub.Context() != ctx || stub.Config() != cfg || stub.Logger() != logger {
		t.Fatalf("unexpected created scope: %#v", stub)
	}
}

func TestPathsRuntimeOptionsFromInputUsesPathsInputValues(t *testing.T) {
	cfg := &config.Config{
		ModulesPath: "/tmp/modules",
		DistPath:    "/tmp/dist",
		TmpPath:     "/tmp/tmp",
	}

	options, ok := PathsRuntimeOptionsFromInput(factoryTestInput{cfg: cfg})
	if !ok {
		t.Fatal("PathsRuntimeOptionsFromInput() ok = false, want true")
	}
	if options.ModulesPath != cfg.ModulesPath {
		t.Fatalf("ModulesPath = %q, want %q", options.ModulesPath, cfg.ModulesPath)
	}
	if options.DistPath != cfg.DistPath {
		t.Fatalf("DistPath = %q, want %q", options.DistPath, cfg.DistPath)
	}
	if options.TmpPath != cfg.TmpPath {
		t.Fatalf("TmpPath = %q, want %q", options.TmpPath, cfg.TmpPath)
	}
}
