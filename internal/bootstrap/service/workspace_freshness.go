// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

func (c *coordinator) defaultCheckWorkspaceFreshness(ctx context.Context) error {
	if c.runtimeScope == nil {
		return newBootstrapError(bootstrapErrCodeGateError, "scope is not available", nil)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	txRoot := c.runtimeScope.WithContext(ctx)
	return txRoot.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		session := txScope.Session()
		if session == nil || session.DB == nil {
			return newBootstrapError(bootstrapErrCodeGateError, "database session is not available", nil)
		}

		if session.Migrator().HasTable((&meta.Module{}).TableName()) {
			var moduleCount int64
			if err := session.Model(&meta.Module{}).Limit(1).Count(&moduleCount).Error; err != nil {
				return newBootstrapError(bootstrapErrCodeGateError, "failed to inspect existing setup data", err)
			}
			if moduleCount > 0 {
				return newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "initial setup has already been completed: existing module metadata was found", nil)
			}
		}

		if session.Migrator().HasTable((&meta.Model{}).TableName()) {
			var model meta.Model
			err := session.Select("id").Where("application = ? AND name = ?", "auth", "User").Take(&model).Error
			if err == nil {
				return newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "initial setup has already been completed: administrator model metadata already exists", nil)
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return newBootstrapError(bootstrapErrCodeGateError, "failed to inspect existing administrator account data", err)
			}
		}

		if session.Migrator().HasTable((&modmeta.ModelData{}).TableName()) {
			var modelData modmeta.ModelData
			err := session.Select("id").Where("module = ? AND name = ?", "auth", "user_admin").Take(&modelData).Error
			if err == nil {
				return newBootstrapError(bootstrapErrCodeWorkspaceNotFresh, "initial setup has already been completed: administrator setup data already exists", nil)
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return newBootstrapError(bootstrapErrCodeGateError, "failed to inspect existing administrator setup data", err)
			}
		}

		return nil
	})
}
