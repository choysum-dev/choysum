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
	"io"
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
	messagesToCSVBytes     = defaultMessagesToCSVBytes
	storeArtifact          = defaultStoreArtifact
	newStoredContentDriver = documentdb.NewStoredContentDriver
	putStoredContent       = func(driver pkgstorage.StoredContentDriver, ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
		return driver.Put(ctx, input)
	}
	newStoredContentRepository = documentdb.NewStoredContentRepository
	ioWriterForErrorCSV        = func(w io.Writer) io.Writer { return w }
	writeCSVRecord             = func(w *csv.Writer, record []string) error { return w.Write(record) }
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
	buf := &bytes.Buffer{}
	w := csv.NewWriter(ioWriterForErrorCSV(buf))
	if err := writeCSVRecord(w, []string{"type", "row", "field", "code", "text", "record_ref"}); err != nil {
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
		if err := writeCSVRecord(w, row); err != nil {
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

	driver, err := newStoredContentDriver(nil)
	if err != nil {
		return "", fmt.Errorf("create stored content driver: %w", err)
	}
	mutation, err := putStoredContent(driver, ctx, pkgstorage.PutPayloadInput{
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
	repo := newStoredContentRepository(documentdb.RepositoryDeps{DB: session.DB})
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
