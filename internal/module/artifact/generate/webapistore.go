// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

//go:embed webapistore.ts.tpl
var webApiStoreTplStr string

//go:embed webapistore.index.ts.tpl
var webApiStoreIndexTplStr string

// FieldMetadata contains client-facing field metadata.
type FieldMetadata struct {
	Id                       *string `json:"id,omitempty"`
	Name                     string  `json:"name"`
	FieldType                string  `json:"fieldType"`
	TypeAnnotation           string  `json:"typeAnnotation"`
	StorageKind              *string `json:"storageKind,omitempty"`
	ShouldCreateColumn       *bool   `json:"shouldCreateColumn,omitempty"`
	ResolvedColumnType       *string `json:"resolvedColumnType,omitempty"`
	ReasonCode               *string `json:"reasonCode,omitempty"`
	ComputedKind             *string `json:"computedKind,omitempty"`
	RelatedPath              *string `json:"relatedPath,omitempty"`
	RelatedStore             *bool   `json:"relatedStore,omitempty"`
	Searchable               *bool   `json:"searchable,omitempty"`
	RelationModel            *string `json:"relationModel,omitempty"`
	RelationFilter           *string `json:"relationFilter,omitempty"`
	RelationModelParentField *string `json:"relationModelParentField,omitempty"`
	NotNull                  *bool   `json:"notNull,omitempty"`
	Size                     *int    `json:"size,omitempty"`
	Precision                *int    `json:"precision,omitempty"`
	Scale                    *int    `json:"scale,omitempty"`
	Round                    *string `json:"round,omitempty"`
	IsReadonly               *bool   `json:"isReadonly,omitempty"`
	Indexed                  *bool   `json:"indexed,omitempty"`
	Translate                *bool   `json:"translate,omitempty"`
	Copy                     *bool   `json:"copy,omitempty"`

	RelationInverseField     *string `json:"relationInverseField,omitempty"`
	RelationJoinModel        *string `json:"relationJoinModel,omitempty"`
	RelationJoinField        *string `json:"relationJoinField,omitempty"`
	RelationInverseJoinField *string `json:"relationInverseJoinField,omitempty"`

	// Selection is forwarded when available.
	Selection *string `json:"selection,omitempty"`
	// SelectionKind is "static" or "dynamic" (P3).
	SelectionKind *string `json:"selectionKind,omitempty"`

	// String is a JS string literal expression (already quoted) for the field title msgid.
	String *string `json:"string,omitempty"`
	// StringText is a JSON object literal for the field title TermReference.
	StringText *string `json:"stringText,omitempty"`

	// ScaleField points to the dynamic decimal scale source.
	ScaleField *string `json:"scaleField,omitempty"`
}

func toStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func applyResolvedFieldContract(metadata *FieldMetadata, field *meta.IrField) {
	if metadata == nil || field == nil {
		return
	}

	resolved, err := field.GetResolvedSpec()
	if err != nil || resolved == nil {
		return
	}

	metadata.StorageKind = toStringPtr(resolved.Migration.StorageKind)
	shouldCreateColumn := resolved.Migration.ShouldCreateColumn
	metadata.ShouldCreateColumn = &shouldCreateColumn
	metadata.ResolvedColumnType = toStringPtr(resolved.Migration.ResolvedColumnType)
	metadata.ReasonCode = toStringPtr(resolved.Migration.ReasonCode)

	computedKind := ""
	switch {
	case resolved.Behavior.SqlCompute != nil:
		computedKind = "sql"
	case resolved.Behavior.Compute != nil:
		computedKind = "runtime"
	case resolved.Structural.Related != nil:
		computedKind = "related"
	}
	metadata.ComputedKind = toStringPtr(computedKind)

	if related := resolved.Structural.Related; related != nil {
		metadata.RelatedPath = toStringPtr(related.Path)
		relatedStore := related.Store
		metadata.RelatedStore = &relatedStore
	}

	if resolved.Resolved.Searchable.Value != nil {
		searchable := *resolved.Resolved.Searchable.Value
		metadata.Searchable = &searchable
	} else if resolved.Behavior.Search != nil {
		searchable := true
		metadata.Searchable = &searchable
	}

	if resolved.Structural.Translate != nil && *resolved.Structural.Translate {
		t := true
		metadata.Translate = &t
		if metadata.Size == nil && resolved.Structural.StorageHints != nil && resolved.Structural.StorageHints.Size != nil && *resolved.Structural.StorageHints.Size > 0 {
			size := *resolved.Structural.StorageHints.Size
			metadata.Size = &size
		}
	}
	if resolved.Structural.Copy != nil && !*resolved.Structural.Copy {
		f := false
		metadata.Copy = &f
	}
}

type webApiStoreGenerator struct {
	runtimeScope scope.Scope
	module       *meta.IrModule

	// Optional override for pipeline-managed staging.
	modulesWebDir string
}

// convertFieldToMetadata converts ChoysumMetaField into FieldMetadata.
func convertFieldToMetadata(field *meta.IrField) FieldMetadata {
	metadata := FieldMetadata{
		Name:           field.Name,
		FieldType:      field.FieldType,
		TypeAnnotation: field.TsTypeAnnotation,
	}

	applyResolvedFieldContract(&metadata, field)

	// Pass through the ID when it is valid.
	if field.Id.Valid && field.Id.String != "" {
		s := field.Id.String
		metadata.Id = &s
	}

	// Only include non-empty values in the metadata.
	if field.RelationModel != "" {
		metadata.RelationModel = &field.RelationModel
	}
	if field.RelationFilter != "" {
		metadata.RelationFilter = &field.RelationFilter
	}
	// Pass through RelationModelParentField.
	if s := field.RelationModelParentField; s != "" {
		metadata.RelationModelParentField = &s
	}
	if field.NotNull {
		metadata.NotNull = &field.NotNull
	}
	if field.Size > 0 {
		metadata.Size = &field.Size
	}
	if field.Precision > 0 {
		metadata.Precision = &field.Precision
	}
	if field.Scale > 0 {
		metadata.Scale = &field.Scale
	}
	// Pass through scaleField.
	if field.ScaleField != "" {
		metadata.ScaleField = &field.ScaleField
	}
	if field.IsReadonly {
		metadata.IsReadonly = &field.IsReadonly
	}
	if field.Indexed {
		metadata.Indexed = &field.Indexed
	}

	// Pass through relationship details.
	if s := field.RelationInverseField; s != "" {
		metadata.RelationInverseField = &s
	}
	if s := field.RelationJoinModel; s != "" {
		metadata.RelationJoinModel = &s
	}
	if s := field.RelationJoinField; s != "" {
		metadata.RelationJoinField = &s
	}
	if s := field.RelationInverseJoinField; s != "" {
		metadata.RelationInverseJoinField = &s
	}

	// Pass through Selection when it is non-empty (strip legacy labelText wire keys).
	// Prefer ResolvedSpec so _lt labels stay as msgid strings even if legacy
	// field.Selection was overwritten with decorator source text.
	// Dynamic selection fields omit the inline array (T3.1).
	kind := strings.TrimSpace(field.SelectionKind)
	selectionJSON := strings.TrimSpace(field.Selection)
	if resolved, err := field.GetResolvedSpec(); err == nil && resolved != nil {
		if k := strings.TrimSpace(resolved.Structural.SelectionKind); k != "" {
			kind = k
		}
		if kind != "dynamic" && len(resolved.Structural.Selection) > 0 {
			items := make([]map[string]any, 0, len(resolved.Structural.Selection))
			for _, item := range resolved.Structural.Selection {
				clean := map[string]any{
					"value": item.Value,
					"label": item.Label,
				}
				items = append(items, clean)
			}
			if encoded, err := json.Marshal(items); err == nil {
				selectionJSON = string(encoded)
			}
		}
	}
	if kind == "" && selectionJSON != "" {
		kind = "static"
	}
	if kind != "" {
		metadata.SelectionKind = &kind
	}
	if kind != "dynamic" && selectionJSON != "" {
		stripped := stripSelectionLabelTextJSON(selectionJSON)
		metadata.Selection = &stripped
	}

	if s := strings.TrimSpace(field.FieldString); s != "" {
		quoted := strconv.Quote(s)
		metadata.String = &quoted
	}
	if strings.TrimSpace(field.StringText) != "" {
		metadata.StringText = &field.StringText
	}

	// Pass through Round only when it is set.
	if field.Round != nil {
		metadata.Round = field.Round
	}

	return metadata
}

// stripSelectionLabelTextJSON removes legacy selection labelText keys from IR JSON
// before emitting static web store metadata (D5·D15 hard cut).
func stripSelectionLabelTextJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return raw
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		clean := make(map[string]any, len(item))
		for k, v := range item {
			if k == "labelText" {
				continue
			}
			clean[k] = v
		}
		out = append(out, clean)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return string(encoded)
}

// RelationFieldInfo describes a relationship field.
type RelationFieldInfo struct {
	Name          string
	RelationModel string
	IsArray       bool
}

// analyzeRelationFields collects and deduplicates relationship fields.
func analyzeRelationFields(model *meta.IrModel) ([]RelationFieldInfo, []string) {
	var relationFields []RelationFieldInfo
	uniqueModels := make(map[string]bool)
	var importModels []string

	for _, field := range model.Fields {
		if field.RelationModel != "" && (field.FieldType == "ManyToOne" || field.FieldType == "OneToMany" || field.FieldType == "ManyToMany") {
			relationFields = append(relationFields, RelationFieldInfo{
				Name:          field.Name,
				RelationModel: field.RelationModel,
				IsArray:       strings.Contains(field.TsTypeAnnotation, "[]"),
			})

			// Import only non-self-referential models once.
			if field.RelationModel != model.Name && !uniqueModels[field.RelationModel] {
				uniqueModels[field.RelationModel] = true
				importModels = append(importModels, field.RelationModel)
			}
		}
	}

	return relationFields, importModels
}

// resolveBaseServiceNames loads conventional service names from the abstract BaseModel
// IrModel (by BaseModelModuleSpec path). These names are filtered out of generated
// XxxStore interfaces because they are already declared on the hand-written WebModelStore.
// Do not maintain a parallel hardcoded name list here.
func resolveBaseServiceNames(runtimeScope scope.Scope) (map[string]bool, error) {
	path, _ := meta.BaseModelModuleSpec(runtimeScope)
	return resolveBaseServiceNamesAtPath(runtimeScope, path)
}

func resolveBaseServiceNamesAtPath(runtimeScope scope.Scope, path string) (map[string]bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, xfmt.Errorf("base model module path is empty")
	}
	path = esbplugins.NormalizePath(path)
	pathNoExt := strings.TrimSuffix(path, ".ts")
	pathWithExt := pathNoExt + ".ts"

	var model meta.IrModel
	result := runtimeScope.Session().
		Preload("Services", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Where("(path = ? OR path = ?) AND abstract = ?", pathNoExt, pathWithExt, true).
		Order("id DESC").
		Take(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, xfmt.Errorf("abstract BaseModel not found at path %s (or %s)", pathNoExt, pathWithExt)
		}
		return nil, xfmt.Errorf("load abstract BaseModel by path %s: %w", pathWithExt, result.Error)
	}

	names := conventionalBaseServiceNames(model.Services)
	if len(names) == 0 {
		return nil, xfmt.Errorf("abstract BaseModel at path %s has no conventional services", model.Path)
	}
	return names, nil
}

func conventionalBaseServiceNames(services []*meta.IrService) map[string]bool {
	names := make(map[string]bool)
	for _, svc := range services {
		if svc == nil {
			continue
		}
		if !meta.IsConventionalModelService(svc.AccessibilityModifier, svc.IsStatic, svc.Name) {
			continue
		}
		names[svc.Name] = true
	}
	return names
}

func (g *webApiStoreGenerator) generate(ctx context.Context, app *meta.IrApplication) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	var OutPaths []string
	if len(app.Models) == 0 {
		return nil, nil
	}

	baseServiceNames, err := resolveBaseServiceNames(g.runtimeScope)
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{
		"ToLowerCase": strings.ToLower,
		"ToSnakeCase": strcase.ToSnake,
		// Drop the .ts extension.
		"ConvertPathNoExt": func(path string) string {
			p := strings.ReplaceAll(path, runtimeOpts.modulesPath, "@")
			return strings.TrimSuffix(p, ".ts")
		},
		// True when name is a BaseModel conventional service (already on WebModelStore).
		"IsBaseService": func(name string) bool {
			return baseServiceNames[name]
		},
		"Contains": strings.Contains,
	}

	modulesWebDir := g.modulesWebDir
	if modulesWebDir == "" {
		_, webDir, _, err := WorkspaceGeneratedAPITargets(runtimeOpts.modulesPath, g.module.ApplicationStr, runtimeOpts.defaultChoysumPath)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace generated api targets: %w", err)
		}
		modulesWebDir = webDir
	}

	// Create the <web>/stores directory.
	outDir, _ := filepath.Abs(filepath.Join(modulesWebDir, "stores"))

	writeTo := func(dir string) error {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		// Generate a standalone store file for each model.
		for _, model := range app.Models {
			// Convert fields into metadata.
			var fieldsMetadata []FieldMetadata
			for _, field := range model.Fields {
				fieldsMetadata = append(fieldsMetadata, convertFieldToMetadata(field))
			}

			// Analyze relationship fields with dedupe and self-reference handling.
			relationFields, importModels := analyzeRelationFields(model)

			// Parse the template.
			tpl, err := template.New(model.Name).Funcs(funcMap).Parse(webApiStoreTplStr)
			if err != nil {
				return err
			}

			// Build the template data.
			data := struct {
				App            string
				Model          *meta.IrModel
				FieldsMetadata []FieldMetadata
				RelationFields []RelationFieldInfo
				ImportModels   []string
			}{
				App:            app.Name,
				Model:          model,
				FieldsMetadata: fieldsMetadata,
				RelationFields: relationFields,
				ImportModels:   importModels,
			}

			// Execute the template.
			buf := new(bytes.Buffer)
			if err := tpl.Execute(buf, data); err != nil {
				return xfmt.Errorf("error executing Store template for %s: %w", model.Name, err)
			}

			// Write the file.
			storeFileStage := filepath.Join(dir, strcase.ToSnake(model.Name)+".ts")
			if err := os.WriteFile(storeFileStage, buf.Bytes(), 0644); err != nil {
				return err
			}
			OutPaths = append(OutPaths, filepath.Join(outDir, strcase.ToSnake(model.Name)+".ts"))
		}

		// Generate the index file.
		indexTpl, err := template.New("store-index").Funcs(funcMap).Parse(webApiStoreIndexTplStr)
		if err != nil {
			return err
		}

		indexBuf := new(bytes.Buffer)
		if err := indexTpl.Execute(indexBuf, app); err != nil {
			return xfmt.Errorf("error executing Store index template: %w", err)
		}

		indexStage := filepath.Join(dir, "index.ts")
		if err := os.WriteFile(indexStage, indexBuf.Bytes(), 0644); err != nil {
			return err
		}
		OutPaths = append(OutPaths, filepath.Join(outDir, "index.ts"))
		return nil
	}

	if g.modulesWebDir != "" {
		if err := writeTo(outDir); err != nil {
			return nil, err
		}
	} else {
		if err := staging.WithStagingDir(ctx, outDir, func(stagingDir string) error {
			return writeTo(stagingDir)
		}); err != nil {
			return nil, err
		}
	}

	return []*module.GeneratorResult{
		{
			Name:     "webapistore",
			OutPaths: OutPaths,
		}}, nil
}

func NewWebApiStoreGenerator(runtimeScope scope.Scope, module *meta.IrModule) *webApiStoreGenerator {
	return &webApiStoreGenerator{
		runtimeScope: runtimeScope,
		module:       module,
	}
}
