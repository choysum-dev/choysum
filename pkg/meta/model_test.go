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
		reflect.TypeOf(&Application{}),
		reflect.TypeOf(&Module{}),
		reflect.TypeOf(&Component{}),
		reflect.TypeOf(&Model{}),
		reflect.TypeOf(&Field{}),
		reflect.TypeOf(&Service{}),
		reflect.TypeOf(&TypeParameter{}),
		reflect.TypeOf(&Parameter{}),
		reflect.TypeOf(&Decorator{}),
		reflect.TypeOf(&Argument{}),
		reflect.TypeOf(&UiResource{}),
		reflect.TypeOf(&UiResourceMenuRoute{}),
		reflect.TypeOf(&UiResourceRouteAction{}),
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
		{name: "Application", got: (&Application{}).TableName(), want: "meta_application"},
		{name: "Module", got: (&Module{}).TableName(), want: "meta_module"},
		{name: "Component", got: (&Component{}).TableName(), want: "meta_component"},
		{name: "Model", got: (&Model{}).TableName(), want: "meta_model"},
		{name: "Field", got: (&Field{}).TableName(), want: "meta_field"},
		{name: "Service", got: (&Service{}).TableName(), want: "meta_service"},
		{name: "TypeParameter", got: (&TypeParameter{}).TableName(), want: "meta_type_parameter"},
		{name: "Parameter", got: (&Parameter{}).TableName(), want: "meta_parameter"},
		{name: "Decorator", got: (&Decorator{}).TableName(), want: "meta_decorator"},
		{name: "Argument", got: (&Argument{}).TableName(), want: "meta_argument"},
		{name: "UiResource", got: (&UiResource{}).TableName(), want: "meta_ui_resource"},
		{name: "UiResourceMenuRoute", got: (&UiResourceMenuRoute{}).TableName(), want: "meta_ui_resource_menu_route"},
		{name: "UiResourceRouteAction", got: (&UiResourceRouteAction{}).TableName(), want: "meta_ui_resource_route_action"},
	}

	for _, check := range tableNames {
		if check.got != check.want {
			t.Fatalf("%s.TableName() = %q, want %q", check.name, check.got, check.want)
		}
	}
}
