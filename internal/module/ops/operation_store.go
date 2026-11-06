// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Overridable for tests that force encode/decode failures.
var (
	jsonMarshal   = json.Marshal
	jsonUnmarshal = json.Unmarshal
)

func (o *Ops) session(ctx context.Context) (*scope.Session, error) {
	if o == nil || o.Scope == nil {
		return nil, errors.New("ops: scope is unavailable")
	}
	sess, ok := scope.SessionForScope(ctx, o.Scope)
	if !ok || sess == nil || sess.DB == nil {
		return nil, errors.New("ops: session is unavailable")
	}
	return sess, nil
}

func (o *Ops) loadOperation(ctx context.Context, id string) (*modmeta.ModuleOperation, error) {
	sess, err := o.session(ctx)
	if err != nil {
		return nil, err
	}
	var row modmeta.ModuleOperation
	if err := sess.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("ops: operation %q not found", id)
		}
		return nil, err
	}
	return &row, nil
}

func (o *Ops) saveOperation(ctx context.Context, row *modmeta.ModuleOperation) error {
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).Save(row).Error
}

func (o *Ops) createOperation(ctx context.Context, row *modmeta.ModuleOperation) error {
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).Create(row).Error
}

func marshalPlanDocument(doc PlanDocument) (datatypes.JSON, error) {
	raw, err := jsonMarshal(doc)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func unmarshalPlanDocument(raw datatypes.JSON) (PlanDocument, error) {
	var doc PlanDocument
	if len(raw) == 0 {
		return doc, nil
	}
	if err := jsonUnmarshal(raw, &doc); err != nil {
		return doc, err
	}
	return doc, nil
}
