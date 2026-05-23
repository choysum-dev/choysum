// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcwebplugin

import (
	"fmt"
	"time"

	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/protobuf/types/pluginpb"
)

type tsGenerator interface {
	Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error)
}

type grpcWebPlugin struct {
	runtimeScope scope.Scope
	generator    tsGenerator
}

func (p *grpcWebPlugin) Name() string {
	return "grpc-web"
}

func (p *grpcWebPlugin) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	now := time.Now()
	defer func() {
		p.runtimeScope.Logger().Debug("grpc-web typescript generation completed", "duration_ms", time.Since(now).Milliseconds())
	}()

	response, err := p.generator.Generate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %v", err)
	}

	return response, nil
}

func NewGrpcWebPluginWithGenerator(runtimeScope scope.Scope, generator tsGenerator) *grpcWebPlugin {
	if generator == nil {
		panic("grpc-web generator is nil")
	}

	return &grpcWebPlugin{
		runtimeScope: runtimeScope,
		generator:    generator,
	}
}

func NewGrpcWebPlugin(runtimeScope scope.Scope) *grpcWebPlugin {
	return NewGrpcWebPluginWithGenerator(runtimeScope, gots.NewGenerator())
}
