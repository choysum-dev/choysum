// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"sort"
	"strings"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type grpcGenerator struct {
	runtimeScope           scope.Scope
	module                 *meta.IrModule
	protobufGenerator      *protobufGenerator
	webGrpcGenerator       *webGrpcGenerator
	webServiceGenerator    *webServiceGenerator
	webApiStoreGenerator   *webApiStoreGenerator
	serviceClientGenerator *serviceClientGenerator
}

func (g *grpcGenerator) filterServices(services []*meta.IrService) []*meta.IrService {
	filtered := make([]*meta.IrService, 0, len(services))
	for _, s := range services {
		if s == nil {
			continue
		}
		if !meta.IsConventionalModelService(s.AccessibilityModifier, s.IsStatic, s.Name) {
			continue
		}
		filtered = append(filtered, s)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered
}

func (g *grpcGenerator) getApplication() (*meta.IrApplication, error) {
	var application *meta.IrApplication
	if result := g.runtimeScope.Session().Take(&application, g.module.ApplicationId); result.Error != nil {
		return nil, result.Error
	}

	var appModels []*meta.IrModel
	if result := g.runtimeScope.Session().
		Preload("Services", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Services.Decorators", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Services.TypeParameters", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Preload("Services.Parameters", func(db *gorm.DB) *gorm.DB {
			return db.Where("name != ?", "this").Order("id ASC")
		}).
		Preload("Fields", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Joins("JOIN meta_ir_module ON meta_ir_model.module_id = meta_ir_module.id ").
		Where("meta_ir_module.application_id = ?", application.Id).
		Order("meta_ir_model.id ASC").
		Find(&appModels); result.Error != nil {
		return nil, result.Error
	}

	// Deduplicate models by name with a map.
	// First deduplicate by path, keeping the newest version for the same path,
	// then merge same-name models along the extension chain to avoid losing
	// extended fields in frontend fieldsMetadata.
	byPath := make(map[string]*meta.IrModel)
	var noPath []*meta.IrModel
	for _, model := range appModels {
		if model == nil || model.Name == "" {
			continue
		}
		if model.Path == "" {
			noPath = append(noPath, model)
			continue
		}
		existing, ok := byPath[model.Path]
		if !ok || existing == nil {
			byPath[model.Path] = model
			continue
		}
		if model.UpdatedAt.After(existing.UpdatedAt) || (model.UpdatedAt.Equal(existing.UpdatedAt) && model.Id.String > existing.Id.String) {
			byPath[model.Path] = model
		}
	}

	pathModels := make([]*meta.IrModel, 0, len(byPath))
	for _, m := range byPath {
		pathModels = append(pathModels, m)
	}

	canonicalModels := append(pathModels, noPath...)
	nameGroups := make(map[string][]*meta.IrModel)
	for _, model := range canonicalModels {
		if model == nil || model.Name == "" {
			continue
		}
		nameGroups[model.Name] = append(nameGroups[model.Name], model)
	}

	modelMap := make(map[string]*meta.IrModel, len(nameGroups))
	for name, group := range nameGroups {
		candidates := selectSameNameModelsInPrimaryExtensionChain(group)
		merged := mergeSameNameModelsByExtensionChain(candidates)
		if merged == nil {
			continue
		}
		modelMap[name] = merged
	}

	// Convert the deduplicated models back into a slice.
	uniqueModels := make([]*meta.IrModel, 0, len(modelMap))
	for _, model := range modelMap {
		model.Services = g.filterServices(model.Services)

		uniqueModels = append(uniqueModels, model)
	}

	// Sort by name.
	sort.Slice(uniqueModels, func(i, j int) bool {
		return uniqueModels[i].Name < uniqueModels[j].Name
	})

	application.Models = uniqueModels
	return application, nil
}

func selectSameNameModelsInPrimaryExtensionChain(models []*meta.IrModel) []*meta.IrModel {
	if len(models) <= 1 {
		return models
	}

	// Pick anchor deterministically: latest UpdatedAt, then lexicographically larger Id.
	anchor := models[0]
	for _, m := range models[1:] {
		if m == nil {
			continue
		}
		if anchor == nil || m.UpdatedAt.After(anchor.UpdatedAt) || (m.UpdatedAt.Equal(anchor.UpdatedAt) && m.Id.String > anchor.Id.String) {
			anchor = m
		}
	}
	if anchor == nil || anchor.Path == "" {
		return []*meta.IrModel{anchor}
	}

	byPath := make(map[string]*meta.IrModel, len(models))
	for _, m := range models {
		if m != nil && m.Path != "" {
			byPath[m.Path] = m
		}
	}

	neighbors := make(map[string][]string, len(byPath))
	for _, m := range byPath {
		if m == nil || m.Path == "" {
			continue
		}
		if parent := strings.TrimSpace(m.Extends); parent != "" {
			if _, ok := byPath[parent]; ok {
				neighbors[m.Path] = append(neighbors[m.Path], parent)
				neighbors[parent] = append(neighbors[parent], m.Path)
			}
		}
	}

	connected := make(map[string]bool, len(byPath))
	queue := []string{anchor.Path}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if connected[cur] {
			continue
		}
		connected[cur] = true
		for _, nxt := range neighbors[cur] {
			if !connected[nxt] {
				queue = append(queue, nxt)
			}
		}
	}

	out := make([]*meta.IrModel, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		if m.Path == "" {
			if m == anchor {
				out = append(out, m)
			}
			continue
		}
		if connected[m.Path] {
			out = append(out, m)
		}
	}

	if len(out) == 0 {
		return []*meta.IrModel{anchor}
	}
	return out
}

func mergeSameNameModelsByExtensionChain(models []*meta.IrModel) *meta.IrModel {
	if len(models) == 0 {
		return nil
	}
	if len(models) == 1 {
		return models[0]
	}

	byPath := make(map[string]*meta.IrModel, len(models))
	for _, m := range models {
		if m != nil && m.Path != "" {
			byPath[m.Path] = m
		}
	}

	type rankedModel struct {
		model *meta.IrModel
		depth int
	}

	depthMemo := make(map[string]int)
	visiting := make(map[string]bool)
	var depthOf func(*meta.IrModel) int
	depthOf = func(m *meta.IrModel) int {
		if m == nil {
			return 0
		}
		if m.Path == "" {
			return 0
		}
		if d, ok := depthMemo[m.Path]; ok {
			return d
		}
		if visiting[m.Path] {
			return 0
		}
		visiting[m.Path] = true
		defer delete(visiting, m.Path)

		parent := byPath[m.Extends]
		depth := 0
		if parent != nil {
			depth = depthOf(parent) + 1
		}
		depthMemo[m.Path] = depth
		return depth
	}

	ranked := make([]rankedModel, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		ranked = append(ranked, rankedModel{model: m, depth: depthOf(m)})
	}
	if len(ranked) == 0 {
		return nil
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		li := ranked[i]
		lj := ranked[j]
		if li.depth != lj.depth {
			return li.depth < lj.depth
		}
		if !li.model.UpdatedAt.Equal(lj.model.UpdatedAt) {
			return li.model.UpdatedAt.Before(lj.model.UpdatedAt)
		}
		return li.model.Id.String < lj.model.Id.String
	})

	fieldIndex := make(map[string]int)
	mergedFields := make([]*meta.IrField, 0)
	serviceIndex := make(map[string]int)
	mergedServices := make([]*meta.IrService, 0)

	for _, item := range ranked {
		m := item.model
		for _, f := range m.Fields {
			if f == nil || f.Name == "" {
				continue
			}
			if idx, ok := fieldIndex[f.Name]; ok {
				mergedFields[idx] = f
			} else {
				fieldIndex[f.Name] = len(mergedFields)
				mergedFields = append(mergedFields, f)
			}
		}

		for _, s := range m.Services {
			if s == nil || s.Name == "" {
				continue
			}
			if idx, ok := serviceIndex[s.Name]; ok {
				mergedServices[idx] = s
			} else {
				serviceIndex[s.Name] = len(mergedServices)
				mergedServices = append(mergedServices, s)
			}
		}
	}

	canonical := ranked[len(ranked)-1].model
	merged := *canonical
	merged.Fields = mergedFields
	merged.Services = mergedServices
	return &merged
}

func (g *grpcGenerator) Generate() ([]*module.GeneratorResult, error) {
	return g.GenerateCtx(context.Background())
}

func (g *grpcGenerator) GenerateCtx(ctx context.Context) ([]*module.GeneratorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !g.module.ApplicationId.Valid {
		return nil, nil
	}

	var results []*module.GeneratorResult
	app, err := g.getApplication()
	if err != nil {
		return nil, err
	}

	generatedTSConfigResult, err := ensureWorkspaceGeneratedTSConfig(g.runtimeScope)
	if err != nil {
		return nil, err
	}

	// generate protobuf
	protobufResults, err := g.protobufGenerator.generate(ctx, app)
	if err != nil {
		return nil, err
	}
	results = append(results, protobufResults...)

	// generate grpc-web
	bufResults, err := g.webGrpcGenerator.generate(ctx, results)
	if err != nil {
		return nil, err
	}
	results = append(results, bufResults...)

	// generate service
	serviceResults, err := g.webServiceGenerator.generate(app)
	if err != nil {
		return nil, err
	}
	results = append(results, serviceResults...)

	// generate service client (server-side)
	serviceClientResults, err := g.serviceClientGenerator.generate(ctx, app)
	if err != nil {
		return nil, err
	}
	results = append(results, serviceClientResults...)

	// // generate web API client
	// webApiResults, err := g.webApiGenerator.generate(app)
	// if err != nil {
	// 	return nil, err
	// }
	// results = append(results, webApiResults...)

	// generate web API store
	webApiStoreResults, err := g.webApiStoreGenerator.generate(ctx, app)
	if err != nil {
		return nil, err
	}
	results = append(results, webApiStoreResults...)

	if generatedTSConfigResult != nil {
		results = append(results, generatedTSConfigResult)
	}

	return results, nil
}

// GenerateToDirsCtx generates application artifacts into the provided directories.
// This is intended for pipeline-managed staging, where the pipeline will commit the
// staged directories atomically.
func (g *grpcGenerator) GenerateToTargetsCtx(
	ctx context.Context,
	addonsProtoDir string,
	addonsWebDir string,
	addonsServiceDir string,
	distAppDir string,
) ([]*module.GeneratorResult, error) {
	if g.protobufGenerator != nil {
		g.protobufGenerator.addonsProtoDir = addonsProtoDir
		g.protobufGenerator.distAppDir = distAppDir
	}
	if g.webGrpcGenerator != nil {
		g.webGrpcGenerator.addonsProtoDir = addonsProtoDir
		g.webGrpcGenerator.addonsWebDir = addonsWebDir
	}
	if g.webServiceGenerator != nil {
		g.webServiceGenerator.addonsWebDir = addonsWebDir
	}
	if g.webApiStoreGenerator != nil {
		g.webApiStoreGenerator.addonsWebDir = addonsWebDir
	}
	if g.serviceClientGenerator != nil {
		g.serviceClientGenerator.addonsProtoDir = addonsProtoDir
		g.serviceClientGenerator.addonsServiceDir = addonsServiceDir
	}
	return g.GenerateCtx(ctx)
}

func NewGrpcGenerator(runtimeScope scope.Scope, module *meta.IrModule) module.Generator {
	return &grpcGenerator{
		runtimeScope:           runtimeScope,
		module:                 module,
		protobufGenerator:      newProtobufGenerator(runtimeScope, module),
		webGrpcGenerator:       NewWebGrpcGenerator(runtimeScope, module),
		webServiceGenerator:    NewWebServiceGenerator(runtimeScope, module),
		webApiStoreGenerator:   NewWebApiStoreGenerator(runtimeScope, module),
		serviceClientGenerator: NewServiceClientGenerator(runtimeScope, module),
	}
}
