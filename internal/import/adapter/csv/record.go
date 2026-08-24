// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const Format = "csv"

const externalIDColumn = "id"

func init() {
	adapter.RegisterPlanBuilder(Format, Builder{})
}

// Builder builds record plans from CSV sources.
type Builder struct{}

// Build implements adapter.PlanBuilder.
func (Builder) Build(ctx context.Context, spec importpkg.Spec) (plan.Plan, error) {
	_ = ctx
	if spec.Profile != importpkg.ProfileRecord {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "csv adapter requires record profile")
	}
	model := strings.TrimSpace(spec.Model)
	if model == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeModelNotFound, "model is required for record CSV import")
	}
	raw, err := readSourceBytes(ctx, spec)
	if err != nil {
		return plan.Plan{}, err
	}
	p, err := BuildRecordPlan(model, raw, spec.Options.ColumnMapping)
	if err != nil {
		return plan.Plan{}, err
	}
	return injectCompanyID(p, spec.Options.CompanyID), nil
}

// BuildRecordPlan parses CSV bytes into record units for one target model.
func BuildRecordPlan(model string, raw []byte, columnMapping map[string]string) (plan.Plan, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeModelNotFound, "model is required")
	}
	table, err := ReadTable(raw)
	if err != nil {
		return plan.Plan{}, err
	}
	fieldByHeader, err := mapHeaders(table.Headers, columnMapping)
	if err != nil {
		return plan.Plan{}, err
	}

	units := make([]plan.Unit, 0, len(table.Rows))
	for i, row := range table.Rows {
		values := make(map[string]string)
		externalID := ""
		for colIdx, header := range table.Headers {
			fieldPath, ok := fieldByHeader[header]
			if !ok {
				continue
			}
			cell := strings.TrimSpace(row[colIdx])
			if fieldPath == externalIDColumn {
				externalID = cell
				continue
			}
			if cell == "" {
				continue
			}
			values[fieldPath] = cell
		}
		rowNumber := i + 2
		if i < len(table.RowNumbers) {
			rowNumber = table.RowNumbers[i]
		}
		units = append(units, recordplan.Unit{
			Index:      i + 1,
			RowNumber:  rowNumber,
			Model:      model,
			ExternalID: externalID,
			Values:     values,
		})
	}
	return plan.Plan{Units: units}, nil
}

func mapHeaders(headers []string, columnMapping map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(headers))
	headerByField := make(map[string]string, len(headers))
	for _, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "CSV header must not be empty")
		}
		if _, dup := out[header]; dup {
			return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("duplicate CSV header %q", header))
		}
		fieldPath := header
		if columnMapping != nil {
			if mapped, ok := columnMapping[header]; ok {
				mapped = strings.TrimSpace(mapped)
				if mapped == "" {
					continue
				}
				fieldPath = mapped
			}
		}
		if previousHeader, exists := headerByField[fieldPath]; exists {
			return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf(
				"CSV headers %q and %q map to the same field %q",
				previousHeader, header, fieldPath))
		}
		headerByField[fieldPath] = header
		out[header] = fieldPath
	}
	return out, nil
}
