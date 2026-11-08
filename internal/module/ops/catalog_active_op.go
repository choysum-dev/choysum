// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"database/sql"
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
)

// setCatalogActiveOp stamps ActiveOperationId on catalog rows for the given modules.
// Skips empty names. Does not clear other modules' active ids.
func (o *Ops) setCatalogActiveOp(ctx context.Context, moduleNames []string, opID string) error {
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil
	}
	names := uniqueTrimmedNames(moduleNames)
	if len(names) == 0 {
		return nil
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Where("module_name IN ?", names).
		Update("active_operation_id", sql.NullString{String: opID, Valid: true}).Error
}

// clearCatalogActiveOp clears ActiveOperationId wherever it still points at opID.
func (o *Ops) clearCatalogActiveOp(ctx context.Context, opID string) error {
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Where("active_operation_id = ?", opID).
		Update("active_operation_id", sql.NullString{}).Error
}

func uniqueTrimmedNames(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, name := range in {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func operationModuleNames(row *modmeta.ModuleOperation, doc PlanDocument) []string {
	names := make([]string, 0, len(doc.ModuleOrder)+1)
	if row != nil {
		if root := strings.TrimSpace(row.ModuleName); root != "" {
			names = append(names, root)
		}
	}
	for _, e := range doc.ModuleOrder {
		names = append(names, e.ModuleName)
	}
	return uniqueTrimmedNames(names)
}
