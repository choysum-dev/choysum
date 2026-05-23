// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

func TestBaseModelBeforeCreate(t *testing.T) {
	model := &BaseModel{}
	if err := model.BeforeCreate((*gorm.DB)(nil)); err != nil {
		t.Fatalf("BeforeCreate empty id: %v", err)
	}
	if !model.Id.Valid || model.Id.String == "" {
		t.Fatalf("expected generated id, got %#v", model.Id)
	}

	existing := sql.NullString{String: "preset", Valid: true}
	model = &BaseModel{Id: existing}
	if err := model.BeforeCreate((*gorm.DB)(nil)); err != nil {
		t.Fatalf("BeforeCreate preset id: %v", err)
	}
	if model.Id != existing {
		t.Fatalf("expected preset id to remain unchanged, got %#v", model.Id)
	}
}

func TestEntitiesAndTableNames(t *testing.T) {
	entities := Entities()
	expectedTypes := []reflect.Type{
		reflect.TypeOf(&IrApplication{}),
		reflect.TypeOf(&IrModule{}),
		reflect.TypeOf(&IrComponent{}),
		reflect.TypeOf(&IrModel{}),
		reflect.TypeOf(&IrField{}),
		reflect.TypeOf(&IrService{}),
		reflect.TypeOf(&IrTypeParameter{}),
		reflect.TypeOf(&IrParameter{}),
		reflect.TypeOf(&IrDecorator{}),
		reflect.TypeOf(&IrArgument{}),
		reflect.TypeOf(&IrUiResource{}),
		reflect.TypeOf(&IrUiResourceMenuRoute{}),
		reflect.TypeOf(&IrUiResourceRouteAction{}),
	}

	if len(entities) != len(expectedTypes) {
		t.Fatalf("Entities() len = %d, want %d", len(entities), len(expectedTypes))
	}
	for index, entity := range entities {
		if reflect.TypeOf(entity) != expectedTypes[index] {
			t.Fatalf("Entities()[%d] type = %v, want %v", index, reflect.TypeOf(entity), expectedTypes[index])
		}
	}

	tableNames := []struct {
		name string
		got  string
		want string
	}{
		{name: "IrApplication", got: (&IrApplication{}).TableName(), want: "meta_ir_application"},
		{name: "IrModule", got: (&IrModule{}).TableName(), want: "meta_ir_module"},
		{name: "IrComponent", got: (&IrComponent{}).TableName(), want: "meta_ir_component"},
		{name: "IrModel", got: (&IrModel{}).TableName(), want: "meta_ir_model"},
		{name: "IrField", got: (&IrField{}).TableName(), want: "meta_ir_field"},
		{name: "IrService", got: (&IrService{}).TableName(), want: "meta_ir_service"},
		{name: "IrTypeParameter", got: (&IrTypeParameter{}).TableName(), want: "meta_ir_type_parameter"},
		{name: "IrParameter", got: (&IrParameter{}).TableName(), want: "meta_ir_parameter"},
		{name: "IrDecorator", got: (&IrDecorator{}).TableName(), want: "meta_ir_decorator"},
		{name: "IrArgument", got: (&IrArgument{}).TableName(), want: "meta_ir_argument"},
		{name: "IrUiResource", got: (&IrUiResource{}).TableName(), want: "meta_ir_ui_resource"},
		{name: "IrUiResourceMenuRoute", got: (&IrUiResourceMenuRoute{}).TableName(), want: "meta_ir_ui_resource_menu_route"},
		{name: "IrUiResourceRouteAction", got: (&IrUiResourceRouteAction{}).TableName(), want: "meta_ir_ui_resource_route_action"},
	}

	for _, check := range tableNames {
		if check.got != check.want {
			t.Fatalf("%s.TableName() = %q, want %q", check.name, check.got, check.want)
		}
	}
}
