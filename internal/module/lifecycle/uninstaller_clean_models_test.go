// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type cleanModelsSeed struct {
	module        *meta.Module
	rawModel      *meta.RawModel
	rawService    *meta.RawService
	rawField      *meta.RawField
	rawDecorator  *meta.RawDecorator
	rawArgument   *meta.RawArgument
	component     *meta.Component
	compDecorator *meta.Decorator
	compArgument  *meta.Argument
	menuUI        *meta.UiResource
	routeUI       *meta.UiResource
	actionUI      *meta.UiResource
	menuRoute     *meta.UiResourceMenuRoute
	routeAction   *meta.UiResourceRouteAction
	rawTypeParam  *meta.RawTypeParameter
	rawParameter  *meta.RawParameter
	dependModule  *meta.Module
}

func seedCleanModelsFixture(t *testing.T, db *gorm.DB) cleanModelsSeed {
	t.Helper()

	mod := &meta.Module{Name: "demo", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	rawModel := &meta.RawModel{
		Name:        "demo.model",
		Path:        "demo/model",
		Application: "demo",
		ModuleId:    mod.Id,
	}
	if err := db.Create(rawModel).Error; err != nil {
		t.Fatalf("create raw model: %v", err)
	}

	rawService := &meta.RawService{Name: "search", ModelId: rawModel.Id}
	if err := db.Create(rawService).Error; err != nil {
		t.Fatalf("create raw service: %v", err)
	}

	rawField := &meta.RawField{Name: "name", ModelId: rawModel.Id}
	if err := db.Create(rawField).Error; err != nil {
		t.Fatalf("create raw field: %v", err)
	}

	rawDecorator := &meta.RawDecorator{Name: "readonly", ModelId: rawModel.Id}
	if err := db.Create(rawDecorator).Error; err != nil {
		t.Fatalf("create raw decorator: %v", err)
	}

	rawArgument := &meta.RawArgument{Type: "string", Value: "true", DecoratorId: rawDecorator.Id}
	if err := db.Create(rawArgument).Error; err != nil {
		t.Fatalf("create raw argument: %v", err)
	}

	component := &meta.Component{Name: "demo.comp", Path: "demo/comp", ModuleId: mod.Id}
	if err := db.Create(component).Error; err != nil {
		t.Fatalf("create component: %v", err)
	}

	compDecorator := &meta.Decorator{Name: "comp_readonly", ComponentId: component.Id}
	if err := db.Create(compDecorator).Error; err != nil {
		t.Fatalf("create component decorator: %v", err)
	}

	compArgument := &meta.Argument{Type: "string", Value: "true", DecoratorId: compDecorator.Id}
	if err := db.Create(compArgument).Error; err != nil {
		t.Fatalf("create component argument: %v", err)
	}

	menuUI := &meta.UiResource{Name: "demo.menu", Type: meta.UiResourceTypeMenu, ModuleId: mod.Id}
	routeUI := &meta.UiResource{Name: "demo.route", Type: meta.UiResourceTypeRoute, ModuleId: mod.Id}
	actionUI := &meta.UiResource{Name: "demo.action", Type: meta.UiResourceTypeAction, ModuleId: mod.Id}
	for _, row := range []*meta.UiResource{menuUI, routeUI, actionUI} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create ui resource: %v", err)
		}
	}

	menuRoute := &meta.UiResourceMenuRoute{
		MenuUiResourceId:  menuUI.Id,
		RouteUiResourceId: routeUI.Id,
	}
	if err := db.Create(menuRoute).Error; err != nil {
		t.Fatalf("create ui menu-route relation: %v", err)
	}

	routeAction := &meta.UiResourceRouteAction{
		RouteUiResourceId:  routeUI.Id,
		ActionUiResourceId: actionUI.Id,
	}
	if err := db.Create(routeAction).Error; err != nil {
		t.Fatalf("create ui route-action relation: %v", err)
	}

	rawTypeParam := &meta.RawTypeParameter{Name: "T", ServiceId: rawService.Id}
	if err := db.Create(rawTypeParam).Error; err != nil {
		t.Fatalf("create raw type parameter: %v", err)
	}

	rawParameter := &meta.RawParameter{Name: "id", ServiceId: rawService.Id}
	if err := db.Create(rawParameter).Error; err != nil {
		t.Fatalf("create raw parameter: %v", err)
	}

	dependModule := &meta.Module{Name: "dep", Status: meta.Installed, Version: "1.0.0"}
	dependModule.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(dependModule).Error; err != nil {
		t.Fatalf("create depend module: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO meta_module_dependencies (module_id, depend_module_id) VALUES (?, ?)",
		mod.Id.String, dependModule.Id.String,
	).Error; err != nil {
		t.Fatalf("create module dependency: %v", err)
	}

	return cleanModelsSeed{
		module:        mod,
		rawModel:      rawModel,
		rawService:    rawService,
		rawField:      rawField,
		rawDecorator:  rawDecorator,
		rawArgument:   rawArgument,
		component:     component,
		compDecorator: compDecorator,
		compArgument:  compArgument,
		menuUI:        menuUI,
		routeUI:       routeUI,
		actionUI:      actionUI,
		menuRoute:     menuRoute,
		routeAction:   routeAction,
		rawTypeParam:  rawTypeParam,
		rawParameter:  rawParameter,
		dependModule:  dependModule,
	}
}

func dropMetaTable(t *testing.T, db *gorm.DB, model any) {
	t.Helper()
	if err := db.Migrator().DropTable(model); err != nil {
		t.Fatalf("drop table for %#T: %v", model, err)
	}
}

func blockMetaSoftDeletes(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	stmt := `
CREATE TRIGGER block_` + table + `_soft_delete
BEFORE UPDATE OF deleted_at ON ` + table + `
WHEN NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL
BEGIN
  SELECT RAISE(ABORT, 'soft delete blocked');
END`
	if err := db.Exec(stmt).Error; err != nil {
		t.Fatalf("create soft delete trigger on %s: %v", table, err)
	}
}

func deleteRawCleanModelsPrefix(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
	t.Helper()
	if err := db.Unscoped().Delete(seed.rawArgument).Error; err != nil {
		t.Fatalf("delete raw argument: %v", err)
	}
	if err := db.Unscoped().Delete(seed.rawDecorator).Error; err != nil {
		t.Fatalf("delete raw decorator: %v", err)
	}
	if err := db.Unscoped().Delete(seed.rawTypeParam).Error; err != nil {
		t.Fatalf("delete raw type parameter: %v", err)
	}
	if err := db.Unscoped().Delete(seed.rawParameter).Error; err != nil {
		t.Fatalf("delete raw parameter: %v", err)
	}
}

func deleteRawModelTree(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
	t.Helper()
	deleteRawCleanModelsPrefix(t, db, seed)
	if err := db.Unscoped().Delete(seed.rawService).Error; err != nil {
		t.Fatalf("delete raw service: %v", err)
	}
	if err := db.Unscoped().Delete(seed.rawField).Error; err != nil {
		t.Fatalf("delete raw field: %v", err)
	}
	if err := db.Unscoped().Delete(seed.rawModel).Error; err != nil {
		t.Fatalf("delete raw model: %v", err)
	}
}

func TestModuleUninstallerCleanModelsErrorPaths(t *testing.T) {
	cases := []struct {
		name    string
		wantMsg string
		setup   func(t *testing.T, db *gorm.DB, seed cleanModelsSeed)
	}{
		{
			name:    "status update",
			wantMsg: "error updating module status",
			setup: func(t *testing.T, db *gorm.DB, _ cleanModelsSeed) {
				if err := db.Exec(`
CREATE TRIGGER block_module_status_update
BEFORE UPDATE OF status ON meta_module
BEGIN
  SELECT RAISE(ABORT, 'status update blocked');
END`).Error; err != nil {
					t.Fatalf("create status update trigger: %v", err)
				}
			},
		},
		{
			name:    "raw decorator arguments",
			wantMsg: "error deleting raw decorator arguments",
			setup: func(t *testing.T, db *gorm.DB, _ cleanModelsSeed) {
				dropMetaTable(t, db, &meta.RawArgument{})
			},
		},
		{
			name:    "raw decorators",
			wantMsg: "error deleting raw decorators",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				if err := db.Unscoped().Delete(seed.rawArgument).Error; err != nil {
					t.Fatalf("delete raw argument: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_raw_decorator")
			},
		},
		{
			name:    "component decorator arguments",
			wantMsg: "error deleting component decorator arguments",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				dropMetaTable(t, db, &meta.Argument{})
			},
		},
		{
			name:    "component decorators",
			wantMsg: "error deleting component decorators",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_decorator")
			},
		},
		{
			name:    "raw type parameters",
			wantMsg: "error deleting raw type parameters",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				dropMetaTable(t, db, &meta.RawTypeParameter{})
			},
		},
		{
			name:    "raw parameters",
			wantMsg: "error deleting raw parameters",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawTypeParam).Error; err != nil {
					t.Fatalf("delete raw type parameter: %v", err)
				}
				dropMetaTable(t, db, &meta.RawParameter{})
			},
		},
		{
			name:    "raw services",
			wantMsg: "error deleting raw services",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_raw_service")
			},
		},
		{
			name:    "raw fields",
			wantMsg: "error deleting raw fields",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawService).Error; err != nil {
					t.Fatalf("delete raw service: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_raw_field")
			},
		},
		{
			name:    "raw models",
			wantMsg: "error deleting raw models",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawService).Error; err != nil {
					t.Fatalf("delete raw service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawField).Error; err != nil {
					t.Fatalf("delete raw field: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_raw_model")
			},
		},
		{
			name:    "recompute effective",
			wantMsg: "error recomputing effective models after uninstall",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawService).Error; err != nil {
					t.Fatalf("delete raw service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.rawField).Error; err != nil {
					t.Fatalf("delete raw field: %v", err)
				}
				dropMetaTable(t, db, &meta.Model{})
			},
		},
		{
			name:    "components",
			wantMsg: "error deleting components",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawModelTree(t, db, seed)
				blockMetaSoftDeletes(t, db, "meta_component")
			},
		},
		{
			name:    "ui menu-route relations",
			wantMsg: "error deleting UI resource menu-route relations",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawModelTree(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.component).Error; err != nil {
					t.Fatalf("delete component: %v", err)
				}
				dropMetaTable(t, db, &meta.UiResourceMenuRoute{})
			},
		},
		{
			name:    "ui route-action relations",
			wantMsg: "error deleting UI resource route-action relations",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawModelTree(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.component).Error; err != nil {
					t.Fatalf("delete component: %v", err)
				}
				if err := db.Unscoped().Delete(seed.menuRoute).Error; err != nil {
					t.Fatalf("delete menu-route relation: %v", err)
				}
				dropMetaTable(t, db, &meta.UiResourceRouteAction{})
			},
		},
		{
			name:    "ui resources",
			wantMsg: "error deleting UI resources",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteRawModelTree(t, db, seed)
				if err := db.Unscoped().Delete(seed.compArgument).Error; err != nil {
					t.Fatalf("delete component argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.compDecorator).Error; err != nil {
					t.Fatalf("delete component decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.component).Error; err != nil {
					t.Fatalf("delete component: %v", err)
				}
				if err := db.Unscoped().Delete(seed.menuRoute).Error; err != nil {
					t.Fatalf("delete menu-route relation: %v", err)
				}
				if err := db.Unscoped().Delete(seed.routeAction).Error; err != nil {
					t.Fatalf("delete route-action relation: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_ui_resource")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtimeScope := newLifecycleCommitTestScope(t)
			db := runtimeScope.Session().DB
			seed := seedCleanModelsFixture(t, db)
			tc.setup(t, db, seed)

			uninstaller := &moduleUninstaller{
				runtimeScope:  runtimeScope,
				module:        seed.module,
				moduleManager: &ModuleManager{runtimeScope: runtimeScope},
				ctx:           newOpContext(),
			}
			err := uninstaller.cleanModels()
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("cleanModels() error = %v, want substring %q", err, tc.wantMsg)
			}
		})
	}
}
