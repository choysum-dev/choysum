// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"log/slog"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// spikeBuildScope is a minimal scope for NewCompilerExecutor (vuesfc / vueplugin).
type spikeBuildScope struct {
	ctx context.Context
	cfg *config.Config
}

func (s *spikeBuildScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *spikeBuildScope) Session() *scope.Session              { return nil }
func (s *spikeBuildScope) Transactor() scope.Transactor         { return nil }
func (s *spikeBuildScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}
func (s *spikeBuildScope) Context() context.Context { return s.ctx }
func (s *spikeBuildScope) Logger() *slog.Logger     { return slog.Default() }
func (s *spikeBuildScope) FactoryInput() scope.FactoryInput {
	if s.cfg == nil {
		return nil
	}
	return &spikeFactoryInput{cfg: s.cfg}
}

type spikeFactoryInput struct{ cfg *config.Config }

func (i *spikeFactoryInput) Environment() string                  { return "" }
func (i *spikeFactoryInput) ModulesPath() string                  { return "" }
func (i *spikeFactoryInput) DistPath() string                     { return "" }
func (i *spikeFactoryInput) TmpPath() string                      { return "" }
func (i *spikeFactoryInput) DefaultChoysumPath() string           { return "" }
func (i *spikeFactoryInput) ConfigPath() string                   { return "" }
func (i *spikeFactoryInput) ESMUpstreamURL() string               { return "" }
func (i *spikeFactoryInput) NpmRegistryURL() string               { return "" }
func (i *spikeFactoryInput) ModuleCatalogIndexURL() string        { return "" }
func (i *spikeFactoryInput) CompileConfig() *config.CompileConfig { return nil }
func (i *spikeFactoryInput) AuthConfig() *config.AuthConfig       { return nil }
func (i *spikeFactoryInput) TaskConfig() *config.TaskConfig       { return nil }
func (i *spikeFactoryInput) LogConfig() *config.LogConfig         { return nil }
func (i *spikeFactoryInput) ServerConfig() *config.ServerConfig   { return i.cfg.Server }
