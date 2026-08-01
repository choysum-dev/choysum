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
		{Value: "  ", Label: "skip"},
	}
	add := []IrFieldSelectionItem{
		{Value: "c", Label: "C"},
		{Value: "b", Label: "B2"},
		{Value: "", Label: "empty"},
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

func TestMergeSelectionByValue_PlainLabelClearsLabelText(t *testing.T) {
	ref := NewTermReference("demo", "demo.status", "Done", "literal")
	base := []IrFieldSelectionItem{
		{Value: "done", Label: "Done", LabelText: &ref},
	}
	got := MergeSelectionByValue(base, []IrFieldSelectionItem{
		{Value: "done", Label: "Finished"},
	})
	if len(got) != 1 || got[0].Label != "Finished" || got[0].LabelText != nil {
		t.Fatalf("expected plain override to clear LabelText, got %#v", got)
	}
}

func TestSelectionItemsFromFieldHelpers(t *testing.T) {
	if got := selectionItemsFromField(nil, nil); got != nil {
		t.Fatalf("expected nil for empty inputs, got %#v", got)
	}
	spec := &IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{
		Selection: []IrFieldSelectionItem{{Value: "a", Label: "A"}},
	}}
	if got := selectionItemsFromField(nil, spec); len(got) != 1 || got[0].Value != "a" {
		t.Fatalf("expected spec selection, got %#v", got)
	}
	if got := selectionItemsFromField(&IrField{Selection: ""}, &IrFieldResolvedSpec{}); got != nil {
		t.Fatalf("expected nil for empty selection JSON, got %#v", got)
	}
	if got := selectionItemsFromField(&IrField{Selection: "{bad"}, &IrFieldResolvedSpec{}); got != nil {
		t.Fatalf("expected nil for invalid JSON, got %#v", got)
	}
	got := selectionItemsFromField(&IrField{Selection: `[{"value":"x","label":"X"}]`}, &IrFieldResolvedSpec{})
	if len(got) != 1 || got[0].Value != "x" {
		t.Fatalf("expected legacy JSON items, got %#v", got)
	}
}

func TestIsDynamicAndChildDeclaresFullSelection(t *testing.T) {
	if isDynamicSelectionField(nil, nil) {
		t.Fatal("expected false for empty inputs")
	}
	if !isDynamicSelectionField(nil, &IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{SelectionKind: "dynamic"}}) {
		t.Fatal("expected dynamic from spec kind")
	}
	if !isDynamicSelectionField(nil, &IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{SelectionMethod: "Opts"}}) {
		t.Fatal("expected dynamic from selection method")
	}
	if !isDynamicSelectionField(&IrField{SelectionKind: "dynamic"}, nil) {
		t.Fatal("expected dynamic from field kind")
	}
	if !isDynamicSelectionField(&IrField{SelectionMethod: "Opts", Selection: ""}, nil) {
		t.Fatal("expected dynamic from field method")
	}
	if childDeclaresFullSelection(nil) {
		t.Fatal("expected false for nil spec")
	}
	if !childDeclaresFullSelection(&IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{
		Selection: []IrFieldSelectionItem{{Value: "a", Label: "A"}},
	}}) {
		t.Fatal("expected true for static selection list")
	}
	if !childDeclaresFullSelection(&IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{SelectionKind: "dynamic"}}) {
		t.Fatal("expected true for dynamic kind")
	}
	if !childDeclaresFullSelection(&IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{SelectionMethod: "Opts"}}) {
		t.Fatal("expected true for selection method")
	}
	if childDeclaresFullSelection(&IrFieldResolvedSpec{Structural: IrFieldStructuralSpec{HasSelectionAdd: true}}) {
		t.Fatal("expected false for selectionAdd-only")
	}
}

func TestFieldHasSelectionAdd(t *testing.T) {
	if FieldHasSelectionAdd(nil) {
		t.Fatal("expected false for nil")
	}
	field := &IrField{Name: "Status"}
	if FieldHasSelectionAdd(field) {
		t.Fatal("expected false without resolved spec")
	}
	field.ResolvedSpec = "{bad"
	if FieldHasSelectionAdd(field) {
		t.Fatal("expected false for invalid resolved spec")
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

func TestResolveSelectionFieldConflict_NilAndReplacePaths(t *testing.T) {
	base := &IrField{Name: "Status"}
	child := &IrField{Name: "Status"}
	if got, err := ResolveSelectionFieldConflict(base, nil); err != nil || got != base {
		t.Fatalf("nil child should return base, got %#v err=%v", got, err)
	}
	if got, err := ResolveSelectionFieldConflict(nil, child); err != nil || got != child {
		t.Fatalf("nil base should return child, got %#v err=%v", got, err)
	}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{FieldName: "Status", Structural: IrFieldStructuralSpec{Name: "Status"}})
	if got, err := ResolveSelectionFieldConflict(base, child); err != nil || got != child {
		t.Fatalf("child without selectionAdd should replace, got %#v err=%v", got, err)
	}
	child.ResolvedSpec = "{bad"
	if _, err := ResolveSelectionFieldConflict(base, child); err == nil {
		t.Fatal("expected child GetResolvedSpec error")
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

	stringText := NewTermReference("demo", "demo.fields", "Status", "literal")
	helpText := NewTermReference("demo", "demo.fields", "Help", "literal")
	required := true
	indexed := true
	size := 64
	precision := 10
	scale := 2
	upload := 1024
	width := 100
	height := 80
	translate := true
	companyDependent := true
	copyFlag := false
	readonly := true
	childSpec := &IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			String:          "Status Label",
			StringText:      &stringText,
			Help:            "Help text",
			HelpText:        &helpText,
			HasSelectionAdd: true,
			SelectionAdd: []IrFieldSelectionItem{
				{Value: "cancel", Label: "Cancel"},
				{Value: "done", Label: "Finished"},
			},
			Related:          &IrFieldRelatedSpec{Path: "Parent.Status"},
			StorageHints:     &IrFieldStructuralStorageHints{Required: &required, Indexed: &indexed, Size: &size, Precision: &precision, Scale: &scale},
			ColumnType:       "varchar",
			CheckConstraint:  "status <> ''",
			Translate:        &translate,
			CompanyDependent: &companyDependent,
			Copy:             &copyFlag,
			Readonly:         &readonly,
			MaxUploadBytes:   &upload,
			MaxWidth:         &width,
			MaxHeight:        &height,
		},
		Behavior: IrFieldBehaviorSpec{
			Compute:    &IrFieldBehaviorComputeSpec{Method: "ComputeStatus", Deps: []string{"Name"}, Store: true},
			SqlCompute: &IrFieldBehaviorSqlComputeSpec{Method: "SqlStatus", CtxType: "Ctx", ReturnType: "string"},
			Inverse:    &IrFieldBehaviorMethodRef{Method: "InverseStatus"},
			Search:     &IrFieldBehaviorMethodRef{Method: "SearchStatus"},
		},
	}
	child := &IrField{
		Name:             "Status",
		FieldType:        "selection",
		OriginModelPath:  "/child",
		TsTypeAnnotation: "string",
		TsTypeReference:  "Status",
	}
	if err := child.SetResolvedSpec(childSpec); err != nil {
		t.Fatalf("set child spec: %v", err)
	}

	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("ResolveSelectionFieldConflict: %v", err)
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
	if spec.Structural.String != "Status Label" || spec.Structural.Help != "Help text" {
		t.Fatalf("expected string/help overlay, got %#v", spec.Structural)
	}
	if spec.Structural.Related == nil || spec.Structural.Related.Path != "Parent.Status" {
		t.Fatalf("expected related overlay, got %#v", spec.Structural.Related)
	}
	if spec.Behavior.Compute == nil || spec.Behavior.SqlCompute == nil || spec.Behavior.Inverse == nil || spec.Behavior.Search == nil {
		t.Fatalf("expected behavior overlay, got %#v", spec.Behavior)
	}
	if !merged.NotNull || !merged.Indexed || merged.Size != 64 || merged.Precision != 10 || merged.Scale != 2 {
		t.Fatalf("unexpected storage legacy columns: %#v", merged)
	}
	if !merged.IsReadonly || merged.MaxUploadBytes != 1024 || merged.MaxWidth != 100 || merged.MaxHeight != 80 {
		t.Fatalf("unexpected readonly/upload legacy columns: %#v", merged)
	}
	if merged.TsTypeAnnotation != "string" || merged.TsTypeReference != "Status" {
		t.Fatalf("expected child type annotations, got %#v", merged)
	}
	if !strings.Contains(merged.Selection, `"cancel"`) {
		t.Fatalf("expected legacy Selection JSON to include cancel, got %s", merged.Selection)
	}
}

func TestResolveSelectionFieldConflict_LegacyBaseWithoutResolvedSpec(t *testing.T) {
	base := &IrField{
		Name:          "Status",
		FieldType:     "selection",
		SelectionKind: "static",
		Selection:     `[{"value":"a","label":"A"}]`,
	}
	child := &IrField{Name: "Status", FieldType: "selection", TsTypeAnnotation: "string"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "b", Label: "B"}},
		},
	})
	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("ResolveSelectionFieldConflict: %v", err)
	}
	spec, err := merged.GetResolvedSpec()
	if err != nil || spec == nil || len(spec.Structural.Selection) != 2 {
		t.Fatalf("expected merge from legacy JSON base, got %#v err=%v", spec, err)
	}
}

func TestResolveSelectionFieldConflict_OverlaysChildStorageHints(t *testing.T) {
	required := true
	size := 64
	base := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "static", Size: 32}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []IrFieldSelectionItem{{Value: "a", Label: "A"}},
			StorageHints:  &IrFieldStructuralStorageHints{Size: intPtr(32)},
		},
	})
	child := &IrField{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "b", Label: "B"}},
			StorageHints:    &IrFieldStructuralStorageHints{Required: &required, Size: &size},
			Readonly:        boolPtr(true),
		},
	})

	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("ResolveSelectionFieldConflict: %v", err)
	}
	if !merged.NotNull || merged.Size != 64 || !merged.IsReadonly {
		t.Fatalf("expected legacy columns from child hints/readonly, got notNull=%v size=%d readonly=%v", merged.NotNull, merged.Size, merged.IsReadonly)
	}
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

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
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	_, err := ResolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected dynamic base rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_MissingStaticBaseRejected(t *testing.T) {
	base := &IrField{Name: "Status", FieldType: "varchar"}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName:  "Status",
		Structural: IrFieldStructuralSpec{Name: "Status", FieldType: "varchar"},
	})
	child := &IrField{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	_, err := ResolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected missing static base rejection, got %v", err)
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
			HasSelectionAdd: true,
			Selection:       []IrFieldSelectionItem{{Value: "x", Label: "X"}},
			SelectionAdd:    []IrFieldSelectionItem{{Value: "y", Label: "Y"}},
		},
	})
	_, err := ResolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "cannot combine") {
		t.Fatalf("expected combine rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_BaseResolvedSpecError(t *testing.T) {
	base := &IrField{Name: "Status", ResolvedSpec: "{bad"}
	child := &IrField{Name: "Status"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	if _, err := ResolveSelectionFieldConflict(base, child); err == nil {
		t.Fatal("expected base GetResolvedSpec error")
	}
}

func TestOverlayAndApplySelectionLegacyColumnsNilGuards(t *testing.T) {
	overlayStructuralSelectionAdd(nil, &IrFieldStructuralSpec{})
	overlayStructuralSelectionAdd(&IrFieldStructuralSpec{}, nil)
	overlayBehaviorSelectionAdd(nil, &IrFieldBehaviorSpec{})
	overlayBehaviorSelectionAdd(&IrFieldBehaviorSpec{}, nil)
	applySelectionLegacyColumns(nil, &IrFieldResolvedSpec{})
	applySelectionLegacyColumns(&IrField{}, nil)

	field := &IrField{}
	applySelectionLegacyColumns(field, &IrFieldResolvedSpec{
		Structural: IrFieldStructuralSpec{SelectionKind: "static"},
	})
	if field.Selection != "" || field.SelectionKind != "static" {
		t.Fatalf("expected empty selection cleared, got %#v", field)
	}

	zero := 0
	applySelectionLegacyColumns(field, &IrFieldResolvedSpec{
		Structural: IrFieldStructuralSpec{
			Selection:      []IrFieldSelectionItem{{Value: "a", Label: "A"}},
			MaxUploadBytes: &zero,
			MaxWidth:       &zero,
			MaxHeight:      &zero,
			StorageHints:   &IrFieldStructuralStorageHints{},
		},
		Behavior: IrFieldBehaviorSpec{
			SqlCompute: &IrFieldBehaviorSqlComputeSpec{Method: "SqlOnly"},
		},
	})
	if !field.IsReadonly {
		t.Fatal("expected sqlCompute to mark readonly")
	}
	if field.MaxUploadBytes != 0 || field.MaxWidth != 0 || field.MaxHeight != 0 {
		t.Fatalf("zero caps should stay cleared, got %#v", field)
	}
}

func TestResolveSelectionFieldConflict_DefensiveErrorSeams(t *testing.T) {
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
	child := &IrField{Name: "Status"}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "b", Label: "B"}},
		},
	})

	origSet := selectionSetResolved
	selectionSetResolved = func(field *IrField, spec *IrFieldResolvedSpec) error {
		return fmtError("set resolved failed")
	}
	defer func() { selectionSetResolved = origSet }()
	if _, err := ResolveSelectionFieldConflict(base, child); err == nil || !strings.Contains(err.Error(), "set resolved failed") {
		t.Fatalf("expected setResolved seam error, got %v", err)
	}

	selectionSetResolved = origSet
	origMarshal := selectionJSONMarshal
	selectionJSONMarshal = func(v any) ([]byte, error) {
		return nil, fmtError("marshal failed")
	}
	defer func() { selectionJSONMarshal = origMarshal }()
	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("marshal failures should be ignored for legacy columns, got %v", err)
	}
	if merged == nil {
		t.Fatal("expected merged field")
	}
	ref := NewTermReference("demo", "demo.fields", "Status", "literal")
	applySelectionLegacyColumns(merged, &IrFieldResolvedSpec{
		Structural: IrFieldStructuralSpec{
			Selection:  []IrFieldSelectionItem{{Value: "a", Label: "A"}},
			StringText: &ref,
			HelpText:   &ref,
		},
	})
}

type fmtError string

func (e fmtError) Error() string { return string(e) }

func TestResolveSelectionFieldConflict_EmptyStaticBaseAllowsAdd(t *testing.T) {
	base := &IrField{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
		},
	})
	child := &IrField{Name: "Status", FieldType: ""}
	_ = child.SetResolvedSpec(&IrFieldResolvedSpec{
		FieldName: "Status",
		Structural: IrFieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []IrFieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	merged, err := ResolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("empty static base should accept selectionAdd, got %v", err)
	}
	spec, _ := merged.GetResolvedSpec()
	if len(spec.Structural.Selection) != 1 || spec.Structural.Selection[0].Value != "a" {
		t.Fatalf("unexpected merged selection: %#v", spec.Structural.Selection)
	}
}
