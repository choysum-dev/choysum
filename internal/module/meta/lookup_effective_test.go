// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestLookupEffectiveModel_FindsLiveRow(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	ts := time.Now().UTC()
	eff := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "eff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Path:        "/eff",
		Application: "partner",
	}
	if err := db.Create(eff).Error; err != nil {
		t.Fatalf("create eff: %v", err)
	}

	got, err := LookupEffectiveModel(db, "partner", "Partner")
	if err != nil {
		t.Fatalf("LookupEffectiveModel: %v", err)
	}
	if got.Id.String != "eff" {
		t.Fatalf("expected effective row, got %#v", got)
	}

	if _, err := LookupEffectiveModel(db, "partner", "Missing"); !IsEffectiveModelNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := LookupEffectiveModel(nil, "a", "b"); err == nil {
		t.Fatal("expected nil db error")
	}
	if _, err := LookupEffectiveModel(db, "", "x"); err == nil {
		t.Fatal("expected empty key error")
	}
}

func TestLookupEffectiveModel_FindErrorAndPickBranches(t *testing.T) {
	t.Run("find_error_closed_db", func(t *testing.T) {
		db := openDualStoreTestDB(t)
		if err := ensureDualStoreTables(db); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("db.DB: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
		if _, err := LookupEffectiveModel(db, "a", "B"); err == nil || !strings.Contains(err.Error(), "lookup effective") {
			t.Fatalf("expected find error, got %v", err)
		}
	})

	t.Run("single_row_and_tie_breaks", func(t *testing.T) {
		ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		assertPick := func(want string, rows ...pkgmeta.Model) {
			t.Helper()
			if got := pickEffectiveAmong(rows); got.Id.String != want {
				t.Fatalf("forward want %s, got %#v", want, got)
			}
			rev := make([]pkgmeta.Model, len(rows))
			for i := range rows {
				rev[len(rows)-1-i] = rows[i]
			}
			if got := pickEffectiveAmong(rev); got.Id.String != want {
				t.Fatalf("reversed want %s, got %#v", want, got)
			}
		}

		only := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "only", Valid: true}, UpdatedAt: ts},
			Name:      "X", Application: "a",
			ModuleId: sql.NullString{String: "mod", Valid: true},
		}
		assertPick("only", only)

		older := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "a-id", Valid: true}, UpdatedAt: ts},
			ModuleId:  sql.NullString{String: "m1", Valid: true},
		}
		newer := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "b-id", Valid: true}, UpdatedAt: ts.Add(time.Hour)},
			ModuleId:  sql.NullString{String: "m2", Valid: true},
		}
		assertPick("b-id", older, newer)

		low := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "aaa", Valid: true}, UpdatedAt: ts},
		}
		high := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "zzz", Valid: true}, UpdatedAt: ts},
		}
		assertPick("zzz", low, high)

		// Whitespace ModuleId counts as empty; prefer it over shell.
		shell := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "shell", Valid: true}, UpdatedAt: ts.Add(time.Hour)},
			ModuleId:  sql.NullString{String: "mod", Valid: true},
		}
		wsEmpty := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "ws", Valid: true}, UpdatedAt: ts},
			ModuleId:  sql.NullString{String: "  ", Valid: true},
		}
		assertPick("ws", shell, wsEmpty)

		// Three-way fold must match all-at-once pick (map-iteration order independence).
		mid := pkgmeta.Model{
			BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "mid", Valid: true}, UpdatedAt: ts.Add(30 * time.Minute)},
			ModuleId:  sql.NullString{String: "m3", Valid: true},
		}
		all := []pkgmeta.Model{older, mid, newer}
		want := pickEffectiveAmong(all).Id.String
		fold := func(rows []pkgmeta.Model) string {
			best := rows[0]
			for i := 1; i < len(rows); i++ {
				best = pickEffectiveAmong([]pkgmeta.Model{best, rows[i]})
			}
			return best.Id.String
		}
		if got := fold(all); got != want {
			t.Fatalf("forward fold %s != all-at-once %s", got, want)
		}
		rev := []pkgmeta.Model{newer, mid, older}
		if got := fold(rev); got != want {
			t.Fatalf("reverse fold %s != all-at-once %s", got, want)
		}
	})
}
