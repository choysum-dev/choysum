// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"os"
	"testing"

	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestSemanticTypeResolver_MapsAliasesAndLiteralUnions(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: "/virtual/modules/demo", ApplicationStr: "demo", Name: "demo"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/demo/service/model.ts"
	content := `
type Id = string
type Mode = 'read' | 'write'
type Flag = true | false
type Count = 1 | 2

export default class Demo {
  public static async ById(id: Id, mode: Mode, flag: Flag, count: Count): Promise<Id> {
    return id
  }

  public static async Done(): Promise<void | undefined> {}

  public static async Mixed(x: string | number): Promise<{ name: string }> {
    return { name: '' }
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil {
		t.Fatal("expected model")
	}

	byName := map[string]*meta.Service{}
	for _, service := range r.Model.Services {
		byName[service.Name] = service
	}

	byId := byName["ById"]
	if byId == nil {
		t.Fatalf("missing ById, services=%v", r.Model.Services)
	}
	if byId.ProtobufType != "string" {
		t.Fatalf("ById return ProtobufType=%q, want string", byId.ProtobufType)
	}
	if len(byId.Parameters) != 4 {
		t.Fatalf("ById params=%v", byId.Parameters)
	}
	wantParams := []string{"string", "string", "bool", "double"}
	for i, want := range wantParams {
		if byId.Parameters[i].ProtobufType != want {
			t.Fatalf("ById param[%d] ProtobufType=%q, want %q", i, byId.Parameters[i].ProtobufType, want)
		}
	}

	done := byName["Done"]
	if done == nil || done.ProtobufType != "google.protobuf.Empty" {
		t.Fatalf("Done ProtobufType=%v", done)
	}

	mixed := byName["Mixed"]
	if mixed == nil {
		t.Fatal("missing Mixed")
	}
	if mixed.ProtobufType != "google.protobuf.Value" {
		t.Fatalf("Mixed return ProtobufType=%q, want Value", mixed.ProtobufType)
	}
	if len(mixed.Parameters) != 1 || mixed.Parameters[0].ProtobufType != "google.protobuf.Value" {
		t.Fatalf("Mixed params=%v", mixed.Parameters)
	}
}

func TestSemanticTypeResolver_FallsBackWhenDisabled(t *testing.T) {
	t.Setenv(envDisableSemanticProto, "1")

	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: "/virtual/modules/demo", ApplicationStr: "demo", Name: "demo"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/demo/service/model.ts"
	content := `
type Id = string
export default class Demo {
  public static async ById(id: Id): Promise<Id> {
    return id
  }
}
`
	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil || len(r.Model.Services) != 1 {
		t.Fatalf("unexpected model: %+v", r.Model)
	}
	service := r.Model.Services[0]
	// Text mapping cannot see the alias, so both stay as Value.
	if service.ProtobufType != "google.protobuf.Value" {
		t.Fatalf("return ProtobufType=%q, want Value fallback", service.ProtobufType)
	}
	if len(service.Parameters) != 1 || service.Parameters[0].ProtobufType != "google.protobuf.Value" {
		t.Fatalf("params=%v", service.Parameters)
	}
}

func TestMapCheckerTypeToProto_NilSafe(t *testing.T) {
	if got, ok := mapCheckerTypeToProto(nil, nil, true); ok || got != "" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}
