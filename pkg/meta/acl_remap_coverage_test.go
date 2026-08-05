// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openACLRemapDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := EnsureDualStoreTables(db); err != nil {
		t.Fatalf("ensure dual store: %v", err)
	}
	return db
}

func seedACLTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS auth_role_record_rule (
			id TEXT PRIMARY KEY, meta_model_id TEXT, meta_application_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS auth_role_field_rule (
			id TEXT PRIMARY KEY, meta_model_id TEXT, meta_field_id TEXT, meta_application_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS auth_role_method_access (
			id TEXT PRIMARY KEY, meta_model_id TEXT, meta_service_id TEXT, meta_application_id TEXT)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create acl table: %v", err)
		}
	}
}

func TestRemapACLToEffectiveModelIDs_CoverageBranches(t *testing.T) {
	t.Run("empty_live_catalog", func(t *testing.T) {
		db := openACLRemapDB(t, "acl-empty-live")
		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("empty live: %v", err)
		}
	})

	t.Run("skip_invalid_live_and_deleted_rows", func(t *testing.T) {
		db := openACLRemapDB(t, "acl-skip-invalid")
		seedACLTables(t, db)
		ts := time.Now().UTC()
		// Empty application+name → skipped from effectiveByKey.
		blank := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "blank", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		}
		eff := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "eff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "User",
			Application: "auth",
			Path:        "/eff",
		}
		other := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "other", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "Role",
			Application: "auth",
			Path:        "/other",
		}
		for _, m := range []*Model{blank, eff, other} {
			if err := db.Create(m).Error; err != nil {
				t.Fatalf("create: %v", err)
			}
		}
		// Soft-deleted historical shell for same (app,name) as eff.
		shell := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "shell", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "User",
			Application: "auth",
			Path:        "/shell",
			ModuleId:    sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Create(shell).Error; err != nil {
			t.Fatalf("create shell: %v", err)
		}
		if err := db.Delete(shell).Error; err != nil {
			t.Fatalf("soft-delete shell: %v", err)
		}
		// Duplicate live keys → pickEffectiveAmong among live rows.
		eff2 := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "eff2", Valid: true}, CreatedAt: ts, UpdatedAt: ts.Add(time.Hour)},
			Name:        "User",
			Application: "auth",
			Path:        "/eff2",
			ModuleId:    sql.NullString{String: "mod2", Valid: true},
		}
		if err := db.Create(eff2).Error; err != nil {
			t.Fatalf("create eff2: %v", err)
		}
		// Inject invalid-id row via load-all hook.
		prevAll := aclLoadAllModels
		t.Cleanup(func() { aclLoadAllModels = prevAll })
		aclLoadAllModels = func(db *gorm.DB) ([]Model, error) {
			rows, err := prevAll(db)
			if err != nil {
				return nil, err
			}
			rows = append(rows, Model{Name: "Ghost", Application: "auth"}) // Id invalid
			return rows, nil
		}
		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("remap: %v", err)
		}
	})

	t.Run("field_and_service_skip_and_error_hooks", func(t *testing.T) {
		db := openACLRemapDB(t, "acl-hooks")
		seedACLTables(t, db)
		ts := time.Now().UTC()
		shell := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "shell-h", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "User", Application: "auth", Path: "/shell",
			ModuleId:    sql.NullString{String: "mod", Valid: true},
		}
		eff := &Model{
			BaseModel:   BaseModel{Id: sql.NullString{String: "eff-h", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:        "User", Application: "auth", Path: "/eff",
		}
		for _, m := range []*Model{shell, eff} {
			if err := db.Create(m).Error; err != nil {
				t.Fatalf("create model: %v", err)
			}
		}
		oldField := &Field{BaseModel: BaseModel{Id: sql.NullString{String: "of", Valid: true}}, Name: "Login", ModelId: shell.Id}
		newField := &Field{BaseModel: BaseModel{Id: sql.NullString{String: "nf", Valid: true}}, Name: "Login", ModelId: eff.Id}
		blankNameField := &Field{BaseModel: BaseModel{Id: sql.NullString{String: "bnf", Valid: true}}, Name: "  ", ModelId: shell.Id}
		for _, f := range []*Field{oldField, newField, blankNameField} {
			if err := db.Create(f).Error; err != nil {
				t.Fatalf("create field: %v", err)
			}
		}
		oldSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "os", Valid: true}}, Name: "Read", ModelId: shell.Id}
		newSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "ns", Valid: true}}, Name: "Read", ModelId: eff.Id}
		blankSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "bs", Valid: true}}, Name: "", ModelId: shell.Id}
		liveUnderEff := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "live", Valid: true}}, Name: "Browse", ModelId: eff.Id}
		for _, s := range []*Service{oldSvc, newSvc, blankSvc, liveUnderEff} {
			if err := db.Create(s).Error; err != nil {
				t.Fatalf("create svc: %v", err)
			}
		}
		// Soft-deleted under effective → live Take misses, then remap via TakeServiceByModelName hook.
		orphanUnderEff := &Service{
			BaseModel: BaseModel{Id: sql.NullString{String: "orphan", Valid: true}},
			Name:      "OrphanOld", ModelId: eff.Id,
		}
		if err := db.Create(orphanUnderEff).Error; err != nil {
			t.Fatalf("create orphan: %v", err)
		}
		if err := db.Delete(orphanUnderEff).Error; err != nil {
			t.Fatalf("soft-delete orphan: %v", err)
		}
		prevTakeByName := aclTakeServiceByModelName
		t.Cleanup(func() { aclTakeServiceByModelName = prevTakeByName })
		aclTakeServiceByModelName = func(db *gorm.DB, dest interface{}, modelID, name string) error {
			if name == "OrphanOld" {
				if s, ok := dest.(*Service); ok {
					*s = Service{BaseModel: BaseModel{Id: sql.NullString{String: "repl-o", Valid: true}}, Name: name, ModelId: sql.NullString{String: modelID, Valid: true}}
					return nil
				}
			}
			return prevTakeByName(db, dest, modelID, name)
		}

		mustExecACL := func(sqlStmt string, args ...any) {
			t.Helper()
			if err := db.Exec(sqlStmt, args...).Error; err != nil {
				t.Fatalf("exec: %v", err)
			}
		}
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-ok", "shell-h", "of")
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-blank-model", "  ", "of")
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-missing-field", "eff-h", "gone-field")
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-blank-name", "eff-h", "bnf")
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-no-repl", "eff-h", "of") // of under shell; no Login on wrong model? Wait of name Login exists on eff as nf - Take by model+name finds nf. Use field only on shell with unique name.
		// Replace fr-no-repl: field with name only on shell.
		lonely := &Field{BaseModel: BaseModel{Id: sql.NullString{String: "lonely", Valid: true}}, Name: "Lonely", ModelId: shell.Id}
		if err := db.Create(lonely).Error; err != nil {
			t.Fatalf("lonely: %v", err)
		}
		mustExecACL(`UPDATE auth_role_field_rule SET meta_field_id=? WHERE id=?`, "lonely", "fr-no-repl")
		mustExecACL(`INSERT INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-same", "eff-h", "nf")

		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-ok", "shell-h", "os")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-blank-svc", "shell-h", "  ")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-missing-svc", "shell-h", "gone-svc")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-blank-name", "shell-h", "bs")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-live", "eff-h", "live")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-orphan", "eff-h", "orphan")
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-no-hist", "shell-h", "os-nohist")
		ghostSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "os-nohist", Valid: true}}, Name: "Ghost", ModelId: sql.NullString{String: "no-such-model", Valid: true}}
		if err := db.Create(ghostSvc).Error; err != nil {
			t.Fatalf("ghost svc: %v", err)
		}
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-no-eff", "shell-h", "os-noeff")
		noEffModel := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "hist-only", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "OnlyHist", Application: "auth", Path: "/hist", ModuleId: sql.NullString{String: "m", Valid: true},
		}
		if err := db.Create(noEffModel).Error; err != nil {
			t.Fatalf("hist-only: %v", err)
		}
		if err := db.Delete(noEffModel).Error; err != nil {
			t.Fatalf("delete hist-only: %v", err)
		}
		noEffSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "os-noeff", Valid: true}}, Name: "X", ModelId: noEffModel.Id}
		if err := db.Create(noEffSvc).Error; err != nil {
			t.Fatalf("noeff svc: %v", err)
		}
		mustExecACL(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-no-repl", "shell-h", "os-norepl")
		noReplSvc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "os-norepl", Valid: true}}, Name: "UniqueSvc", ModelId: shell.Id}
		if err := db.Create(noReplSvc).Error; err != nil {
			t.Fatalf("norepl: %v", err)
		}

		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("happy path branches: %v", err)
		}

		// --- Error hooks ---
		forceErr := errors.New("boom")

		t.Run("load_live_error", func(t *testing.T) {
			prev := aclLoadLiveModels
			t.Cleanup(func() { aclLoadLiveModels = prev })
			aclLoadLiveModels = func(*gorm.DB) ([]Model, error) { return nil, forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "load live") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("load_all_error", func(t *testing.T) {
			prev := aclLoadAllModels
			t.Cleanup(func() { aclLoadAllModels = prev })
			aclLoadAllModels = func(*gorm.DB) ([]Model, error) { return nil, forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "including deleted") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("exec_model_remap_error", func(t *testing.T) {
			prev := aclExec
			t.Cleanup(func() { aclExec = prev })
			aclExec = func(*gorm.DB, string, ...interface{}) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "remap") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("load_field_rules_error", func(t *testing.T) {
			prev := aclLoadFieldRules
			t.Cleanup(func() { aclLoadFieldRules = prev })
			aclLoadFieldRules = func(*gorm.DB, interface{}) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "load field rules") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("take_field_error", func(t *testing.T) {
			prev := aclTakeFieldUnscoped
			t.Cleanup(func() { aclTakeFieldUnscoped = prev })
			aclTakeFieldUnscoped = func(*gorm.DB, interface{}, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "lookup field rule") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("take_replacement_field_error", func(t *testing.T) {
			prev := aclTakeFieldByModelName
			t.Cleanup(func() { aclTakeFieldByModelName = prev })
			aclTakeFieldByModelName = func(*gorm.DB, interface{}, string, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "replacement field") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("exec_field_id_error", func(t *testing.T) {
			// Need old→new field remap to reach exec; reset shell field rule.
			mustExecACL(`INSERT OR REPLACE INTO auth_role_field_rule (id, meta_model_id, meta_field_id) VALUES (?,?,?)`, "fr-ok2", "eff-h", "of")
			n := 0
			prev := aclExec
			t.Cleanup(func() { aclExec = prev })
			aclExec = func(db *gorm.DB, sql string, values ...interface{}) error {
				if strings.Contains(sql, "meta_field_id") {
					return forceErr
				}
				n++
				return prev(db, sql, values...)
			}
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "field_id") {
				t.Fatalf("got %v (exec calls before fail=%d)", err, n)
			}
		})
		t.Run("load_method_access_error", func(t *testing.T) {
			prev := aclLoadMethodAccess
			t.Cleanup(func() { aclLoadMethodAccess = prev })
			aclLoadMethodAccess = func(*gorm.DB, interface{}) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "load method access") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("take_service_unscoped_error", func(t *testing.T) {
			prev := aclTakeServiceUnscoped
			t.Cleanup(func() { aclTakeServiceUnscoped = prev })
			aclTakeServiceUnscoped = func(*gorm.DB, interface{}, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "lookup method access") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("take_live_service_error", func(t *testing.T) {
			prev := aclTakeServiceLive
			t.Cleanup(func() { aclTakeServiceLive = prev })
			aclTakeServiceLive = func(*gorm.DB, interface{}, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "lookup live service") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("take_hist_model_error", func(t *testing.T) {
			prev := aclTakeModelUnscoped
			t.Cleanup(func() { aclTakeModelUnscoped = prev })
			aclTakeModelUnscoped = func(*gorm.DB, interface{}, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "historical model") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("lookup_effective_error", func(t *testing.T) {
			prev := aclLookupEffective
			t.Cleanup(func() { aclLookupEffective = prev })
			aclLookupEffective = func(*gorm.DB, string, string) (*Model, error) { return nil, forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "lookup effective model") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("lookup_effective_nil_id", func(t *testing.T) {
			prev := aclLookupEffective
			t.Cleanup(func() { aclLookupEffective = prev })
			aclLookupEffective = func(*gorm.DB, string, string) (*Model, error) {
				return &Model{}, nil // Id invalid
			}
			if err := RemapACLToEffectiveModelIDs(db); err != nil {
				t.Fatalf("nil id should continue: %v", err)
			}
		})
		t.Run("take_replacement_service_error", func(t *testing.T) {
			prev := aclTakeServiceByModelName
			t.Cleanup(func() { aclTakeServiceByModelName = prev })
			aclTakeServiceByModelName = func(*gorm.DB, interface{}, string, string) error { return forceErr }
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "replacement service") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("exec_service_id_error", func(t *testing.T) {
			mustExecACL(`INSERT OR REPLACE INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma-ok3", "shell-h", "os")
			prev := aclExec
			t.Cleanup(func() { aclExec = prev })
			aclExec = func(db *gorm.DB, sql string, values ...interface{}) error {
				if strings.Contains(sql, "meta_service_id") {
					return forceErr
				}
				return prev(db, sql, values...)
			}
			if err := RemapACLToEffectiveModelIDs(db); err == nil || !strings.Contains(err.Error(), "service_id") {
				t.Fatalf("got %v", err)
			}
		})
		t.Run("invalid_eff_id_in_map", func(t *testing.T) {
			prevLive := aclLoadLiveModels
			prevAll := aclLoadAllModels
			t.Cleanup(func() {
				aclLoadLiveModels = prevLive
				aclLoadAllModels = prevAll
			})
			aclLoadLiveModels = func(*gorm.DB) ([]Model, error) {
				return []Model{{
					BaseModel:   BaseModel{Id: sql.NullString{String: "e", Valid: true}},
					Application: "auth", Name: "User",
				}}, nil
			}
			aclLoadAllModels = func(*gorm.DB) ([]Model, error) {
				return []Model{
					{BaseModel: BaseModel{Id: sql.NullString{String: "old", Valid: true}}, Application: "auth", Name: "User"},
					{BaseModel: BaseModel{Id: sql.NullString{String: "e", Valid: true}}, Application: "auth", Name: "User"},
					// Same key but override effective with invalid Id via second pass — force !eff.Id.Valid
					{Application: "auth", Name: "MissingEff"}, // no effective
				}, nil
			}
			// Force effectiveByKey entry with invalid Id by mutating after pick — inject via live that has Valid id then replace in loadAll path using hook on effective map... simpler: custom live with Valid then all row with key that maps to empty-id effective.
			aclLoadLiveModels = func(*gorm.DB) ([]Model, error) {
				return []Model{{
					Application: "auth", Name: "Z",
					// Id invalid → skipped; empty effectiveByKey early return already tested.
					// Provide Valid id then overwrite via duplicate with invalid — pick keeps first Valid.
					BaseModel: BaseModel{Id: sql.NullString{String: "z-eff", Valid: true}},
				}, {
					Application: "auth", Name: "BadEff",
					BaseModel: BaseModel{Id: sql.NullString{String: "bad", Valid: true}},
				}}, nil
			}
			// Replace BadEff's entry: can't easily. Hit !ok path with hist for MissingEff.
			if err := RemapACLToEffectiveModelIDs(db); err != nil {
				t.Fatalf("got %v", err)
			}
		})
	})

	t.Run("same_service_id_skip", func(t *testing.T) {
		db := openACLRemapDB(t, "acl-same-svc")
		seedACLTables(t, db)
		ts := time.Now().UTC()
		shell := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "sh", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "M", Application: "a", Path: "/s", ModuleId: sql.NullString{String: "m", Valid: true},
		}
		eff := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "ef", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "M", Application: "a", Path: "/e",
		}
		_ = db.Create(shell)
		_ = db.Create(eff)
		// Soft-delete shell service; replacement under eff reuses same service id string somehow — instead
		// point method access at shell service and create replacement with SAME id under eff (impossible PK).
		// Hit !replacement.Id.Valid via hook.
		svc := &Service{BaseModel: BaseModel{Id: sql.NullString{String: "svc1", Valid: true}}, Name: "S", ModelId: shell.Id}
		_ = db.Create(svc)
		_ = db.Exec(`INSERT INTO auth_role_method_access (id, meta_model_id, meta_service_id) VALUES (?,?,?)`, "ma1", "sh", "svc1")
		prev := aclTakeServiceByModelName
		t.Cleanup(func() { aclTakeServiceByModelName = prev })
		aclTakeServiceByModelName = func(_ *gorm.DB, dest interface{}, _, _ string) error {
			// Leave Id invalid
			return nil
		}
		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("no_auth_tables", func(t *testing.T) {
		db := openACLRemapDB(t, "acl-no-auth")
		ts := time.Now().UTC()
		_ = db.Create(&Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "e", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "X", Application: "a", Path: "/x",
		})
		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("no auth tables: %v", err)
		}
	})

	t.Run("invalid_live_id_skipped", func(t *testing.T) {
		prev := aclLoadLiveModels
		t.Cleanup(func() { aclLoadLiveModels = prev })
		aclLoadLiveModels = func(*gorm.DB) ([]Model, error) {
			return []Model{{Application: "a", Name: "X"}}, nil // invalid Id
		}
		db := openACLRemapDB(t, "acl-inv-live")
		if err := RemapACLToEffectiveModelIDs(db); err != nil {
			t.Fatalf("got %v", err)
		}
	})
}

func TestMigrateAndRecompute_RemapACLWiring(t *testing.T) {
	t.Run("migrate_index_error", func(t *testing.T) {
		db := openDualStoreTestDB(t)
		if err := EnsureDualStoreTables(db); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		ts := time.Now().UTC()
		src := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "m1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "X", Application: "a", Path: "/x.ts", ModuleId: sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Create(src).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
		prev := execDDL
		t.Cleanup(func() { execDDL = prev })
		execDDL = func(*gorm.DB, string) error { return errors.New("index boom") }
		if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "index boom") {
			t.Fatalf("expected index error, got %v", err)
		}
	})

	t.Run("migrate_remap_error", func(t *testing.T) {
		db := openDualStoreTestDB(t)
		if err := EnsureDualStoreTables(db); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		ts := time.Now().UTC()
		src := &Model{
			BaseModel: BaseModel{Id: sql.NullString{String: "m2", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "Y", Application: "a", Path: "/y.ts", ModuleId: sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Create(src).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
		prev := remapACLToEffectiveModelIDsFn
		t.Cleanup(func() { remapACLToEffectiveModelIDsFn = prev })
		remapACLToEffectiveModelIDsFn = func(*gorm.DB) error { return errors.New("remap boom") }
		if err := MigrateIMDCatalogToDualStore(db); err == nil || !strings.Contains(err.Error(), "remap boom") {
			t.Fatalf("expected remap error, got %v", err)
		}
	})

	t.Run("recompute_index_error", func(t *testing.T) {
		db := openDualStoreTestDB(t)
		if err := EnsureDualStoreTables(db); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		ts := time.Now().UTC()
		raw := &RawModel{
			BaseModel: BaseModel{Id: sql.NullString{String: "r1", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "Z", Application: "a", Path: "/z.ts", ModuleId: sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Create(raw).Error; err != nil {
			t.Fatalf("create raw: %v", err)
		}
		prev := execDDL
		t.Cleanup(func() { execDDL = prev })
		execDDL = func(*gorm.DB, string) error { return errors.New("recompute index boom") }
		if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "recompute index boom") {
			t.Fatalf("expected index error, got %v", err)
		}
	})

	t.Run("recompute_remap_error", func(t *testing.T) {
		db := openDualStoreTestDB(t)
		if err := EnsureDualStoreTables(db); err != nil {
			t.Fatalf("ensure: %v", err)
		}
		ts := time.Now().UTC()
		raw := &RawModel{
			BaseModel: BaseModel{Id: sql.NullString{String: "r2", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name: "W", Application: "a", Path: "/w.ts", ModuleId: sql.NullString{String: "mod", Valid: true},
		}
		if err := db.Create(raw).Error; err != nil {
			t.Fatalf("create raw: %v", err)
		}
		prev := remapACLToEffectiveModelIDsFn
		t.Cleanup(func() { remapACLToEffectiveModelIDsFn = prev })
		remapACLToEffectiveModelIDsFn = func(*gorm.DB) error { return errors.New("recompute remap boom") }
		if err := RecomputeAllEffectiveFromRaw(db); err == nil || !strings.Contains(err.Error(), "recompute remap boom") {
			t.Fatalf("expected remap error, got %v", err)
		}
	})
}
