// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"context"
	"os"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

type sourceBytesContextKey struct{}

// SourceBytesLoader reads import source bytes for a document reference.
type SourceBytesLoader func(ctx context.Context, documentRef string) ([]byte, error)

// ContextWithSourceBytes attaches a loader used when Spec.Source.DocumentRef is set.
func ContextWithSourceBytes(ctx context.Context, loader SourceBytesLoader) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if loader == nil {
		return ctx
	}
	return context.WithValue(ctx, sourceBytesContextKey{}, loader)
}

// HasSourceBytesLoader reports whether ctx already carries a document source loader.
func HasSourceBytesLoader(ctx context.Context) bool {
	_, ok := sourceBytesFromContext(ctx)
	return ok
}

// SourceBytesLoaderFromContext returns the attached document source loader, if any.
func SourceBytesLoaderFromContext(ctx context.Context) (SourceBytesLoader, bool) {
	return sourceBytesFromContext(ctx)
}

func sourceBytesFromContext(ctx context.Context) (SourceBytesLoader, bool) {
	if ctx == nil {
		return nil, false
	}
	loader, ok := ctx.Value(sourceBytesContextKey{}).(SourceBytesLoader)
	return loader, ok && loader != nil
}

func readSourceBytes(ctx context.Context, spec importpkg.Spec) ([]byte, error) {
	path := strings.TrimSpace(spec.Source.Path)
	if path != "" {
		return os.ReadFile(path)
	}
	docRef := strings.TrimSpace(spec.Source.DocumentRef)
	if docRef == "" {
		return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "source path or document_ref is required for record CSV import")
	}
	loader, ok := sourceBytesFromContext(ctx)
	if !ok {
		return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "document source loader is unavailable")
	}
	raw, err := loader(ctx, docRef)
	if err != nil {
		return nil, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "read document source", err)
	}
	return raw, nil
}

func injectCompanyID(p plan.Plan, companyID string) plan.Plan {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" || len(p.Units) == 0 {
		return p
	}
	for i, unit := range p.Units {
		u, ok := unit.(recordplan.Unit)
		if !ok {
			continue
		}
		if u.Values == nil {
			u.Values = map[string]string{}
		}
		u.Values["CompanyId"] = companyID
		p.Units[i] = u
	}
	return p
}
