// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package storage

import "context"

type PutPayloadInput struct {
	UploadID       string
	CompanyID      string
	ContentType    string
	ChecksumSHA256 string
	Body           []byte
}

type PayloadMutation struct {
	BlobData    []byte
	LocatorJSON string
}

type StoredContentDriver interface {
	Provider() string
	Put(ctx context.Context, input PutPayloadInput) (PayloadMutation, error)
	Open(ctx context.Context, record StoredContentRecord) ([]byte, error)
	Delete(ctx context.Context, record StoredContentRecord) error
}
