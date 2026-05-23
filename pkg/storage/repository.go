// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package storage

import (
	"context"
	"time"
)

const DocumentStoredContentTable = "document_stored_content"

type CreateStoredContentInput struct {
	ID          string
	Provider    string
	BlobData    []byte
	LocatorJSON string
	Status      string
	CompanyID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StoredContentRecord struct {
	ID          string
	Provider    string
	BlobData    []byte
	LocatorJSON string
	Status      string
	CompanyID   string
}

type StoredContentRepository interface {
	Create(ctx context.Context, input CreateStoredContentInput) error
	GetByID(ctx context.Context, id string) (StoredContentRecord, error)
	MarkDeleted(ctx context.Context, id string, updatedAt time.Time) (rowsAffected int64, err error)
	Exists(ctx context.Context, id string) (bool, error)
}
