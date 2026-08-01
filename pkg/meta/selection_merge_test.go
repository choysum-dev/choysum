// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeSelectionByValue(t *testing.T) {
	base := []IrFieldSelectionItem{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B"},
	}
	add := []IrFieldSelectionItem{
		{Value: "c", Label: "C"},
		{Value: "b", Label: "B2"},
	}
	got := MergeSelectionByValue(base, add)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %#v", got)
	}
	if got[0].Value != "a" || got[0].Label != "A" {
		t.Fatalf("unexpected first item: %#v", got[0])
	}
	if got[1].Value != "b" || got[1].Label != "B2" {
		t.Fatalf("expected later label to win for b, got %#v", got[1])
	}
	if got[2].Value != "c" || got[2].Label != "C" {
		t.Fatalf("unexpected third item: %#v", got[2])
	}
}

func TestResolveSelectionFieldConflict_SelectionAddMerge(t *testing.T) {
	baseSpec := &IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			String:        "Status",
			SelectionKind: "static",
			Selection: []IrFieldSelectionItem{
				{Value: "draft", Label: "Draft"},
				{Value: "done", Label: "Done"},
			},
		},
	}
	base := &IrField{Name: "Status", FieldType: "selection", FieldString: "Status"}
	if err := base.SetResolvedSpec(baseSpec); err != nil {
		t.Fatalf("set base spec: %v", err)
	}
	applySelectionLegacyColumns(base, baseSpec)

	childSpec := &IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd: []IrFieldSelectionItem{
				{Value: "cancel", Label: "Cancel"},
				{Value: "done", Label: "Finished"},
			},
		},
	}
	child := &IrField{Name: "Status", FieldType: "selection", OriginModelPath: "/child"}
	if err := child.SetResolvedSpec(childSpec); err != nil {
		t.Fatalf("set child spec: %v", err)
	}

	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("ResolveSelectionFieldConflict: %v", err)
	}
	if merged == nil {
		t.Fatal("expected merged field")
	}
	spec, err := merged.GetResolvedSpec()
	if err != nil || spec == nil {
		t.Fatalf("get merged spec: %v", err)
	}
	if spec.Structural.HasSelectionAdd || len(spec.Structural.SelectionAdd) != 0 {
		t.Fatalf("expected selectionAdd cleared after merge, got %#v", spec.Structural)
	}
	if spec.Structural.SelectionKind != "static" || len(spec.Structural.Selection) != 3 {
		t.Fatalf("unexpected merged selection: %#v", spec.Structural)
	}
	if spec.Structural.Selection[1].Label != "Finished" {
		t.Fatalf("expected done label override, got %#v", spec.Structural.Selection[1])
	}
	if spec.Structural.Selection[2].Value != "cancel" {
		t.Fatalf("expected cancel appended, got %#v", spec.Structural.Selection)
	}
	if merged.FieldString != "Status" {
		t.Fatalf("expected base string preserved, got %q", merged.FieldString)
	}
	if !strings.Contains(merged.Selection, `"cancel"`) {
		t.Fatalf("expected legacy Selection JSON to include cancel, got %s", merged.Selection)
	}
}

func TestResolveSelectionFieldConflict_FullSelectionReplaces(t *testing.T) {
	base := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	child := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "static", TsTypeAnnotation: "fork"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []IrFieldSelectionItem{{Value: "x", Label: "X"}},
		},
	})

	got, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != child {
		t.Fatalf("expected full selection child to replace base")
	}
}

func TestResolveSelectionFieldConflict_DynamicBaseRejected(t *testing.T) {
	base := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "dynamic", SelectionMethod: "Opts"}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	child := &IrField{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd:  true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	_, err := ResolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected dynamic base rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_BothSelectionAndAddRejected(t *testing.T) {
	base := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	child := &IrField{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd:  true,
			Selection:       []IrFieldSelectionItem{{Value: "x", Label: "X"}},
			SelectionAdd:    []IrFieldSelectionItem{{Value: "y", Label: "Y"}},
		},
	})
	_, err := ResolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "cannot combine") {
		t.Fatalf("expected combine rejection, got %v", err)
	}
}

func TestFieldHasSelectionAdd(t *testing.T) {
	field := &IrField{Name: "Status"}
	if FieldHasSelectionAdd(field) {
		t.Fatal("expected false without resolved spec")
	}
	spec := &IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{},
		},
	}
	raw, _ := json.Marshal(spec)
	field.ResolvedSpec = string(raw)
	if !FieldHasSelectionAdd(field) {
		t.Fatal("expected true for empty selectionAdd authoring")
	}
}
