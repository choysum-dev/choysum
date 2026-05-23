// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package database

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type databaseTestScope struct {
	ctx        context.Context
	cfg        *config.Config
	logger     *slog.Logger
	session    *scope.Session
	runSession *scope.Session
	runCalls   *int
	runCtx     *context.Context
	props      *[]scope.Propagation
}

type databaseTestTransaction struct {
	ctx     context.Context
	session *scope.Session
}

type databaseTestTransactor struct {
	runtimeScope *databaseTestScope
}

func (e *databaseTestScope) Run(fn func(scope.Scope) error) error {
	if e.runCalls != nil {
		*e.runCalls++
	}
	if e.runCtx != nil {
		*e.runCtx = e.ctx
	}
	if e.runSession != nil {
		clone := *e
		clone.session = e.runSession
		return fn(&clone)
	}
	return fn(e)
}

func (e *databaseTestScope) Transactor() scope.Transactor {
	return &databaseTestTransactor{runtimeScope: e}
}

func (e *databaseTestScope) Session() *scope.Session { return e.session }

func (e *databaseTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}

func (e *databaseTestScope) Context() context.Context { return e.ctx }
func (e *databaseTestScope) Logger() *slog.Logger     { return e.logger }
func (e *databaseTestScope) Config() *config.Config   { return e.cfg }
func (e *databaseTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func (u *databaseTestTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if opts.Propagation == "" {
		opts.Propagation = scope.PropagationRequired
	}
	if u.runtimeScope.props != nil {
		*u.runtimeScope.props = append(*u.runtimeScope.props, opts.Propagation)
	}
	txCtx := ctx
	if txCtx == nil {
		txCtx = u.runtimeScope.Context()
	}
	if u.runtimeScope.runCtx != nil {
		*u.runtimeScope.runCtx = txCtx
	}
	if u.runtimeScope.runCalls != nil {
		*u.runtimeScope.runCalls++
	}
	txScope, _ := u.runtimeScope.WithContext(txCtx).(*databaseTestScope)
	if u.runtimeScope.runSession != nil {
		txScope.session = u.runtimeScope.runSession
	}
	tx := &databaseTestTransaction{session: txScope.Session()}
	tx.ctx = scope.ContextWithTransaction(txCtx, tx)
	txScope.ctx = tx.ctx
	return fn(txScope, tx)
}

func (u *databaseTestTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (u *databaseTestTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (u *databaseTestTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (tx *databaseTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *databaseTestTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *databaseTestTransaction) Savepoint(string) error           { return nil }
func (tx *databaseTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *databaseTestTransaction) ReleaseSavepoint(string) error    { return nil }

func newDatabaseTestSession(t *testing.T, name string) *scope.Session {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &scope.Session{DB: db}
}

func newDatabaseTestScope(session *scope.Session) *databaseTestScope {
	return &databaseTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Auth: config.NewDefaultAuthConfig(), Log: config.NewDefaultLogConfig(), Server: config.NewDefaultServerConfig(), Db: config.NewDefaultDbConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: session,
	}
}

func TestDatabaseDriverRegistrationAndStoreLifecycle(t *testing.T) {
	factory, ok := revocation.GetDriver("database")
	if !ok || factory == nil {
		t.Fatal("expected database revocation driver to be registered")
	}

	session := newDatabaseTestSession(t, "registration.db")
	runtimeScope := newDatabaseTestScope(session)
	store, err := NewDatabaseStore(runtimeScope)
	if err != nil {
		t.Fatalf("NewDatabaseStore() error = %v", err)
	}
	defer store.Close()

	if (revokedTokenRecord{}).TableName() != "auth_token" {
		t.Fatalf("unexpected table name: %q", (revokedTokenRecord{}).TableName())
	}
	if !session.Migrator().HasTable("auth_token") {
		t.Fatal("expected auth_token table to be auto-created")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNewDatabaseStoreUsesExistingTableAndSurfacesInitFailures(t *testing.T) {
	t.Run("bootstraps auth_token table via requires-new seam", func(t *testing.T) {
		session := newDatabaseTestSession(t, "bootstrap.db")
		propagations := make([]scope.Propagation, 0, 4)
		runtimeScope := newDatabaseTestScope(session)
		runtimeScope.props = &propagations

		store, err := NewDatabaseStore(runtimeScope)
		if err != nil {
			t.Fatalf("NewDatabaseStore() error = %v", err)
		}
		defer store.Close()

		if len(propagations) == 0 {
			t.Fatal("expected bootstrap to record transaction propagation")
		}
		if got := propagations[0]; got != scope.PropagationRequiresNew {
			t.Fatalf("bootstrap propagation = %q, want requires_new", got)
		}
	})

	t.Run("reuses existing auth_token table", func(t *testing.T) {
		session := newDatabaseTestSession(t, "existing.db")
		if err := session.AutoMigrate(&revokedTokenRecord{}); err != nil {
			t.Fatalf("precreate auth_token table: %v", err)
		}

		store, err := NewDatabaseStore(newDatabaseTestScope(session))
		if err != nil {
			t.Fatalf("NewDatabaseStore() error = %v", err)
		}
		defer store.Close()
	})

	t.Run("returns init errors when database session is unavailable", func(t *testing.T) {
		session := newDatabaseTestSession(t, "closed.db")
		sqlDB, err := session.DB.DB()
		if err != nil {
			t.Fatalf("session.DB.DB() error = %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db: %v", err)
		}

		_, err = NewDatabaseStore(newDatabaseTestScope(session))
		if err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
			t.Fatalf("expected revocation store init error, got %v", err)
		}
	})

}

func TestDatabaseStoreCRUDAndCleanup(t *testing.T) {
	session := newDatabaseTestSession(t, "crud.db")
	runtimeScope := newDatabaseTestScope(session)
	storeIface, err := NewDatabaseStore(runtimeScope)
	if err != nil {
		t.Fatalf("NewDatabaseStore() error = %v", err)
	}
	defer storeIface.Close()
	store := storeIface.(*DatabaseStore)
	var nilCtx context.Context

	if _, err := store.IsRevoked(context.Background(), ""); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidTokenID) {
		t.Fatalf("expected invalid token id error, got %v", err)
	}
	if err := store.RevokeToken(context.Background(), "", "user-1", auth.AccessToken, time.Now().Add(time.Hour), "bad"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidTokenID) {
		t.Fatalf("expected invalid token id revoke error, got %v", err)
	}
	if err := store.RevokeToken(context.Background(), "token-1", "", auth.AccessToken, time.Now().Add(time.Hour), "bad"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidUserID) {
		t.Fatalf("expected invalid user id revoke error, got %v", err)
	}
	if _, err := store.RevokeAllUserTokens(context.Background(), "", "", ""); err == nil || !autherrors.IsAuthError(err, autherrors.ErrInvalidUserID) {
		t.Fatalf("expected invalid user id bulk revoke error, got %v", err)
	}

	revoked, err := store.IsRevoked(context.Background(), "token-1")
	if err != nil || revoked {
		t.Fatalf("IsRevoked(token-1) = %v, %v; want false, nil", revoked, err)
	}

	expiresAt := time.Now().Add(2 * time.Hour).Round(time.Second)
	if err := store.RevokeToken(nilCtx, "token-1", "user-1", auth.AccessToken, expiresAt, "manual revoke"); err != nil {
		t.Fatalf("RevokeToken(insert) error = %v", err)
	}

	var inserted revokedTokenRecord
	if err := session.Where("token_id = ?", "token-1").Take(&inserted).Error; err != nil {
		t.Fatalf("load inserted token: %v", err)
	}
	if !inserted.Revoked || inserted.UserID != "user-1" || inserted.TokenType != string(auth.AccessToken) || inserted.RevocationReason != "manual revoke" {
		t.Fatalf("unexpected inserted record: %#v", inserted)
	}

	revoked, err = store.IsRevoked(context.Background(), "token-1")
	if err != nil || !revoked {
		t.Fatalf("IsRevoked(token-1) = %v, %v; want true, nil", revoked, err)
	}
	if err := store.RevokeToken(context.Background(), "token-1", "user-1", auth.AccessToken, expiresAt, "again"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenAlreadyRevoked) {
		t.Fatalf("expected token already revoked error, got %v", err)
	}

	updatable := revokedTokenRecord{UserID: "user-update", TokenID: "token-update", TokenType: string(auth.RefreshToken), Revoked: false, ExpiresAt: time.Now().Add(4 * time.Hour)}
	if err := session.Create(&updatable).Error; err != nil {
		t.Fatalf("seed updatable token: %v", err)
	}
	if err := store.RevokeToken(context.Background(), "token-update", "user-update", auth.RefreshToken, time.Now().Add(4*time.Hour), "updated in place"); err != nil {
		t.Fatalf("RevokeToken(update existing) error = %v", err)
	}
	var updated revokedTokenRecord
	if err := session.Where("token_id = ?", "token-update").Take(&updated).Error; err != nil {
		t.Fatalf("load updated token: %v", err)
	}
	if !updated.Revoked || updated.RevocationReason != "updated in place" {
		t.Fatalf("unexpected updated token: %#v", updated)
	}
	var updatedCount int64
	if err := session.Model(&revokedTokenRecord{}).Where("token_id = ?", "token-update").Count(&updatedCount).Error; err != nil {
		t.Fatalf("count updated token rows: %v", err)
	}
	if updatedCount != 1 {
		t.Fatalf("expected existing token to be updated in place, rows = %d", updatedCount)
	}

	rows := []revokedTokenRecord{
		{UserID: "user-1", TokenID: "keep", TokenType: string(auth.AccessToken), Revoked: false, ExpiresAt: time.Now().Add(3 * time.Hour)},
		{UserID: "user-1", TokenID: "revoke-me", TokenType: string(auth.RefreshToken), Revoked: false, ExpiresAt: time.Now().Add(3 * time.Hour)},
		{UserID: "user-2", TokenID: "other-user", TokenType: string(auth.AccessToken), Revoked: false, ExpiresAt: time.Now().Add(3 * time.Hour)},
		{UserID: "user-3", TokenID: "expired", TokenType: string(auth.AccessToken), Revoked: false, ExpiresAt: time.Now().Add(-time.Hour)},
		{UserID: "user-4", TokenID: "future", TokenType: string(auth.AccessToken), Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
	}
	if err := session.Create(&rows).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	count, err := store.RevokeAllUserTokens(context.Background(), "user-1", "keep", "")
	if err != nil {
		t.Fatalf("RevokeAllUserTokens() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("RevokeAllUserTokens() count = %d, want 1", count)
	}

	var keep revokedTokenRecord
	if err := session.Where("token_id = ?", "keep").Take(&keep).Error; err != nil {
		t.Fatalf("load keep token: %v", err)
	}
	if keep.Revoked {
		t.Fatal("expected except token to remain active")
	}

	var bulkRevoked revokedTokenRecord
	if err := session.Where("token_id = ?", "revoke-me").Take(&bulkRevoked).Error; err != nil {
		t.Fatalf("load bulk revoked token: %v", err)
	}
	if !bulkRevoked.Revoked || bulkRevoked.RevocationReason != "bulk revocation" {
		t.Fatalf("unexpected bulk revoked token: %#v", bulkRevoked)
	}

	var expiredBefore revokedTokenRecord
	if err := session.Where("token_id = ?", "expired").Take(&expiredBefore).Error; err != nil {
		t.Fatalf("load expired token before cleanup: %v", err)
	}
	if expiredBefore.Revoked {
		t.Fatal("expected expired token to be active before cleanup")
	}

	cleaned, err := store.CleanExpired(context.Background())
	if err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}
	if cleaned < 1 {
		t.Fatalf("CleanExpired() count = %d, want at least 1", cleaned)
	}

	var expired revokedTokenRecord
	if err := session.Where("token_id = ?", "expired").Take(&expired).Error; err != nil {
		t.Fatalf("load expired token: %v", err)
	}
	if !expired.Revoked || expired.RevocationReason != "token expired" {
		t.Fatalf("unexpected expired cleanup token: %#v", expired)
	}

	var future revokedTokenRecord
	if err := session.Where("token_id = ?", "future").Take(&future).Error; err != nil {
		t.Fatalf("load future token: %v", err)
	}
	if future.Revoked {
		t.Fatal("expected unexpired token to remain active")
	}

	count, err = store.RevokeAllUserTokens(context.Background(), "user-2", "", "manual batch")
	if err != nil {
		t.Fatalf("RevokeAllUserTokens(all) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("RevokeAllUserTokens(all) count = %d, want 1", count)
	}

	var otherUser revokedTokenRecord
	if err := session.Where("token_id = ?", "other-user").Take(&otherUser).Error; err != nil {
		t.Fatalf("load other-user token: %v", err)
	}
	if !otherUser.Revoked || otherUser.RevocationReason != "manual batch" {
		t.Fatalf("unexpected full-batch revoked token: %#v", otherUser)
	}
}

func TestDatabaseStoreReturnsWrappedDatabaseErrors(t *testing.T) {
	session := newDatabaseTestSession(t, "missing-table.db")
	store := &DatabaseStore{runtimeScope: newDatabaseTestScope(session), tableName: "missing_table", ctx: context.Background(), cancel: func() {}}

	if _, err := store.IsRevoked(context.Background(), "token-missing"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
		t.Fatalf("expected wrapped IsRevoked database error, got %v", err)
	}

	if err := store.RevokeToken(context.Background(), "token-missing", "user-missing", auth.AccessToken, time.Now().Add(time.Hour), "missing table"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
		t.Fatalf("expected wrapped RevokeToken database error, got %v", err)
	}

	if _, err := store.RevokeAllUserTokens(context.Background(), "user-missing", "", "missing table"); err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
		t.Fatalf("expected wrapped RevokeAllUserTokens database error, got %v", err)
	}

	if _, err := store.CleanExpired(context.Background()); err == nil || !autherrors.IsAuthError(err, autherrors.ErrTokenCleanupFailed) {
		t.Fatalf("expected wrapped CleanExpired database error, got %v", err)
	}
}

func TestDatabaseStoreWithDBUsesContextScopeAndTransactorFallback(t *testing.T) {
	t.Run("nil store or env returns auth error", func(t *testing.T) {
		var store *DatabaseStore
		if err := store.withDB(context.Background(), func(db *gorm.DB) error { return nil }); err == nil || !autherrors.IsAuthError(err, autherrors.ErrRevocationStoreFailed) {
			t.Fatalf("expected uninitialized store error, got %v", err)
		}
	})

	t.Run("context scope overrides env session", func(t *testing.T) {
		baseSession := newDatabaseTestSession(t, "base.db")
		ctxSession := newDatabaseTestSession(t, "ctx.db")
		propagations := make([]scope.Propagation, 0, 2)
		store := &DatabaseStore{runtimeScope: newDatabaseTestScope(baseSession), tableName: "auth_token", ctx: context.Background(), cancel: func() {}}
		store.runtimeScope.(*databaseTestScope).props = &propagations
		ctx := scope.ContextWithScope(context.Background(), newDatabaseTestScope(ctxSession))

		err := store.withDB(ctx, func(db *gorm.DB) error {
			return db.Exec("CREATE TABLE ctx_only (id integer)").Error
		})
		if err != nil {
			t.Fatalf("withDB(context scope) error = %v", err)
		}
		if !ctxSession.Migrator().HasTable("ctx_only") {
			t.Fatal("expected context-bound session to receive write")
		}
		if baseSession.Migrator().HasTable("ctx_only") {
			t.Fatal("expected env session not to be used when context session exists")
		}
		if len(propagations) != 0 {
			t.Fatalf("expected no transactor fallback when context scope exists, got %#v", propagations)
		}
	})

	t.Run("env unit-of-work fallback executes when session is nil", func(t *testing.T) {
		runSession := newDatabaseTestSession(t, "run.db")
		runCalls := 0
		propagations := make([]scope.Propagation, 0, 4)
		runtimeScope := newDatabaseTestScope(nil)
		runtimeScope.runSession = runSession
		runtimeScope.runCalls = &runCalls
		runtimeScope.props = &propagations
		store := &DatabaseStore{runtimeScope: runtimeScope, tableName: "auth_token", ctx: context.Background(), cancel: func() {}}
		var nilCtx context.Context

		err := store.withDB(nilCtx, func(db *gorm.DB) error {
			return db.Exec("CREATE TABLE run_only (id integer)").Error
		})
		if err != nil {
			t.Fatalf("withDB(run fallback) error = %v", err)
		}
		if runCalls != 1 {
			t.Fatalf("expected unit-of-work fallback to be called once, got %d", runCalls)
		}
		if len(propagations) != 1 || propagations[0] != scope.PropagationRequired {
			t.Fatalf("fallback propagation = %#v, want [required]", propagations)
		}
		if !runSession.Migrator().HasTable("run_only") {
			t.Fatal("expected run fallback session to receive write")
		}
	})
}

func TestDatabaseStoreBackgroundCleanupStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &DatabaseStore{ctx: ctx, cancel: cancel}
	done := make(chan struct{})
	go func() {
		store.backgroundCleanup()
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("backgroundCleanup did not stop after cancellation")
	}
}
