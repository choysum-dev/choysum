// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package db

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/config"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
)

type storedContentDriver struct{}

// NewStoredContentDriver builds the database-backed stored content driver.
func NewStoredContentDriver(att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
	_ = att
	return storedContentDriver{}, nil
}

func (storedContentDriver) Provider() string {
	return "db"
}

func (storedContentDriver) Put(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
	_ = ctx
	return pkgstorage.PayloadMutation{BlobData: append([]byte(nil), input.Body...)}, nil
}

func (storedContentDriver) Open(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error) {
	_ = ctx
	return append([]byte(nil), record.BlobData...), nil
}

func (storedContentDriver) Delete(ctx context.Context, record pkgstorage.StoredContentRecord) error {
	_ = ctx
	_ = record
	return nil
}
