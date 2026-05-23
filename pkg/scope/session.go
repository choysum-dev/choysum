// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"context"

	"gorm.io/gorm"
)

type Session struct {
	*gorm.DB
}

func (s *Session) Savepoint(name string) error {
	if s == nil || s.DB == nil {
		return ErrSessionUnavailable
	}
	return s.SavePoint(name).Error
}

func (s *Session) RollbackToSavepoint(name string) error {
	if s == nil || s.DB == nil {
		return ErrSessionUnavailable
	}
	return s.RollbackTo(name).Error
}

func (s *Session) ReleaseSavepoint(name string) error {
	if s == nil || s.DB == nil {
		return ErrSessionUnavailable
	}

	switch s.Dialector.Name() {
	case "mysql", "postgres":
		return s.Exec("RELEASE SAVEPOINT " + name).Error
	case "sqlite":
		return s.Exec("RELEASE " + name).Error
	default:
		return nil
	}
}

// SessionForScope resolves the effective DB session for ctx and scope.
func SessionForScope(ctx context.Context, scope Scope) (*Session, bool) {
	if sess, ok := SessionFromContext(ctx); ok {
		return sess, true
	}
	if scope == nil {
		return nil, false
	}
	if ctx != nil {
		if scoped := scope.WithContext(ctx); scoped != nil {
			if sess := scoped.Session(); sess != nil {
				return sess, true
			}
		}
	}
	sess := scope.Session()
	return sess, sess != nil
}

// DBForScope resolves the effective DB handle for ctx and scope.
func DBForScope(ctx context.Context, scope Scope) (*gorm.DB, bool) {
	sess, ok := SessionForScope(ctx, scope)
	if !ok || sess == nil || sess.DB == nil {
		return nil, false
	}
	if ctx != nil {
		return sess.DB.WithContext(ctx), true
	}
	return sess.DB, true
}
