// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	documentdb "github.com/choysum-dev/choysum/internal/document/storage/db"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"github.com/rs/xid"
)

const errorCSVMimeType = "text/csv; charset=utf-8"

var (
	messagesToCSVBytes = defaultMessagesToCSVBytes
	storeArtifact      = defaultStoreArtifact
)

// WriteErrorArtifact persists report messages as a CSV document and sets report.ArtifactRef.
func WriteErrorArtifact(ctx context.Context, runtimeScope scope.Scope, companyID string, report *importpkg.Report) error {
	if report == nil || len(report.Messages) == 0 {
		return nil
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return fmt.Errorf("company id is required to store import error artifact")
	}

	raw, err := messagesToCSVBytes(report.Messages)
	if err != nil {
		return fmt.Errorf("encode error artifact csv: %w", err)
	}

	ref, err := storeArtifact(ctx, runtimeScope, companyID, raw, errorCSVMimeType)
	if err != nil {
		return err
	}
	report.ArtifactRef = ref
	return nil
}

func defaultMessagesToCSVBytes(messages []importpkg.Message) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"type", "row", "field", "code", "text", "record_ref"}); err != nil {
		return nil, err
	}
	for _, msg := range messages {
		row := []string{
			string(msg.Type),
			strconv.Itoa(msg.Row),
			msg.Field,
			msg.Code,
			msg.Text,
			msg.RecordRef,
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func defaultStoreArtifact(ctx context.Context, runtimeScope scope.Scope, companyID string, body []byte, contentType string) (string, error) {
	if runtimeScope == nil {
		return "", fmt.Errorf("runtime scope is required")
	}
	session, ok := scope.SessionForScope(ctx, runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return "", fmt.Errorf("database session is required")
	}

	driver, err := documentdb.NewStoredContentDriver(nil)
	if err != nil {
		return "", fmt.Errorf("create stored content driver: %w", err)
	}
	mutation, err := driver.Put(ctx, pkgstorage.PutPayloadInput{
		CompanyID:      companyID,
		ContentType:    contentType,
		ChecksumSHA256: sha256Hex(body),
		Body:           body,
	})
	if err != nil {
		return "", fmt.Errorf("put stored content payload: %w", err)
	}

	storedContentID := xid.New().String()
	now := time.Now().UTC()
	repo := documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: session.DB})
	if repo == nil {
		return "", fmt.Errorf("stored content repository is unavailable")
	}
	if err := repo.Create(ctx, pkgstorage.CreateStoredContentInput{
		ID:          storedContentID,
		Provider:    driver.Provider(),
		BlobData:    mutation.BlobData,
		LocatorJSON: mutation.LocatorJSON,
		Status:      "active",
		CompanyID:   companyID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return "", fmt.Errorf("create stored content: %w", err)
	}

	contentID := xid.New().String()
	if err := session.DB.WithContext(ctx).Table("document_attachment_content").Create(map[string]any{
		"id":                contentID,
		"stored_content_id": storedContentID,
		"company_id":        companyID,
		"status":            "active",
		"mime_type":         contentType,
		"size_bytes":        len(body),
		"checksum_sha256":   sha256Hex(body),
	}).Error; err != nil {
		return "", fmt.Errorf("create attachment content: %w", err)
	}
	return contentID, nil
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
