// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestTransactorRequiredMarksJoinedFailureForRollback(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	if err := runtimeScope.Session().AutoMigrate(&runRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	joinedErr := errors.New("joined failure")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		if err := txScope.Session().Create(&runRecord{Name: "outer"}).Error; err != nil {
			return err
		}

		innerErr := txScope.Transactor().Required(tx.Context(), func(joinedScope scope.Scope, joinedTx scope.Transaction) error {
			if joinedScope.Session() != txScope.Session() {
				t.Fatal("expected Required to reuse the active session")
			}
			if joinedTx.Session() != tx.Session() {
				t.Fatal("expected joined transaction to reuse the active session")
			}
			if err := joinedScope.Session().Create(&runRecord{Name: "joined"}).Error; err != nil {
				return err
			}
			return joinedErr
		})
		if !errors.Is(innerErr, joinedErr) {
			t.Fatalf("joined Required error = %v, want %v", innerErr, joinedErr)
		}

		return nil
	})
	if !errors.Is(err, joinedErr) {
		t.Fatalf("top-level Required error = %v, want %v", err, joinedErr)
	}

	var count int64
	if err := runtimeScope.Session().Model(&runRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("Count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("row count after joined failure = %d, want 0", count)
	}
}

func TestTransactorRequiresNewCommitsIndependently(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	if err := runtimeScope.Session().AutoMigrate(&runRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	outerErr := errors.New("outer failure")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		innerErr := txScope.Transactor().RequiresNew(tx.Context(), func(newScope scope.Scope, newTx scope.Transaction) error {
			if newScope.Session() == txScope.Session() {
				t.Fatal("expected RequiresNew to start a fresh session")
			}
			if newTx.Session() == tx.Session() {
				t.Fatal("expected RequiresNew to use a distinct transaction")
			}
			return newScope.Session().Create(&runRecord{Name: "requires-new"}).Error
		})
		if innerErr != nil {
			return innerErr
		}
		if err := txScope.Session().Create(&runRecord{Name: "outer"}).Error; err != nil {
			return err
		}
		return outerErr
	})
	if !errors.Is(err, outerErr) {
		t.Fatalf("Required error = %v, want %v", err, outerErr)
	}

	var requiresNewCount int64
	if err := runtimeScope.Session().Model(&runRecord{}).Where("name = ?", "requires-new").Count(&requiresNewCount).Error; err != nil {
		t.Fatalf("Count requires-new rows: %v", err)
	}
	if requiresNewCount != 1 {
		t.Fatalf("requires-new row count = %d, want 1", requiresNewCount)
	}

	var outerCount int64
	if err := runtimeScope.Session().Model(&runRecord{}).Where("name = ?", "outer").Count(&outerCount).Error; err != nil {
		t.Fatalf("Count outer rows: %v", err)
	}
	if outerCount != 0 {
		t.Fatalf("outer row count = %d, want 0", outerCount)
	}
}

func TestTransactorNestedRollsBackToSavepoint(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	if err := runtimeScope.Session().AutoMigrate(&runRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	nestedErr := errors.New("nested failure")
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		if err := txScope.Session().Create(&runRecord{Name: "outer-before"}).Error; err != nil {
			return err
		}

		innerErr := txScope.Transactor().Nested(tx.Context(), func(nestedScope scope.Scope, nestedTx scope.Transaction) error {
			if nestedScope.Session() != txScope.Session() {
				t.Fatal("expected Nested to reuse the parent session")
			}
			if nestedTx.Session() != tx.Session() {
				t.Fatal("expected Nested transaction to reuse the parent session")
			}
			if err := nestedScope.Session().Create(&runRecord{Name: "nested"}).Error; err != nil {
				return err
			}
			return nestedErr
		})
		if !errors.Is(innerErr, nestedErr) {
			t.Fatalf("Nested error = %v, want %v", innerErr, nestedErr)
		}

		return txScope.Session().Create(&runRecord{Name: "outer-after"}).Error
	})
	if err != nil {
		t.Fatalf("Required with nested rollback: %v", err)
	}

	var outerCount int64
	if err := runtimeScope.Session().Model(&runRecord{}).Where("name IN ?", []string{"outer-before", "outer-after"}).Count(&outerCount).Error; err != nil {
		t.Fatalf("Count outer rows: %v", err)
	}
	if outerCount != 2 {
		t.Fatalf("outer row count = %d, want 2", outerCount)
	}

	var nestedCount int64
	if err := runtimeScope.Session().Model(&runRecord{}).Where("name = ?", "nested").Count(&nestedCount).Error; err != nil {
		t.Fatalf("Count nested rows: %v", err)
	}
	if nestedCount != 0 {
		t.Fatalf("nested row count = %d, want 0", nestedCount)
	}
}

func TestTransactorRequiredUsesFreshTransactionOnRootContext(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		if tx == nil {
			t.Fatal("expected transaction")
		}
		if tx.Session() == nil {
			t.Fatal("expected transaction session")
		}
		if txScope.Session() != tx.Session() {
			t.Fatal("expected transaction-bound scope session")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Required with root context: %v", err)
	}
}
