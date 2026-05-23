// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *DatabaseStore) withDB(ctx context.Context, fn func(db *gorm.DB) error) error {
	if s == nil || s.runtimeScope == nil {
		return autherrors.NewAuthError(autherrors.ErrRevocationStoreFailed, "revocation store is not initialized")
	}
	if ctx == nil {
		ctx = s.ctx
	}

	// Preferred: reuse the scope-bound DB handle when ctx or scope already carries
	// one so auth_token writes stay on the active transaction/connection.
	if db, ok := scope.DBForScope(ctx, s.runtimeScope); ok {
		return fn(db)
	}

	txRoot := s.runtimeScope.WithContext(ctx)
	return txRoot.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
		sess := tx.Session()
		if sess == nil || sess.DB == nil {
			return autherrors.NewAuthError(autherrors.ErrRevocationStoreFailed, "database session is unavailable")
		}
		db := sess.DB.WithContext(ctx)
		return fn(db)
	})
}

// DatabaseStore persists revoked tokens in a database.
type DatabaseStore struct {
	runtimeScope scope.Scope
	tableName    string
	ctx          context.Context
	cancel       context.CancelFunc
}

// revokedTokenRecord is a minimal schema for the revocation store.
//
// Historically this store assumed `information_schema.tables` existed, which
// breaks on sqlite. We keep the schema minimal and aligned with the raw SQL
// used by this store.
type revokedTokenRecord struct {
	ID               string         `gorm:"column:id;type:varchar(20);primaryKey"`
	CreatedAt        time.Time      `gorm:"column:created_at;index"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;index"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
	UserID           string         `gorm:"column:user_id;type:varchar(20);index"`
	TokenID          string         `gorm:"column:token_id;type:varchar(36);uniqueIndex"`
	TokenType        string         `gorm:"column:token_type;type:varchar(10);index"`
	Revoked          bool           `gorm:"column:revoked;index"`
	RevokedAt        time.Time      `gorm:"column:revoked_at;index"`
	RevocationReason string         `gorm:"column:revocation_reason;type:varchar(255)"`
	ExpiresAt        time.Time      `gorm:"column:expires_at;index"`
	Metadata         datatypes.JSON `gorm:"column:metadata"`
}

func (revokedTokenRecord) TableName() string {
	return "auth_token"
}

func (r *revokedTokenRecord) BeforeCreate(_ *gorm.DB) error {
	if strings.TrimSpace(r.ID) == "" {
		r.ID = xid.New().String()
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
	return nil
}

// NewDatabaseStore creates a database-backed revocation store.
func NewDatabaseStore(runtimeScope scope.Scope) (revocation.Store, error) {
	// Create the background context.
	ctx, cancel := context.WithCancel(runtimeScope.Context())

	// Table name shared with the token model.
	tableName := "auth_token"

	// Create the store.
	store := &DatabaseStore{
		runtimeScope: runtimeScope,
		tableName:    tableName,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Validate and repair the table schema.
	txRoot := runtimeScope.WithContext(runtimeScope.Context())
	if err := txRoot.Transactor().RequiresNew(txRoot.Context(), func(txScope scope.Scope, _ scope.Transaction) error {
		migrator := txScope.Session().Migrator()
		if err := txScope.Session().AutoMigrate(&revokedTokenRecord{}); err != nil {
			return autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed, "failed to create revocation store table")
		}
		if migrator.HasTable(tableName) {
			if err := ensureAuthTokenSchemaCompatibility(txScope.Session().DB, tableName); err != nil {
				return autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed, "failed to repair revocation store schema")
			}
			return nil
		}
		return autherrors.NewAuthError(autherrors.ErrRevocationStoreFailed, "failed to create revocation store table")
	}); err != nil {
		cancel()
		return nil, err
	}

	// Start the background cleanup task.
	go store.backgroundCleanup()

	return store, nil
}

// IsRevoked reports whether the token has been revoked.
func (s *DatabaseStore) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	if !revocation.IsValidTokenID(tokenID) {
		return false, autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token ID cannot be empty")
	}
	if ctx == nil {
		ctx = s.ctx
	}

	var revoked bool
	err := s.withDB(ctx, func(db *gorm.DB) error {
		// Query whether the token has been revoked.
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE token_id = ? AND revoked = true", s.tableName)
		result := db.Raw(query, tokenID).Scan(&count)
		if result.Error != nil {
			return autherrors.WrapAuthError(result.Error, autherrors.ErrRevocationStoreFailed, "failed to query token state")
		}
		revoked = count > 0
		return nil
	})

	return revoked, err
}

// RevokeToken revokes a token.
// It uses a reduced-query path to minimize database round trips.
func (s *DatabaseStore) RevokeToken(ctx context.Context, tokenID string, userID string, tokenType auth.TokenType, expiresAt time.Time, reason string) error {
	if !revocation.IsValidTokenID(tokenID) {
		return autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token ID cannot be empty")
	}
	if !revocation.IsValidUserID(userID) {
		return autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}
	if ctx == nil {
		ctx = s.ctx
	}

	return s.withDB(ctx, func(db *gorm.DB) error {
		// Try updating the token state directly and inspect affected rows.
		now := time.Now()
		updateQuery := fmt.Sprintf(`
            UPDATE %s 
            SET revoked = true, revoked_at = ?, revocation_reason = ? 
            WHERE token_id = ? AND revoked = false
        `, s.tableName)

		result := db.Exec(updateQuery, now, reason, tokenID)
		if result.Error != nil {
			return autherrors.WrapAuthError(result.Error, autherrors.ErrRevocationStoreFailed, "failed to revoke token")
		}

		// Affected rows indicate the token existed and is now revoked.
		if result.RowsAffected > 0 {
			return nil
		}

		// No rows were updated, so the token may be missing or already revoked.
		// Check whether the token has already been revoked.
		var revokedCount int64
		checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE token_id = ? AND revoked = true", s.tableName)
		if err := db.Raw(checkQuery, tokenID).Scan(&revokedCount).Error; err != nil {
			return autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed, "failed to check token state")
		}

		if revokedCount > 0 {
			// The token has already been revoked.
			return autherrors.NewAuthError(autherrors.ErrTokenAlreadyRevoked,
				fmt.Sprintf("token %s has been revoked", tokenID))
		}

		// The token does not exist yet, so create a revocation record.
		err := db.Table(s.tableName).Create(&revokedTokenRecord{
			UserID:           userID,
			TokenID:          tokenID,
			TokenType:        string(tokenType),
			Revoked:          true,
			RevokedAt:        now,
			RevocationReason: reason,
			ExpiresAt:        expiresAt,
		}).Error

		if err != nil {
			return autherrors.WrapAuthError(err, autherrors.ErrRevocationStoreFailed, "failed to create revocation record")
		}

		return nil
	})
}

func ensureAuthTokenSchemaCompatibility(db *gorm.DB, tableName string) error {
	if db == nil {
		return nil
	}
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	columnTypes, err := db.Migrator().ColumnTypes(tableName)
	if err != nil {
		return err
	}

	needsIDTypeRepair := false
	for _, col := range columnTypes {
		if !strings.EqualFold(col.Name(), "id") {
			continue
		}
		dbType := strings.ToLower(strings.TrimSpace(col.DatabaseTypeName()))
		if strings.Contains(dbType, "char") || strings.Contains(dbType, "text") {
			return nil
		}
		if strings.Contains(dbType, "int") || strings.Contains(dbType, "serial") || strings.Contains(dbType, "numeric") {
			needsIDTypeRepair = true
		}
		break
	}

	if !needsIDTypeRepair {
		return nil
	}

	quotedTable := fmt.Sprintf("\"%s\"", tableName)
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN id DROP DEFAULT", quotedTable)).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN id TYPE varchar(20) USING id::text", quotedTable)).Error; err != nil {
		return err
	}

	return nil
}

func (s *DatabaseStore) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	if !revocation.IsValidUserID(userID) {
		return 0, autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}
	if ctx == nil {
		ctx = s.ctx
	}

	// Use the default revocation reason when none is provided.
	if reason == "" {
		reason = "bulk revocation"
	}

	var count int64
	err := s.withDB(ctx, func(db *gorm.DB) error {
		var query string
		var result *gorm.DB
		now := time.Now()

		// Build the query conditions.
		if exceptTokenID == "" {
			query = fmt.Sprintf(`
                UPDATE %s 
                SET revoked = true, revoked_at = ?, revocation_reason = ? 
                WHERE user_id = ? AND revoked = false
            `, s.tableName)
			result = db.Exec(query, now, reason, userID)
		} else {
			query = fmt.Sprintf(`
                UPDATE %s 
                SET revoked = true, revoked_at = ?, revocation_reason = ? 
                WHERE user_id = ? AND token_id != ? AND revoked = false
            `, s.tableName)
			result = db.Exec(query, now, reason, userID, exceptTokenID)
		}

		if result.Error != nil {
			return autherrors.WrapAuthError(result.Error, autherrors.ErrRevocationStoreFailed, "failed to revoke user tokens")
		}

		count = result.RowsAffected
		return nil
	})

	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// CleanExpired marks expired token records as revoked.
func (s *DatabaseStore) CleanExpired(ctx context.Context) (int, error) {
	var count int64
	if ctx == nil {
		ctx = s.ctx
	}

	err := s.withDB(ctx, func(db *gorm.DB) error {
		// Mark expired tokens that are not yet flagged as revoked.
		query := fmt.Sprintf(`
            UPDATE %s 
            SET revoked = true, revoked_at = ?, revocation_reason = ? 
            WHERE expires_at < ? AND revoked = false
        `, s.tableName)

		result := db.Exec(query, time.Now(), "token expired", time.Now())
		if result.Error != nil {
			return autherrors.WrapAuthError(result.Error, autherrors.ErrTokenCleanupFailed, "failed to clean expired tokens")
		}

		count = result.RowsAffected
		return nil
	})

	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// Close closes the store and releases its resources.
func (s *DatabaseStore) Close() error {
	s.cancel() // Stop the background cleanup task.
	return nil
}

// backgroundCleanup periodically cleans expired records.
func (s *DatabaseStore) backgroundCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := s.CleanExpired(s.ctx); err != nil {
				// Logging could be added here, but cleanup should keep running.
				// log.Printf("failed to clean expired tokens from database store: %v", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}
