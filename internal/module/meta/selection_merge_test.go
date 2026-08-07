// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"strings"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

func TestMergeSelectionByValue(t *testing.T) {
	base := []pkgmeta.FieldSelectionItem{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B"},
		{Value: "  ", Label: "skip"},
	}
	add := []pkgmeta.FieldSelectionItem{
		{Value: "c", Label: "C"},
		{Value: "b", Label: "B2"},
		{Value: "", Label: "empty"},
	}
	got := mergeSelectionByValue(base, add)
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
	ref := pkgmeta.NewTermReference("demo", "demo.status", "Done", "literal")
	base := []pkgmeta.FieldSelectionItem{
		{Value: "done", Label: "Done", LabelText: &ref},
	}
	got := mergeSelectionByValue(base, []pkgmeta.FieldSelectionItem{
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
	spec := &pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{
		Selection: []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
	}}
	if got := selectionItemsFromField(nil, spec); len(got) != 1 || got[0].Value != "a" {
		t.Fatalf("expected spec selection, got %#v", got)
	}
	if got := selectionItemsFromField(&pkgmeta.Field{Selection: ""}, &pkgmeta.FieldResolvedSpec{}); got != nil {
		t.Fatalf("expected nil for empty selection JSON, got %#v", got)
	}
	if got := selectionItemsFromField(&pkgmeta.Field{Selection: "{bad"}, &pkgmeta.FieldResolvedSpec{}); got != nil {
		t.Fatalf("expected nil for invalid JSON, got %#v", got)
	}
	got := selectionItemsFromField(&pkgmeta.Field{Selection: `[{"value":"x","label":"X"}]`}, &pkgmeta.FieldResolvedSpec{})
	if len(got) != 1 || got[0].Value != "x" {
		t.Fatalf("expected legacy JSON items, got %#v", got)
	}
}

func TestIsDynamicAndChildDeclaresFullSelection(t *testing.T) {
	if isDynamicSelectionField(nil, nil) {
		t.Fatal("expected false for empty inputs")
	}
	if !isDynamicSelectionField(nil, &pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{SelectionKind: "dynamic"}}) {
		t.Fatal("expected dynamic from spec kind")
	}
	if !isDynamicSelectionField(nil, &pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{SelectionMethod: "Opts"}}) {
		t.Fatal("expected dynamic from selection method")
	}
	if !isDynamicSelectionField(&pkgmeta.Field{SelectionKind: "dynamic"}, nil) {
		t.Fatal("expected dynamic from field kind")
	}
	if !isDynamicSelectionField(&pkgmeta.Field{SelectionMethod: "Opts", Selection: ""}, nil) {
		t.Fatal("expected dynamic from field method")
	}
	if childDeclaresFullSelection(nil) {
		t.Fatal("expected false for nil spec")
	}
	if !childDeclaresFullSelection(&pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{
		Selection: []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
	}}) {
		t.Fatal("expected true for static selection list")
	}
	if !childDeclaresFullSelection(&pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{SelectionKind: "dynamic"}}) {
		t.Fatal("expected true for dynamic kind")
	}
	if !childDeclaresFullSelection(&pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{SelectionMethod: "Opts"}}) {
		t.Fatal("expected true for selection method")
	}
	if childDeclaresFullSelection(&pkgmeta.FieldResolvedSpec{Structural: pkgmeta.FieldStructuralSpec{HasSelectionAdd: true}}) {
		t.Fatal("expected false for selectionAdd-only")
	}
}

func TestFieldHasSelectionAdd(t *testing.T) {
	if fieldHasSelectionAdd(nil) {
		t.Fatal("expected false for nil")
	}
	field := &pkgmeta.Field{Name: "Status"}
	if fieldHasSelectionAdd(field) {
		t.Fatal("expected false without resolved spec")
	}
	field.ResolvedSpec = "{bad"
	if fieldHasSelectionAdd(field) {
		t.Fatal("expected false for invalid resolved spec")
	}
	spec := &pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{},
		},
	}
	raw, _ := json.Marshal(spec)
	field.ResolvedSpec = string(raw)
	if !fieldHasSelectionAdd(field) {
		t.Fatal("expected true for empty selectionAdd authoring")
	}
}

func TestResolveSelectionFieldConflict_NilAndReplacePaths(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status"}
	child := &pkgmeta.Field{Name: "Status"}
	if got, err := resolveSelectionFieldConflict(base, nil); err != nil || got != base {
		t.Fatalf("nil child should return base, got %#v err=%v", got, err)
	}
	if got, err := resolveSelectionFieldConflict(nil, child); err != nil || got != child {
		t.Fatalf("nil base should return child, got %#v err=%v", got, err)
	}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{FieldName: "Status", Structural: pkgmeta.FieldStructuralSpec{Name: "Status"}})
	if got, err := resolveSelectionFieldConflict(base, child); err != nil || got != child {
		t.Fatalf("child without selectionAdd should replace, got %#v err=%v", got, err)
	}
	child.ResolvedSpec = "{bad"
	if _, err := resolveSelectionFieldConflict(base, child); err == nil {
		t.Fatal("expected child GetResolvedSpec error")
	}
}

func TestResolveSelectionFieldConflict_SelectionAddMerge(t *testing.T) {
	baseSpec := &pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			String:        "Status",
			SelectionKind: "static",
			Selection: []pkgmeta.FieldSelectionItem{
				{Value: "draft", Label: "Draft"},
				{Value: "done", Label: "Done"},
			},
		},
	}
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", FieldString: "Status"}
	if err := base.SetResolvedSpec(baseSpec); err != nil {
		t.Fatalf("set base spec: %v", err)
	}
	applySelectionLegacyColumns(base, baseSpec)

	stringText := pkgmeta.NewTermReference("demo", "demo.fields", "Status", "literal")
	helpText := pkgmeta.NewTermReference("demo", "demo.fields", "Help", "literal")
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
	childSpec := &pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			String:          "Status Label",
			StringText:      &stringText,
			Help:            "Help text",
			HelpText:        &helpText,
			HasSelectionAdd: true,
			SelectionAdd: []pkgmeta.FieldSelectionItem{
				{Value: "cancel", Label: "Cancel"},
				{Value: "done", Label: "Finished"},
			},
			Related:          &pkgmeta.FieldRelatedSpec{Path: "Parent.Status"},
			StorageHints:     &pkgmeta.FieldStructuralStorageHints{Required: &required, Indexed: &indexed, Size: &size, Precision: &precision, Scale: &scale},
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
		Behavior: pkgmeta.FieldBehaviorSpec{
			Compute:    &pkgmeta.FieldBehaviorComputeSpec{Method: "ComputeStatus", Deps: []string{"Name"}, Store: true},
			SqlCompute: &pkgmeta.FieldBehaviorSqlComputeSpec{Method: "SqlStatus", CtxType: "Ctx", ReturnType: "string"},
			Inverse:    &pkgmeta.FieldBehaviorMethodRef{Method: "InverseStatus"},
			Search:     &pkgmeta.FieldBehaviorMethodRef{Method: "SearchStatus"},
		},
	}
	child := &pkgmeta.Field{
		Name:             "Status",
		FieldType:        "selection",
		OriginModelPath:  "/child",
		TsTypeAnnotation: "string",
		TsTypeReference:  "Status",
	}
	if err := child.SetResolvedSpec(childSpec); err != nil {
		t.Fatalf("set child spec: %v", err)
	}

	merged, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("resolveSelectionFieldConflict: %v", err)
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
	base := &pkgmeta.Field{
		Name:          "Status",
		FieldType:     "selection",
		SelectionKind: "static",
		Selection:     `[{"value":"a","label":"A"}]`,
	}
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection", TsTypeAnnotation: "string"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "b", Label: "B"}},
		},
	})
	merged, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("resolveSelectionFieldConflict: %v", err)
	}
	spec, err := merged.GetResolvedSpec()
	if err != nil || spec == nil || len(spec.Structural.Selection) != 2 {
		t.Fatalf("expected merge from legacy JSON base, got %#v err=%v", spec, err)
	}
}

func TestResolveSelectionFieldConflict_OverlaysChildStorageHints(t *testing.T) {
	required := true
	indexed := true
	size := 64
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static", Size: 32, Indexed: true}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
			StorageHints:  &pkgmeta.FieldStructuralStorageHints{Size: intPtr(32), Indexed: &indexed, Precision: intPtr(10)},
		},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "b", Label: "B"}},
			// Partial overlay: only Required + Size; Indexed/Precision must stay from base.
			StorageHints: &pkgmeta.FieldStructuralStorageHints{Required: &required, Size: &size},
			Readonly:     boolPtr(true),
		},
	})

	merged, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("resolveSelectionFieldConflict: %v", err)
	}
	spec, err := merged.GetResolvedSpec()
	if err != nil || spec == nil || spec.Structural.StorageHints == nil {
		t.Fatalf("expected merged storage hints, got %#v err=%v", spec, err)
	}
	hints := spec.Structural.StorageHints
	if hints.Required == nil || !*hints.Required || hints.Size == nil || *hints.Size != 64 {
		t.Fatalf("expected child Required/Size overlay, got %#v", hints)
	}
	if hints.Indexed == nil || !*hints.Indexed || hints.Precision == nil || *hints.Precision != 10 {
		t.Fatalf("expected inherited Indexed/Precision preserved, got %#v", hints)
	}
	if !merged.NotNull || merged.Size != 64 || !merged.Indexed || !merged.IsReadonly {
		t.Fatalf("expected legacy columns from merged hints/readonly, got notNull=%v size=%d indexed=%v readonly=%v", merged.NotNull, merged.Size, merged.Indexed, merged.IsReadonly)
	}
}

func TestMergeStorageHints_PartialOverlayAndNilGuards(t *testing.T) {
	if got := mergeStorageHints(nil, nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	base := &pkgmeta.FieldStructuralStorageHints{Size: intPtr(16), Indexed: boolPtr(true)}
	if got := mergeStorageHints(base, nil); got != base {
		t.Fatalf("nil child should return base pointer, got %#v", got)
	}
	indexName := "idx_status"
	uniqueIndex := "uniq_status"
	def := "draft"
	child := &pkgmeta.FieldStructuralStorageHints{
		Required:           boolPtr(true),
		Index:              &indexName,
		PrimaryKey:         boolPtr(false),
		Unique:             boolPtr(true),
		UniqueIndex:        &uniqueIndex,
		UniqueIndexEnabled: boolPtr(true),
		Default:            &def,
		Scale:              intPtr(2),
	}
	got := mergeStorageHints(base, child)
	if got == nil || got == base || got == child {
		t.Fatalf("expected cloned merged hints, got %#v", got)
	}
	if got.Size == nil || *got.Size != 16 || got.Indexed == nil || !*got.Indexed {
		t.Fatalf("expected base Size/Indexed preserved, got %#v", got)
	}
	if got.Required == nil || !*got.Required || got.Scale == nil || *got.Scale != 2 {
		t.Fatalf("expected child Required/Scale overlay, got %#v", got)
	}
	if got.Index == nil || *got.Index != indexName || got.UniqueIndex == nil || *got.UniqueIndex != uniqueIndex {
		t.Fatalf("expected index overlays, got %#v", got)
	}
	if got.PrimaryKey == nil || *got.PrimaryKey || got.Unique == nil || !*got.Unique || got.UniqueIndexEnabled == nil || !*got.UniqueIndexEnabled {
		t.Fatalf("expected boolean overlays, got %#v", got)
	}
	if got.Default == nil || *got.Default != def {
		t.Fatalf("expected default overlay, got %#v", got)
	}
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

func TestResolveSelectionFieldConflict_FullSelectionReplaces(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static", TsTypeAnnotation: "fork"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []pkgmeta.FieldSelectionItem{{Value: "x", Label: "X"}},
		},
	})

	got, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != child {
		t.Fatalf("expected full selection child to replace base")
	}
}

func TestResolveSelectionFieldConflict_DynamicBaseRejected(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "dynamic", SelectionMethod: "Opts"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			SelectionKind:   "dynamic",
			SelectionMethod: "Opts",
		},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	_, err := resolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected dynamic base rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_MissingStaticBaseRejected(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", FieldType: "varchar"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName:  "Status",
		Structural: pkgmeta.FieldStructuralSpec{Name: "Status", FieldType: "varchar"},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	_, err := resolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "inherited static selection") {
		t.Fatalf("expected missing static base rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_BothSelectionAndAddRejected(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: "selection"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			FieldType:       "selection",
			HasSelectionAdd: true,
			Selection:       []pkgmeta.FieldSelectionItem{{Value: "x", Label: "X"}},
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "y", Label: "Y"}},
		},
	})
	_, err := resolveSelectionFieldConflict(base, child)
	if err == nil || !strings.Contains(err.Error(), "cannot combine") {
		t.Fatalf("expected combine rejection, got %v", err)
	}
}

func TestResolveSelectionFieldConflict_BaseResolvedSpecError(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", ResolvedSpec: "{bad"}
	child := &pkgmeta.Field{Name: "Status"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	if _, err := resolveSelectionFieldConflict(base, child); err == nil {
		t.Fatal("expected base GetResolvedSpec error")
	}
}

func TestOverlayAndApplySelectionLegacyColumnsNilGuards(t *testing.T) {
	overlayStructuralSelectionAdd(nil, &pkgmeta.FieldStructuralSpec{})
	overlayStructuralSelectionAdd(&pkgmeta.FieldStructuralSpec{}, nil)
	overlayBehaviorSelectionAdd(nil, &pkgmeta.FieldBehaviorSpec{})
	overlayBehaviorSelectionAdd(&pkgmeta.FieldBehaviorSpec{}, nil)
	applySelectionLegacyColumns(nil, &pkgmeta.FieldResolvedSpec{})
	applySelectionLegacyColumns(&pkgmeta.Field{}, nil)

	field := &pkgmeta.Field{}
	applySelectionLegacyColumns(field, &pkgmeta.FieldResolvedSpec{
		Structural: pkgmeta.FieldStructuralSpec{SelectionKind: "static"},
	})
	if field.Selection != "" || field.SelectionKind != "static" {
		t.Fatalf("expected empty selection cleared, got %#v", field)
	}

	zero := 0
	applySelectionLegacyColumns(field, &pkgmeta.FieldResolvedSpec{
		Structural: pkgmeta.FieldStructuralSpec{
			Selection:      []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
			MaxUploadBytes: &zero,
			MaxWidth:       &zero,
			MaxHeight:      &zero,
			StorageHints:   &pkgmeta.FieldStructuralStorageHints{},
		},
		Behavior: pkgmeta.FieldBehaviorSpec{
			SqlCompute: &pkgmeta.FieldBehaviorSqlComputeSpec{Method: "SqlOnly"},
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
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
			Selection:     []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	child := &pkgmeta.Field{Name: "Status"}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "b", Label: "B"}},
		},
	})

	origSet := selectionSetResolved
	selectionSetResolved = func(field *pkgmeta.Field, spec *pkgmeta.FieldResolvedSpec) error {
		return fmtError("set resolved failed")
	}
	defer func() { selectionSetResolved = origSet }()
	if _, err := resolveSelectionFieldConflict(base, child); err == nil || !strings.Contains(err.Error(), "set resolved failed") {
		t.Fatalf("expected setResolved seam error, got %v", err)
	}

	selectionSetResolved = origSet
	origMarshal := selectionJSONMarshal
	selectionJSONMarshal = func(v any) ([]byte, error) {
		return nil, fmtError("marshal failed")
	}
	defer func() { selectionJSONMarshal = origMarshal }()
	merged, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("marshal failures should be ignored for legacy columns, got %v", err)
	}
	if merged == nil {
		t.Fatal("expected merged field")
	}
	ref := pkgmeta.NewTermReference("demo", "demo.fields", "Status", "literal")
	applySelectionLegacyColumns(merged, &pkgmeta.FieldResolvedSpec{
		Structural: pkgmeta.FieldStructuralSpec{
			Selection:  []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
			StringText: &ref,
			HelpText:   &ref,
		},
	})
}

type fmtError string

func (e fmtError) Error() string { return string(e) }

func TestResolveSelectionFieldConflict_EmptyStaticBaseAllowsAdd(t *testing.T) {
	base := &pkgmeta.Field{Name: "Status", FieldType: "selection", SelectionKind: "static"}
	_ = base.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:          "Status",
			FieldType:     "selection",
			SelectionKind: "static",
		},
	})
	child := &pkgmeta.Field{Name: "Status", FieldType: ""}
	_ = child.SetResolvedSpec(&pkgmeta.FieldResolvedSpec{
		FieldName: "Status",
		Structural: pkgmeta.FieldStructuralSpec{
			Name:            "Status",
			HasSelectionAdd: true,
			SelectionAdd:    []pkgmeta.FieldSelectionItem{{Value: "a", Label: "A"}},
		},
	})
	merged, err := resolveSelectionFieldConflict(base, child)
	if err != nil {
		t.Fatalf("empty static base should accept selectionAdd, got %v", err)
	}
	spec, _ := merged.GetResolvedSpec()
	if len(spec.Structural.Selection) != 1 || spec.Structural.Selection[0].Value != "a" {
		t.Fatalf("unexpected merged selection: %#v", spec.Structural.Selection)
	}
}
