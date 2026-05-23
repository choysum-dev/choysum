// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package payload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	documentdb "github.com/choysum-dev/choysum/internal/document/storage/db"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

// PutRequest describes an upload payload write request.
type PutRequest struct {
	UploadID           string
	PayloadWriteTicket string
	ContentType        string
	Body               []byte
	ChecksumSHA256     string
}

// PutReceipt reports the stored payload metadata after a successful put.
type PutReceipt struct {
	PayloadID      string
	SizeBytes      int64
	ChecksumSHA256 string
	ContentType    string
}

// OpenRequest describes a payload open request for a document binding.
type OpenRequest struct {
	BindingID         string
	PayloadReadTicket string
}

// OpenResult contains the payload body returned by an open request.
type OpenResult struct {
	Body      io.ReadCloser
	SizeBytes int64
}

// DeleteRequest describes a payload delete request for a document binding.
type DeleteRequest struct {
	BindingID         string
	PayloadReadTicket string
}

// PromoteRequest describes a payload promotion request.
type PromoteRequest struct {
	PayloadID string
}

// PromoteReceipt reports the payload promoted into durable ownership.
type PromoteReceipt struct {
	PayloadID string
}

// Adapter defines payload lifecycle operations for document attachments.
type Adapter interface {
	Put(ctx context.Context, req PutRequest) (PutReceipt, error)
	Open(ctx context.Context, req OpenRequest) (OpenResult, error)
	Delete(ctx context.Context, req DeleteRequest) error
	Promote(ctx context.Context, req PromoteRequest) (PromoteReceipt, error)
}

// Options configures adapter dependencies.
type Options struct {
	Repository    pkgstorage.StoredContentRepository
	DriverFactory pkgstorage.StoredContentDriverFactory
}

type adapter struct {
	runtimeScope  scope.Scope
	repository    pkgstorage.StoredContentRepository
	driverFactory pkgstorage.StoredContentDriverFactory
}

const documentStoredContentTable = pkgstorage.DocumentStoredContentTable

// NewAdapter builds a payload adapter using the provided runtime scope and dependencies.
func NewAdapter(runtimeScope scope.Scope, opts Options) Adapter {
	return newAdapter(runtimeScope, opts)
}

// DeleteStoredContent deletes stored content by identifier using a short-lived adapter.
func DeleteStoredContent(ctx context.Context, runtimeScope scope.Scope, storedContentID string, opts Options) error {
	return newAdapter(runtimeScope, opts).deleteStoredContent(ctx, storedContentID)
}

func newAdapter(runtimeScope scope.Scope, opts Options) *adapter {
	driverFactory := opts.DriverFactory
	if driverFactory == nil {
		driverFactory = pkgstorage.NewFactory()
	}

	return &adapter{
		runtimeScope:  runtimeScope,
		repository:    opts.Repository,
		driverFactory: driverFactory,
	}
}

func (a *adapter) repositoryForContext(ctx context.Context) pkgstorage.StoredContentRepository {
	if a == nil {
		return nil
	}
	if a.repository != nil {
		return a.repository
	}
	db := storedContentDB(a.runtimeScope, ctx)
	if db == nil {
		return nil
	}
	return documentdb.NewStoredContentRepository(documentdb.RepositoryDeps{DB: db})
}

func storedContentDB(runtimeScope scope.Scope, ctx context.Context) *gorm.DB {
	db, ok := scope.DBForScope(ctx, runtimeScope)
	if !ok {
		return nil
	}
	return db
}

func (a *adapter) newDriver(provider string, att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
	if a.driverFactory == nil {
		return nil, fmt.Errorf("stored content driver factory is unavailable")
	}
	return a.driverFactory.NewDriver(provider, att)
}

func (a *adapter) Put(ctx context.Context, req PutRequest) (PutReceipt, error) {
	ticket, err := parsePayloadWriteTicket(req.PayloadWriteTicket)
	if err != nil {
		return PutReceipt{}, err
	}

	uploadID := strings.TrimSpace(req.UploadID)
	if uploadID == "" {
		return PutReceipt{}, InvalidArgument("upload id is required")
	}
	if ticket.UploadID != uploadID {
		return PutReceipt{}, FailedPrecondition("payload write ticket upload id mismatch")
	}

	provider, att := resolveAttachmentProvider(a.runtimeScope)
	return a.putStoredContent(ctx, req, ticket, provider, att)
}

func (a *adapter) Open(ctx context.Context, req OpenRequest) (OpenResult, error) {
	ticket, err := parseAndValidateReadTicket(req.BindingID, req.PayloadReadTicket)
	if err != nil {
		return OpenResult{}, err
	}
	record, err := a.loadActiveStoredContentRecord(ctx, ticket.StoredContentID)
	if err != nil {
		return OpenResult{}, err
	}

	return a.openStoredContent(ctx, record)
}

func (a *adapter) Delete(ctx context.Context, req DeleteRequest) error {
	ticket, err := parseAndValidateReadTicket(req.BindingID, req.PayloadReadTicket)
	if err != nil {
		return err
	}
	return a.deleteStoredContent(ctx, ticket.StoredContentID)
}

func (a *adapter) Promote(ctx context.Context, req PromoteRequest) (PromoteReceipt, error) {
	_ = ctx

	payloadID := strings.TrimSpace(req.PayloadID)
	if payloadID == "" {
		return PromoteReceipt{}, InvalidArgument("payload id is required")
	}

	normalized := strings.ToLower(payloadID)
	if strings.HasPrefix(normalized, "inline_base64:") || strings.HasPrefix(normalized, "data:") {
		return PromoteReceipt{}, InvalidArgument("payload id must be an opaque handle")
	}

	if strings.HasPrefix(normalized, "sc:") {
		if strings.TrimSpace(payloadID[3:]) == "" {
			return PromoteReceipt{}, InvalidArgument("stored content payload id is invalid")
		}
		return PromoteReceipt{PayloadID: payloadID}, nil
	}

	return PromoteReceipt{}, InvalidArgument("payload id format is unsupported")
}

func parseAndValidateReadTicket(bindingID string, rawTicket string) (payloadReadTicket, error) {
	ticket, err := parsePayloadReadTicket(rawTicket)
	if err != nil {
		return payloadReadTicket{}, err
	}

	normalizedBindingID := strings.TrimSpace(bindingID)
	if normalizedBindingID == "" {
		return payloadReadTicket{}, InvalidArgument("attachment binding id is required")
	}
	if ticket.AttachmentBindingID != "" && ticket.AttachmentBindingID != normalizedBindingID {
		return payloadReadTicket{}, FailedPrecondition("payload read ticket binding id mismatch")
	}

	return ticket, nil
}

func (a *adapter) loadStoredContentRecord(ctx context.Context, storedContentID string) (pkgstorage.StoredContentRecord, error) {
	repository := a.repositoryForContext(ctx)
	if repository == nil {
		return pkgstorage.StoredContentRecord{}, Internal("document database session is required")
	}
	if strings.TrimSpace(storedContentID) == "" {
		return pkgstorage.StoredContentRecord{}, InvalidArgument("stored content id is required")
	}

	record, err := repository.GetByID(ctx, storedContentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgstorage.StoredContentRecord{}, NotFound("stored content payload not found")
		}
		return pkgstorage.StoredContentRecord{}, InternalWrap("query stored content payload", err)
	}
	return record, nil
}

func (a *adapter) loadActiveStoredContentRecord(ctx context.Context, storedContentID string) (pkgstorage.StoredContentRecord, error) {
	record, err := a.loadStoredContentRecord(ctx, storedContentID)
	if err != nil {
		return pkgstorage.StoredContentRecord{}, err
	}
	if strings.TrimSpace(record.Status) != "active" {
		return pkgstorage.StoredContentRecord{}, NotFound("active stored content not found")
	}
	return record, nil
}

func (a *adapter) deleteStoredContent(ctx context.Context, storedContentID string) error {
	record, err := a.loadStoredContentRecord(ctx, storedContentID)
	if err != nil {
		return err
	}
	return a.deleteStoredContentRecord(ctx, record)
}

func (a *adapter) deleteStoredContentRecord(ctx context.Context, record pkgstorage.StoredContentRecord) error {
	repository := a.repositoryForContext(ctx)
	if repository == nil {
		return Internal("document database session is required")
	}

	provider := normalizeStoredContentProvider(record.Provider)
	driver, err := a.newDriver(provider, attachmentConfigFromRuntimeScope(a.runtimeScope))
	if err != nil {
		return InternalWrap(fmt.Sprintf("create %s stored content driver", provider), err)
	}
	if err := driver.Delete(ctx, record); err != nil {
		return InternalWrap("delete stored content payload", err)
	}

	now := time.Now().UTC()
	rowsAffected, err := repository.MarkDeleted(ctx, record.ID, now)
	if err != nil {
		return InternalWrap("mark stored content payload deleted", err)
	}
	if rowsAffected > 0 {
		return nil
	}

	exists, err := repository.Exists(ctx, record.ID)
	if err != nil {
		return InternalWrap("check stored content payload existence", err)
	}
	if !exists {
		return NotFound("stored content payload not found")
	}

	return nil
}

func (a *adapter) putStoredContent(ctx context.Context, req PutRequest, ticket payloadWriteTicket, provider string, att *config.AttachmentConfig) (PutReceipt, error) {
	repository := a.repositoryForContext(ctx)
	if repository == nil {
		return PutReceipt{}, Internal("document database session is required")
	}

	companyID, err := companyIDFromWriteTicket(ticket)
	if err != nil {
		return PutReceipt{}, err
	}

	provider = normalizeStoredContentProvider(provider)
	driver, err := a.newDriver(provider, att)
	if err != nil {
		return PutReceipt{}, InternalWrap(fmt.Sprintf("create %s stored content driver", provider), err)
	}

	mutation, err := driver.Put(ctx, pkgstorage.PutPayloadInput{
		UploadID:       req.UploadID,
		CompanyID:      companyID,
		ContentType:    req.ContentType,
		ChecksumSHA256: req.ChecksumSHA256,
		Body:           req.Body,
	})
	if err != nil {
		return PutReceipt{}, InternalWrap("put stored content payload", err)
	}

	storedContentID := xid.New().String()
	now := time.Now().UTC()
	err = repository.Create(ctx, pkgstorage.CreateStoredContentInput{
		ID:          storedContentID,
		Provider:    driver.Provider(),
		BlobData:    mutation.BlobData,
		LocatorJSON: mutation.LocatorJSON,
		Status:      "active",
		CompanyID:   companyID,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return PutReceipt{}, InternalWrap("create stored content payload", err)
	}

	return PutReceipt{
		PayloadID:      "sc:" + storedContentID,
		SizeBytes:      int64(len(req.Body)),
		ChecksumSHA256: strings.TrimSpace(req.ChecksumSHA256),
		ContentType:    strings.TrimSpace(req.ContentType),
	}, nil
}

func (a *adapter) openStoredContent(ctx context.Context, record pkgstorage.StoredContentRecord) (OpenResult, error) {
	provider := normalizeStoredContentProvider(record.Provider)
	driver, err := a.newDriver(provider, attachmentConfigFromRuntimeScope(a.runtimeScope))
	if err != nil {
		return OpenResult{}, InternalWrap(fmt.Sprintf("create %s stored content driver", provider), err)
	}

	payload, err := driver.Open(ctx, record)
	if err != nil {
		return OpenResult{}, InternalWrap("open stored content payload", err)
	}

	return OpenResult{
		Body:      io.NopCloser(bytes.NewReader(payload)),
		SizeBytes: int64(len(payload)),
	}, nil
}

func resolveAttachmentProvider(runtimeScope scope.Scope) (string, *config.AttachmentConfig) {
	att := attachmentConfigFromRuntimeScope(runtimeScope)
	if att == nil {
		return "db", nil
	}

	return normalizeStoredContentProvider(att.Backend), att
}

func normalizeStoredContentProvider(provider string) string {
	name := strings.ToLower(strings.TrimSpace(provider))
	if name == "" {
		return "db"
	}
	return name
}

func companyIDFromWriteTicket(ticket payloadWriteTicket) (string, error) {
	companyID := strings.TrimSpace(ticket.CompanyID)
	if companyID == "" {
		companyID = strings.TrimSpace(ticket.ActiveCompanyID)
	}
	if companyID == "" {
		return "", FailedPrecondition("payload write ticket missing companyId")
	}
	return companyID, nil
}
