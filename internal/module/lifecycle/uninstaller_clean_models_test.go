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
	module       *meta.Module
	model        *meta.Model
	service      *meta.Service
	field        *meta.Field
	decorator    *meta.Decorator
	argument     *meta.Argument
	component    *meta.Component
	menuUI       *meta.UiResource
	routeUI      *meta.UiResource
	actionUI     *meta.UiResource
	menuRoute    *meta.UiResourceMenuRoute
	routeAction  *meta.UiResourceRouteAction
	typeParam    *meta.TypeParameter
	parameter    *meta.Parameter
	dependModule *meta.Module
}

func seedCleanModelsFixture(t *testing.T, db *gorm.DB) cleanModelsSeed {
	t.Helper()

	mod := &meta.Module{Name: "demo", Status: meta.Installed, Version: "1.0.0"}
	mod.Id = sql.NullString{String: xid.New().String(), Valid: true}
	if err := db.Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	model := &meta.Model{Name: "demo.model", Path: "demo/model", ModuleId: mod.Id}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	service := &meta.Service{Name: "search", ModelId: model.Id}
	if err := db.Create(service).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	field := &meta.Field{Name: "name", ModelId: model.Id}
	if err := db.Create(field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}

	decorator := &meta.Decorator{Name: "readonly", ModelId: model.Id}
	if err := db.Create(decorator).Error; err != nil {
		t.Fatalf("create decorator: %v", err)
	}

	argument := &meta.Argument{Type: "string", Value: "true", DecoratorId: decorator.Id}
	if err := db.Create(argument).Error; err != nil {
		t.Fatalf("create argument: %v", err)
	}

	component := &meta.Component{Name: "demo.comp", Path: "demo/comp", ModuleId: mod.Id}
	if err := db.Create(component).Error; err != nil {
		t.Fatalf("create component: %v", err)
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

	typeParam := &meta.TypeParameter{Name: "T", ServiceId: service.Id}
	if err := db.Create(typeParam).Error; err != nil {
		t.Fatalf("create type parameter: %v", err)
	}

	parameter := &meta.Parameter{Name: "id", ServiceId: service.Id}
	if err := db.Create(parameter).Error; err != nil {
		t.Fatalf("create parameter: %v", err)
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
		module:       mod,
		model:        model,
		service:      service,
		field:        field,
		decorator:    decorator,
		argument:     argument,
		component:    component,
		menuUI:       menuUI,
		routeUI:      routeUI,
		actionUI:     actionUI,
		menuRoute:    menuRoute,
		routeAction:  routeAction,
		typeParam:    typeParam,
		parameter:    parameter,
		dependModule: dependModule,
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

func deleteCleanModelsPrefix(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
	t.Helper()
	if err := db.Unscoped().Delete(seed.argument).Error; err != nil {
		t.Fatalf("delete argument: %v", err)
	}
	if err := db.Unscoped().Delete(seed.decorator).Error; err != nil {
		t.Fatalf("delete decorator: %v", err)
	}
	if err := db.Unscoped().Delete(seed.typeParam).Error; err != nil {
		t.Fatalf("delete type parameter: %v", err)
	}
	if err := db.Unscoped().Delete(seed.parameter).Error; err != nil {
		t.Fatalf("delete parameter: %v", err)
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
			name:    "decorator arguments",
			wantMsg: "error deleting decorator arguments",
			setup: func(t *testing.T, db *gorm.DB, _ cleanModelsSeed) {
				dropMetaTable(t, db, &meta.Argument{})
			},
		},
		{
			name:    "decorators",
			wantMsg: "error deleting decorators",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				if err := db.Unscoped().Delete(seed.argument).Error; err != nil {
					t.Fatalf("delete argument: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_decorator")
			},
		},
		{
			name:    "type parameters",
			wantMsg: "error deleting type parameters",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				if err := db.Unscoped().Delete(seed.argument).Error; err != nil {
					t.Fatalf("delete argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.decorator).Error; err != nil {
					t.Fatalf("delete decorator: %v", err)
				}
				dropMetaTable(t, db, &meta.TypeParameter{})
			},
		},
		{
			name:    "parameters",
			wantMsg: "error deleting parameters",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				if err := db.Unscoped().Delete(seed.argument).Error; err != nil {
					t.Fatalf("delete argument: %v", err)
				}
				if err := db.Unscoped().Delete(seed.decorator).Error; err != nil {
					t.Fatalf("delete decorator: %v", err)
				}
				if err := db.Unscoped().Delete(seed.typeParam).Error; err != nil {
					t.Fatalf("delete type parameter: %v", err)
				}
				dropMetaTable(t, db, &meta.Parameter{})
			},
		},
		{
			name:    "services",
			wantMsg: "error deleting services",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteCleanModelsPrefix(t, db, seed)
				blockMetaSoftDeletes(t, db, "meta_service")
			},
		},
		{
			name:    "fields",
			wantMsg: "error deleting fields",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_field")
			},
		},
		{
			name:    "models",
			wantMsg: "error deleting models",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.field).Error; err != nil {
					t.Fatalf("delete field: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_model")
			},
		},
		{
			name:    "components",
			wantMsg: "error deleting components",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.field).Error; err != nil {
					t.Fatalf("delete field: %v", err)
				}
				if err := db.Unscoped().Delete(seed.model).Error; err != nil {
					t.Fatalf("delete model: %v", err)
				}
				blockMetaSoftDeletes(t, db, "meta_component")
			},
		},
		{
			name:    "ui menu-route relations",
			wantMsg: "error deleting UI resource menu-route relations",
			setup: func(t *testing.T, db *gorm.DB, seed cleanModelsSeed) {
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.field).Error; err != nil {
					t.Fatalf("delete field: %v", err)
				}
				if err := db.Unscoped().Delete(seed.model).Error; err != nil {
					t.Fatalf("delete model: %v", err)
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
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.field).Error; err != nil {
					t.Fatalf("delete field: %v", err)
				}
				if err := db.Unscoped().Delete(seed.model).Error; err != nil {
					t.Fatalf("delete model: %v", err)
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
				deleteCleanModelsPrefix(t, db, seed)
				if err := db.Unscoped().Delete(seed.service).Error; err != nil {
					t.Fatalf("delete service: %v", err)
				}
				if err := db.Unscoped().Delete(seed.field).Error; err != nil {
					t.Fatalf("delete field: %v", err)
				}
				if err := db.Unscoped().Delete(seed.model).Error; err != nil {
					t.Fatalf("delete model: %v", err)
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
