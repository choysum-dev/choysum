// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"gorm.io/gorm"
)

// RepositoryDeps provides dependencies for the stored content repository.
type RepositoryDeps struct {
	DB *gorm.DB
}

type storedContentRow struct {
	ID          string `gorm:"column:id"`
	Provider    string `gorm:"column:provider"`
	LocatorJSON string `gorm:"column:locator_json"`
	BlobData    []byte `gorm:"column:blob_data"`
	Status      string `gorm:"column:status"`
	CompanyID   string `gorm:"column:company_id"`
}

type storedContentRepository struct {
	db *gorm.DB
}

type s3Locator struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// NewStoredContentRepository builds the stored content repository backed by a database session.
func NewStoredContentRepository(deps RepositoryDeps) pkgstorage.StoredContentRepository {
	if deps.DB == nil {
		return nil
	}
	return &storedContentRepository{db: deps.DB}
}

func (r *storedContentRepository) dbForContext(ctx context.Context) (*gorm.DB, error) {
	if db, ok := scope.DBForScope(ctx, nil); ok {
		return db, nil
	}
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("document database session is required")
	}
	if ctx != nil {
		return r.db.WithContext(ctx), nil
	}
	return r.db, nil
}

func (r *storedContentRepository) Create(ctx context.Context, input pkgstorage.CreateStoredContentInput) error {
	tx, err := r.dbForContext(ctx)
	if err != nil {
		return err
	}

	provider := normalizeProvider(input.Provider)
	locatorJSON, err := normalizeLocatorJSON(provider, input.LocatorJSON)
	if err != nil {
		return err
	}

	return tx.Table(pkgstorage.DocumentStoredContentTable).Create(map[string]any{
		"id":           strings.TrimSpace(input.ID),
		"provider":     provider,
		"locator_json": nullableText(locatorJSON),
		"blob_data":    cloneBytes(input.BlobData),
		"status":       strings.TrimSpace(input.Status),
		"company_id":   strings.TrimSpace(input.CompanyID),
		"created_at":   input.CreatedAt,
		"updated_at":   input.UpdatedAt,
	}).Error
}

func (r *storedContentRepository) GetByID(ctx context.Context, id string) (pkgstorage.StoredContentRecord, error) {
	tx, err := r.dbForContext(ctx)
	if err != nil {
		return pkgstorage.StoredContentRecord{}, err
	}

	var row storedContentRow
	err = tx.Table(pkgstorage.DocumentStoredContentTable).
		Select("id, provider, locator_json, blob_data, status, company_id").
		Where("id = ?", strings.TrimSpace(id)).
		Take(&row).Error
	if err != nil {
		return pkgstorage.StoredContentRecord{}, err
	}

	provider := normalizeProvider(row.Provider)
	locatorJSON, err := normalizeLocatorJSON(provider, row.LocatorJSON)
	if err != nil {
		return pkgstorage.StoredContentRecord{}, err
	}

	return pkgstorage.StoredContentRecord{
		ID:          row.ID,
		Provider:    provider,
		BlobData:    cloneBytes(row.BlobData),
		LocatorJSON: locatorJSON,
		Status:      strings.TrimSpace(row.Status),
		CompanyID:   strings.TrimSpace(row.CompanyID),
	}, nil
}

func (r *storedContentRepository) MarkDeleted(ctx context.Context, id string, updatedAt time.Time) (int64, error) {
	tx, err := r.dbForContext(ctx)
	if err != nil {
		return 0, err
	}

	result := tx.Table(pkgstorage.DocumentStoredContentTable).
		Where("id = ?", strings.TrimSpace(id)).
		Updates(map[string]any{
			"status":     "deleted",
			"blob_data":  nil,
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (r *storedContentRepository) Exists(ctx context.Context, id string) (bool, error) {
	tx, err := r.dbForContext(ctx)
	if err != nil {
		return false, err
	}

	var count int64
	err = tx.Table(pkgstorage.DocumentStoredContentTable).
		Where("id = ?", strings.TrimSpace(id)).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func normalizeProvider(provider string) string {
	name := strings.ToLower(strings.TrimSpace(provider))
	if name == "" {
		return "db"
	}
	return name
}

func normalizeLocatorJSON(provider string, locatorJSON string) (string, error) {
	text := strings.TrimSpace(locatorJSON)
	if text == "" {
		return "", nil
	}
	if !json.Valid([]byte(text)) {
		return "", fmt.Errorf("locator json for provider %q must be valid json", provider)
	}

	if provider == "s3" {
		var locator s3Locator
		if err := json.Unmarshal([]byte(text), &locator); err != nil {
			return "", fmt.Errorf("parse s3 locator json: %w", err)
		}
		locator.Bucket = strings.TrimSpace(locator.Bucket)
		locator.Key = strings.TrimSpace(locator.Key)
		if locator.Bucket == "" || locator.Key == "" {
			return "", fmt.Errorf("s3 locator json requires bucket and key")
		}
		return text, nil
	}
	return text, nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	return append([]byte(nil), value...)
}
