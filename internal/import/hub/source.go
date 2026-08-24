// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"strings"

	documentgateway "github.com/choysum-dev/choysum/internal/document/gateway"
	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// SourceReader loads import source bytes for a document reference.
type SourceReader interface {
	Read(ctx context.Context, sourceRef string) ([]byte, error)
}

// DocumentSourceReader reads attachment binding or content refs via the document gateway.
type DocumentSourceReader struct {
	RuntimeScope scope.Scope
}

// Read implements SourceReader.
func (r DocumentSourceReader) Read(ctx context.Context, sourceRef string) ([]byte, error) {
	if r.RuntimeScope == nil {
		return nil, fmt.Errorf("runtime scope is required")
	}
	identity := auth.IdentityFromContext(ctx)
	if identity == nil || !identity.IsValid() {
		return nil, fmt.Errorf("authentication is required")
	}
	return documentgateway.ReadSourceRefBytes(ctx, r.RuntimeScope, sourceRef, identity)
}

// StubSourceReader returns fixed bytes for tests.
type StubSourceReader map[string][]byte

// Read implements SourceReader.
func (s StubSourceReader) Read(_ context.Context, sourceRef string) ([]byte, error) {
	raw, ok := s[strings.TrimSpace(sourceRef)]
	if !ok {
		return nil, csvReadError("source ref not found")
	}
	return raw, nil
}

func csvReadError(text string) error {
	return &readError{text: text}
}

type readError struct {
	text string
}

func (e *readError) Error() string {
	return e.text
}

// ContextWithSourceReader attaches a reader to the context for csv adapter loading.
func ContextWithSourceReader(ctx context.Context, reader SourceReader) context.Context {
	if reader == nil {
		return ctx
	}
	return csv.ContextWithSourceBytes(ctx, func(ctx context.Context, documentRef string) ([]byte, error) {
		return reader.Read(ctx, documentRef)
	})
}
