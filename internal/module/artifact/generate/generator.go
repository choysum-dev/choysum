// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"sort"
	"strings"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type grpcGenerator struct {
	runtimeScope           scope.Scope
	module                 *meta.Module
	protobufGenerator      *protobufGenerator
	webGrpcGenerator       *webGrpcGenerator
	webServiceGenerator    *webServiceGenerator
	webApiStoreGenerator   *webApiStoreGenerator
	serviceClientGenerator *serviceClientGenerator
}

func (g *grpcGenerator) filterServices(services []*meta.Service) []*meta.Service {
	filtered := make([]*meta.Service, 0, len(services))
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

func (g *grpcGenerator) getApplication() (*meta.Application, error) {
	var application *meta.Application
	if result := g.runtimeScope.Session().Take(&application, g.module.ApplicationId); result.Error != nil {
		return nil, result.Error
	}

	// Effective projections have empty module_id (EDS-2+); scope by application name.
	var appModels []*meta.Model
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
		Where("meta_model.application = ?", application.Name).
		Where("meta_model.abstract = ?", false).
		Where("(meta_model.module_id IS NULL OR meta_model.module_id = '')").
		Order("meta_model.name ASC").
		Find(&appModels); result.Error != nil {
		return nil, result.Error
	}

	// Effective catalog is already E2-merged (one row per name). Keep merge helpers
	// for gold-standard unit tests only — do not re-merge in production codegen.
	uniqueModels := make([]*meta.Model, 0, len(appModels))
	for _, model := range appModels {
		if model == nil || model.Name == "" {
			continue
		}
		model.Services = g.filterServices(model.Services)
		uniqueModels = append(uniqueModels, model)
	}

	application.Models = uniqueModels
	return application, nil
}

func selectSameNameModelsInPrimaryExtensionChain(models []*meta.Model) []*meta.Model {
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
		return []*meta.Model{anchor}
	}

	byPath := make(map[string]*meta.Model, len(models))
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

	out := make([]*meta.Model, 0, len(models))
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

	// Anchor always has a non-empty Path and is marked connected above, so out is non-empty.
	return out
}

func mergeSameNameModelsByExtensionChain(models []*meta.Model) (*meta.Model, error) {
	return modmeta.MergeSameNameModelsByExtensionChain(models)
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
	modulesProtoDir string,
	modulesWebDir string,
	modulesServiceDir string,
	distAppDir string,
) ([]*module.GeneratorResult, error) {
	if g.protobufGenerator != nil {
		g.protobufGenerator.modulesProtoDir = modulesProtoDir
		g.protobufGenerator.distAppDir = distAppDir
	}
	if g.webGrpcGenerator != nil {
		g.webGrpcGenerator.modulesProtoDir = modulesProtoDir
		g.webGrpcGenerator.modulesWebDir = modulesWebDir
	}
	if g.webServiceGenerator != nil {
		g.webServiceGenerator.modulesWebDir = modulesWebDir
	}
	if g.webApiStoreGenerator != nil {
		g.webApiStoreGenerator.modulesWebDir = modulesWebDir
	}
	if g.serviceClientGenerator != nil {
		g.serviceClientGenerator.modulesProtoDir = modulesProtoDir
		g.serviceClientGenerator.modulesServiceDir = modulesServiceDir
	}
	return g.GenerateCtx(ctx)
}

func NewGrpcGenerator(runtimeScope scope.Scope, module *meta.Module) module.Generator {
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
