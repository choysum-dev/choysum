// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	xfmt "golang.org/x/exp/errors/fmt"
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
	RunAs                    *string `json:"runAs,omitempty"`
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

	RelationInverseField     *string `json:"relationInverseField,omitempty"`
	RelationJoinModel        *string `json:"relationJoinModel,omitempty"`
	RelationJoinField        *string `json:"relationJoinField,omitempty"`
	RelationInverseJoinField *string `json:"relationInverseJoinField,omitempty"`

	// Selection is forwarded when available.
	Selection *string `json:"selection,omitempty"`

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

	if resolved.Resolved.RunAs.Value != nil {
		metadata.RunAs = toStringPtr(*resolved.Resolved.RunAs.Value)
	} else if resolved.Behavior.Compute != nil {
		metadata.RunAs = toStringPtr(resolved.Behavior.Compute.RunAs)
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

	// Pass through Selection when it is non-empty.
	if field.Selection != "" {
		metadata.Selection = &field.Selection
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

func (g *webApiStoreGenerator) generate(ctx context.Context, app *meta.IrApplication) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtimeOpts := runtimeOptionsFromScope(g.runtimeScope)
	var OutPaths []string
	if len(app.Models) == 0 {
		return nil, nil
	}

	// Preload BaseModel service names so the template can filter base CRUD services.
	baseServiceNames := map[string]bool{
		"Browse":         true,
		"BrowseMany":     true,
		"Search":         true,
		"Count":          true,
		"Create":         true,
		"CreateMany":     true,
		"Update":         true,
		"UpdateById":     true,
		"Delete":         true,
		"DeleteById":     true,
		"DefaultGet":     true,
		"FieldsGet":      true,
		"Onchange":       true,
		"ReadGroup":      true,
		"ReadGroupCount": true,
	}
	for _, m := range app.Models {
		if m.Name == "BaseModel" {
			for _, svc := range m.Services {
				baseServiceNames[svc.Name] = true
			}
			break
		}
	}

	funcMap := template.FuncMap{
		"ToLowerCase": strings.ToLower,
		"ToSnakeCase": strcase.ToSnake,
		// Drop the .ts extension.
		"ConvertPathNoExt": func(path string) string {
			p := strings.ReplaceAll(path, runtimeOpts.modulesPath, "@")
			return strings.TrimSuffix(p, ".ts")
		},
		// Check whether the service name comes from BaseModel.
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
