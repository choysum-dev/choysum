// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

type namedDialector struct {
	gorm.Dialector
	name string
}

func (d namedDialector) Name() string { return d.name }

func closeDualStoreDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
func TestEnsureEffectiveAppNameUniqueIndex_NilDB(t *testing.T) {
	if err := ensureEffectiveAppNameUniqueIndex(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
}
func TestEnsureBaseModelID_Nil(t *testing.T) {
	ensureBaseModelID(nil) // no panic
}
func TestEnsureDualStoreTables_NilAndClosed(t *testing.T) {
	if err := ensureDualStoreTables(nil); err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("expected nil db error, got %v", err)
	}
	db := openDualStoreTestDB(t)
	closeDualStoreDB(t, db)
	if err := ensureDualStoreTables(db); err == nil {
		t.Fatal("expected AutoMigrate error on closed db")
	}
}
func TestCopyModelTreeToRaw_CreateFailures(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	existing := &rawModel{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "existing", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(existing).Error; err != nil {
		t.Fatalf("seed raw: %v", err)
	}
	src := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		ModuleId:    sql.NullString{String: "mod", Valid: true},
	}
	if err := copyModelTreeToRaw(db, src); err == nil {
		t.Fatal("expected unique violation creating raw model")
	}
}
func TestCopyModelTreeToRaw_ChildTableFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		drop any
		src  *pkgmeta.Model
	}{
		{
			name: "raw field",
			drop: &rawField{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "X",
				Application: "a",
				Path:        "/x.ts",
				ModuleId:    sql.NullString{String: "mod", Valid: true},
				Fields: []*pkgmeta.Field{{
					BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "f", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Name",
				}},
			},
		},
		{
			name: "raw service",
			drop: &rawService{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m2", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "Y",
				Application: "a",
				Path:        "/y.ts",
				ModuleId:    sql.NullString{String: "mod2", Valid: true},
				Services: []*pkgmeta.Service{{
					BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "s", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Create",
				}},
			},
		},
		{
			name: "raw parameter",
			drop: &rawParameter{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m3", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "Z",
				Application: "a",
				Path:        "/z.ts",
				ModuleId:    sql.NullString{String: "mod3", Valid: true},
				Services: []*pkgmeta.Service{{
					BaseModel:  pkgmeta.BaseModel{Id: sql.NullString{String: "s3", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:       "Create",
					Parameters: []*pkgmeta.Parameter{{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "p", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "vals"}},
				}},
			},
		},
		{
			name: "raw type parameter",
			drop: &rawTypeParameter{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m4", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "W",
				Application: "a",
				Path:        "/w.ts",
				ModuleId:    sql.NullString{String: "mod4", Valid: true},
				Services: []*pkgmeta.Service{{
					BaseModel:      pkgmeta.BaseModel{Id: sql.NullString{String: "s4", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:           "Create",
					TypeParameters: []*pkgmeta.TypeParameter{{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "tp", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "T"}},
				}},
			},
		},
		{
			name: "raw decorator",
			drop: &rawDecorator{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m5", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "V",
				Application: "a",
				Path:        "/v.ts",
				ModuleId:    sql.NullString{String: "mod5", Valid: true},
				Decorators: []*pkgmeta.Decorator{{
					BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "d5", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Model",
				}},
			},
		},
		{
			name: "raw argument",
			drop: &rawArgument{},
			src: &pkgmeta.Model{
				BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "m6", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
				Name:        "U",
				Application: "a",
				Path:        "/u.ts",
				ModuleId:    sql.NullString{String: "mod6", Valid: true},
				Decorators: []*pkgmeta.Decorator{{
					BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "d6", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
					Name:      "Model",
					Arguments: []*pkgmeta.Argument{{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "a6", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Type: "string", Value: `"U"`}},
				}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openDualStoreTestDB(t)
			if err := ensureDualStoreTables(db); err != nil {
				t.Fatalf("ensure: %v", err)
			}
			if err := db.Migrator().DropTable(tc.drop); err != nil {
				t.Fatalf("drop %s: %v", tc.name, err)
			}
			if err := copyModelTreeToRaw(db, tc.src); err == nil {
				t.Fatalf("expected %s create failure", tc.name)
			}
		})
	}

	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure nil-decorator db: %v", err)
	}
	if err := copyDecoratorToRaw(db, nil, sql.NullString{}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("nil decorator: %v", err)
	}
}
func TestPersistEffectiveProjection_FailuresAndNils(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	merged := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/x.ts",
		Decorators:  []*pkgmeta.Decorator{nil},
		Fields:      []*pkgmeta.Field{nil, {Name: "Name", OriginModelPath: "kept"}},
		Services: []*pkgmeta.Service{nil, {
			Name:            "Create",
			OriginModelPath: "svc-origin",
			Parameters:      []*pkgmeta.Parameter{nil, {Name: "this"}, {Name: "vals"}},
			TypeParameters:  []*pkgmeta.TypeParameter{nil, {Name: "T"}},
			Decorators:      []*pkgmeta.Decorator{{Name: "Rpc"}},
		}},
	}
	if err := persistEffectiveProjection(db, merged, "eff1", nil); err != nil {
		t.Fatalf("persist: %v", err)
	}

	// model create failure
	if err := persistEffectiveProjection(db, merged, "eff1", nil); err == nil {
		t.Fatal("expected duplicate effective id failure")
	}

	db2 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db2)
	_ = db2.Migrator().DropTable(&pkgmeta.Field{})
	if err := persistEffectiveProjection(db2, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Fields: []*pkgmeta.Field{{Name: "Name"}},
	}, "e2", nil); err == nil {
		t.Fatal("expected field create failure")
	}

	db3 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db3)
	_ = db3.Migrator().DropTable(&pkgmeta.Service{})
	if err := persistEffectiveProjection(db3, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*pkgmeta.Service{{Name: "Create"}},
	}, "e3", nil); err == nil {
		t.Fatal("expected service create failure")
	}

	db4 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db4)
	_ = db4.Migrator().DropTable(&pkgmeta.Parameter{})
	if err := persistEffectiveProjection(db4, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*pkgmeta.Service{{Name: "Create", Parameters: []*pkgmeta.Parameter{{Name: "vals"}}}},
	}, "e4", nil); err == nil {
		t.Fatal("expected parameter create failure")
	}

	db5 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db5)
	_ = db5.Migrator().DropTable(&pkgmeta.TypeParameter{})
	if err := persistEffectiveProjection(db5, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*pkgmeta.Service{{Name: "Create", TypeParameters: []*pkgmeta.TypeParameter{{Name: "T"}}}},
	}, "e5", nil); err == nil {
		t.Fatal("expected type parameter create failure")
	}

	db6 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db6)
	_ = db6.Migrator().DropTable(&pkgmeta.Decorator{})
	if err := persistEffectiveProjection(db6, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Decorators: []*pkgmeta.Decorator{{Name: "Model"}},
	}, "e6", nil); err == nil {
		t.Fatal("expected decorator create failure")
	}

	db7 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db7)
	_ = db7.Migrator().DropTable(&pkgmeta.Argument{})
	if err := persistDecoratorTree(db7, &pkgmeta.Decorator{
		Name: "Model", Arguments: []*pkgmeta.Argument{nil, {Type: "string", Value: `"X"`}},
	}, sql.NullString{String: "mid", Valid: true}, sql.NullString{}, sql.NullString{}); err == nil {
		t.Fatal("expected argument create failure after dropping meta_argument")
	}

	if err := persistDecoratorTree(db7, nil, sql.NullString{}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("nil decorator: %v", err)
	}
}
func TestEnsureEffectiveAppNameUniqueIndex_Dialects(t *testing.T) {
	// namedDialector only overrides Name() for branch selection; SQL still runs on SQLite.
	// Each case opens a fresh DB so Dialector mutation does not leak across subtests.
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("sqlite index: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "postgres"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("postgres index: %v", err)
	}

	db2 := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db2); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	db2.Dialector = namedDialector{Dialector: db2.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db2); err != nil {
		t.Fatalf("mysql first: %v", err)
	}
	// second call: HasIndex true → DropIndex → Create
	if err := ensureEffectiveAppNameUniqueIndex(db2); err != nil {
		t.Fatalf("mysql second: %v", err)
	}

	// CREATE failure when table is missing (first step creates temp index).
	_ = db2.Migrator().DropTable(&pkgmeta.Model{})
	if err := ensureEffectiveAppNameUniqueIndex(db2); err == nil {
		t.Fatal("expected mysql create index failure")
	}

	// sqlite CREATE failure
	db3 := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db3); err != nil {
		t.Fatalf("ensure3: %v", err)
	}
	_ = db3.Migrator().DropTable(&pkgmeta.Model{})
	if err := ensureEffectiveAppNameUniqueIndex(db3); err == nil || !strings.Contains(err.Error(), "create unique index") {
		t.Fatalf("expected sqlite create index failure, got %v", err)
	}

	// sqlite/postgres DDL failure must surface (closed DB; first step is CREATE temp).
	db4 := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db4); err != nil {
		t.Fatalf("ensure4: %v", err)
	}
	closeDualStoreDB(t, db4)
	if err := ensureEffectiveAppNameUniqueIndex(db4); err == nil || !strings.Contains(err.Error(), "create unique index") {
		t.Fatalf("expected create unique index error on closed db, got %v", err)
	}
}
func TestEnsureEffectiveAppNameUniqueIndex_MySQLDropError(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Break migrator by swapping dialector to one that reports HasIndex but fails DropIndex.
	// HasIndex(temp) is also true, so temp create is skipped; drop of final still fails.
	db.Dialector = failingDropDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop unique index") {
		t.Fatalf("expected drop error, got %v", err)
	}
}
func TestEnsurePartialAliveAppNameUniqueIndex_PreservesUniquenessOnCreateFinalFailure(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	soft := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "soft", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p1.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(soft).Error; err != nil {
		t.Fatalf("create soft: %v", err)
	}
	if err := db.Delete(soft).Error; err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	prev := execDDL
	t.Cleanup(func() { execDDL = prev })
	execDDL = func(db *gorm.DB, sql string) error {
		// Allow temp create + drop of final; fail recreate of the final name.
		if strings.Contains(sql, "CREATE UNIQUE INDEX") &&
			strings.Contains(sql, effectiveAppNameUniqueIndex) &&
			!strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("create final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "create final boom") {
		t.Fatalf("expected create final failure, got %v", err)
	}
	// Temp partial index must still protect live uniqueness and allow soft-deleted reuse.
	live := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "live", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p2.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(live).Error; err != nil {
		t.Fatalf("live row after soft-delete should succeed under temp partial index: %v", err)
	}
	dup := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "dup", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Partner",
		Application: "partner",
		Path:        "/p3.ts",
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(dup).Error; err == nil {
		t.Fatal("expected live (application, name) uniqueness still enforced")
	}
}
func TestEnsurePartialAliveAppNameUniqueIndex_DropErrors(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	prev := execDDL
	t.Cleanup(func() { execDDL = prev })

	execDDL = func(db *gorm.DB, sql string) error {
		if strings.HasPrefix(strings.TrimSpace(sql), "DROP INDEX") && strings.Contains(sql, effectiveAppNameUniqueIndex) && !strings.Contains(sql, "_new") {
			return errors.New("drop final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop final boom") {
		t.Fatalf("expected drop final error, got %v", err)
	}

	execDDL = func(db *gorm.DB, sql string) error {
		if strings.HasPrefix(strings.TrimSpace(sql), "DROP INDEX") && strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("drop temp boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop temp boom") {
		t.Fatalf("expected drop temp error, got %v", err)
	}
}
func TestEnsureFullAppNameUniqueIndex_CreateFinalAndDropTempErrors(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	db.Dialector = namedDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err != nil {
		t.Fatalf("initial: %v", err)
	}

	prev := execDDL
	t.Cleanup(func() { execDDL = prev })
	execDDL = func(db *gorm.DB, sql string) error {
		if strings.Contains(sql, "CREATE UNIQUE INDEX") &&
			strings.Contains(sql, effectiveAppNameUniqueIndex) &&
			!strings.Contains(sql, effectiveAppNameUniqueIndexTemp) {
			return errors.New("mysql create final boom")
		}
		return prev(db, sql)
	}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "mysql create final boom") {
		t.Fatalf("expected mysql create final error, got %v", err)
	}

	execDDL = prev
	// Leave temp in place from the failed run, then force DropIndex(temp) to fail.
	db.Dialector = failingDropTempDialector{Dialector: db.Dialector, name: "mysql"}
	if err := ensureEffectiveAppNameUniqueIndex(db); err == nil || !strings.Contains(err.Error(), "drop boom temp") {
		t.Fatalf("expected drop temp error, got %v", err)
	}
}

type failingDropTempDialector struct {
	gorm.Dialector
	name string
}

func (d failingDropTempDialector) Name() string { return d.name }

func (d failingDropTempDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return failingDropTempMigrator{Migrator: d.Dialector.Migrator(db)}
}

type failingDropTempMigrator struct {
	gorm.Migrator
}

func (m failingDropTempMigrator) HasIndex(dst interface{}, name string) bool {
	return m.Migrator.HasIndex(dst, name)
}

func (m failingDropTempMigrator) DropIndex(dst interface{}, name string) error {
	if name == effectiveAppNameUniqueIndexTemp {
		return errors.New("drop boom temp")
	}
	return m.Migrator.DropIndex(dst, name)
}

type failingDropDialector struct {
	gorm.Dialector
	name string
}

func (d failingDropDialector) Name() string { return d.name }

func (d failingDropDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return failingDropMigrator{Migrator: d.Dialector.Migrator(db)}
}

type failingDropMigrator struct {
	gorm.Migrator
}

func (m failingDropMigrator) HasIndex(dst interface{}, name string) bool { return true }
func (m failingDropMigrator) DropIndex(dst interface{}, name string) error {
	return errors.New("drop boom")
}

func TestCopyDecoratorToRaw_NilArgument(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	d := &pkgmeta.Decorator{
		BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "d", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:      "Model",
		Arguments: []*pkgmeta.Argument{nil},
	}
	if err := copyDecoratorToRaw(db, d, sql.NullString{String: "m", Valid: true}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("copyDecoratorToRaw: %v", err)
	}
}
func TestCopyModelTreeToRaw_NestedDecoratorFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	src := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "mf", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "X",
		Application: "a",
		Path:        "/xf.ts",
		ModuleId:    sql.NullString{String: "modf", Valid: true},
		Fields: []*pkgmeta.Field{{
			BaseModel:  pkgmeta.BaseModel{Id: sql.NullString{String: "ff", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:       "Name",
			Decorators: []*pkgmeta.Decorator{{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "df", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "Field"}},
		}},
	}
	if err := db.Migrator().DropTable(&rawDecorator{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := copyModelTreeToRaw(db, src); err == nil {
		t.Fatal("expected field decorator create failure")
	}

	db2 := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db2); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	src2 := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "ms", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "Y",
		Application: "a",
		Path:        "/ys.ts",
		ModuleId:    sql.NullString{String: "mods", Valid: true},
		Services: []*pkgmeta.Service{{
			BaseModel:  pkgmeta.BaseModel{Id: sql.NullString{String: "ss", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:       "Create",
			Decorators: []*pkgmeta.Decorator{{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "ds", Valid: true}, CreatedAt: ts, UpdatedAt: ts}, Name: "Rpc"}},
		}},
	}
	if err := db2.Migrator().DropTable(&rawDecorator{}); err != nil {
		t.Fatalf("drop2: %v", err)
	}
	if err := copyModelTreeToRaw(db2, src2); err == nil {
		t.Fatal("expected service decorator create failure")
	}
}
func TestPersistEffectiveProjection_NestedDecoratorFailures(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	db := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db)
	_ = db.Migrator().DropTable(&pkgmeta.Decorator{})
	if err := persistEffectiveProjection(db, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Fields: []*pkgmeta.Field{{Name: "Name", Decorators: []*pkgmeta.Decorator{{Name: "Field"}}}},
	}, "ef", nil); err == nil {
		t.Fatal("expected field decorator persist failure")
	}

	db2 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db2)
	_ = db2.Migrator().DropTable(&pkgmeta.Decorator{})
	if err := persistEffectiveProjection(db2, &pkgmeta.Model{
		BaseModel: pkgmeta.BaseModel{CreatedAt: ts, UpdatedAt: ts}, Name: "X", Application: "a", Path: "/x.ts",
		Services: []*pkgmeta.Service{{Name: "Create", Decorators: []*pkgmeta.Decorator{{Name: "Rpc"}}}},
	}, "es", nil); err == nil {
		t.Fatal("expected service decorator persist failure")
	}

	db3 := openDualStoreTestDB(t)
	_ = ensureDualStoreTables(db3)
	if err := persistDecoratorTree(db3, &pkgmeta.Decorator{
		Name: "Model", Arguments: []*pkgmeta.Argument{nil, {Type: "string", Value: `"X"`}},
	}, sql.NullString{String: "no-model", Valid: true}, sql.NullString{}, sql.NullString{}); err != nil {
		t.Fatalf("persistDecoratorTree with nil arg should succeed on sqlite: %v", err)
	}
	_ = db3.Migrator().DropTable(&pkgmeta.Argument{})
	if err := persistDecoratorTree(db3, &pkgmeta.Decorator{
		Name: "Model2", Arguments: []*pkgmeta.Argument{{Type: "string", Value: `"Y"`}},
	}, sql.NullString{}, sql.NullString{}, sql.NullString{}); err == nil {
		t.Fatal("expected argument create failure")
	}
}

func TestRawIsNewerTip(t *testing.T) {
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	a := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "a", Valid: true}, CreatedAt: ts}}
	b := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "b", Valid: true}, CreatedAt: ts}}
	if !rawIsNewerTip(a, nil) {
		t.Fatal("nil previous")
	}
	if !rawIsNewerTip(b, a) { // equal time, higher id
		t.Fatal("expected b newer by id")
	}
	older := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "z", Valid: true}, CreatedAt: ts.Add(-time.Hour)}}
	if rawIsNewerTip(older, a) {
		t.Fatal("older should not win")
	}
	newer := &rawModel{BaseModel: pkgmeta.BaseModel{Id: sql.NullString{String: "c", Valid: true}, CreatedAt: ts.Add(time.Hour)}}
	if !rawIsNewerTip(newer, a) {
		t.Fatal("newer CreatedAt should win")
	}
}

func TestCopyModelTreeToRaw_NilChildren(t *testing.T) {
	db := openDualStoreTestDB(t)
	if err := ensureDualStoreTables(db); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ts := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	srcNil := &pkgmeta.Model{
		BaseModel:   pkgmeta.BaseModel{Id: sql.NullString{String: "mn", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
		Name:        "NilKids",
		Application: "a",
		Path:        "/nil.ts",
		ModuleId:    sql.NullString{String: "modn", Valid: true},
		Fields:      []*pkgmeta.Field{nil},
		Services: []*pkgmeta.Service{nil, {
			BaseModel:      pkgmeta.BaseModel{Id: sql.NullString{String: "sn", Valid: true}, CreatedAt: ts, UpdatedAt: ts},
			Name:           "S",
			Parameters:     []*pkgmeta.Parameter{nil},
			TypeParameters: []*pkgmeta.TypeParameter{nil},
		}},
		Decorators: nil,
	}
	if err := copyModelTreeToRaw(db, srcNil); err != nil {
		t.Fatalf("copy with nils: %v", err)
	}
}
