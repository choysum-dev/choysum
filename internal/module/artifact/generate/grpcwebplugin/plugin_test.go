// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcwebplugin

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/protobuf/types/pluginpb"
)

type stubScope struct{}

func (e *stubScope) Run(fn func(scope.Scope) error) error        { return fn(e) }
func (e *stubScope) Transactor() scope.Transactor                { return scopetest.NewPassthroughTransactor(e) }
func (e *stubScope) Session() *scope.Session                     { return nil }
func (e *stubScope) WithContext(ctx context.Context) scope.Scope { return e }
func (e *stubScope) Context() context.Context                    { return context.Background() }
func (e *stubScope) Logger() *slog.Logger                        { return slog.Default() }
func (e *stubScope) Config() *config.Config                      { return &config.Config{} }

func (e *stubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type stubGenerator struct {
	resp *pluginpb.CodeGeneratorResponse
	err  error
}

func (g *stubGenerator) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.resp, nil
}

func TestNewGrpcWebPlugin_DefaultsToGoGenerator(t *testing.T) {
	p := NewGrpcWebPlugin(&stubScope{})
	if p == nil {
		t.Fatalf("expected plugin instance")
	}
	if _, ok := p.generator.(*gots.Generator); !ok {
		t.Fatalf("expected default generator to be *gots.Generator, got %T", p.generator)
	}
}

func TestNewGrpcWebPluginWithGenerator_NilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for nil generator")
		}
	}()
	_ = NewGrpcWebPluginWithGenerator(&stubScope{}, nil)
}

func TestGrpcWebPluginGenerate_DelegatesToInjectedGenerator(t *testing.T) {
	expected := &pluginpb.CodeGeneratorResponse{}
	p := NewGrpcWebPluginWithGenerator(&stubScope{}, &stubGenerator{resp: expected})
	resp, err := p.Generate(&pluginpb.CodeGeneratorRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != expected {
		t.Fatalf("unexpected response pointer")
	}
}

func TestGrpcWebPluginGenerate_WrapsGeneratorError(t *testing.T) {
	p := NewGrpcWebPluginWithGenerator(&stubScope{}, &stubGenerator{err: errors.New("boom")})
	_, err := p.Generate(&pluginpb.CodeGeneratorRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to generate code") {
		t.Fatalf("unexpected error: %v", err)
	}
}
