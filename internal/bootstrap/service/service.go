// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"net/http"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"google.golang.org/grpc"
)

// BootstrapService serves the workspace bootstrap gRPC and web endpoints.
type BootstrapService struct {
	runtimeScope    scope.Scope
	webHandler      http.Handler
	workspaceServer bootstrappb.WorkspaceServer
}

type bootstrapServiceOptions struct {
	switchModeFunc   func(ctx context.Context) error
	runtimeReadyFunc func(ctx context.Context) error
	lockerFactory    statepkg.LockerFactory
}

// BootstrapServiceOption configures BootstrapService construction.
type BootstrapServiceOption func(*bootstrapServiceOptions)

// WithSwitchModeFunc overrides the runtime mode switch used during bootstrap.
func WithSwitchModeFunc(fn func(ctx context.Context) error) BootstrapServiceOption {
	return func(opts *bootstrapServiceOptions) {
		opts.switchModeFunc = fn
	}
}

// WithRuntimeReadyFunc overrides the runtime readiness gate checked during bootstrap.
func WithRuntimeReadyFunc(fn func(ctx context.Context) error) BootstrapServiceOption {
	return func(opts *bootstrapServiceOptions) {
		opts.runtimeReadyFunc = fn
	}
}

// WithLockerFactory overrides the locker factory used to serialize bootstrap work.
func WithLockerFactory(factory statepkg.LockerFactory) BootstrapServiceOption {
	return func(opts *bootstrapServiceOptions) {
		opts.lockerFactory = factory
	}
}

// NewBootstrapService builds the bootstrap service with its web and gRPC handlers.
func NewBootstrapService(runtimeScope scope.Scope, opts ...BootstrapServiceOption) (*BootstrapService, error) {
	h, err := newBootstrapWebHandler(runtimeScope)
	if err != nil {
		return nil, err
	}

	serviceOpts := bootstrapServiceOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&serviceOpts)
		}
	}

	return &BootstrapService{
		runtimeScope:    runtimeScope,
		webHandler:      h,
		workspaceServer: newWorkspaceServer(runtimeScope, serviceOpts.switchModeFunc, serviceOpts.runtimeReadyFunc, serviceOpts.lockerFactory),
	}, nil
}

// Name returns the registration name of the bootstrap service.
func (s *BootstrapService) Name() string {
	return "bootstrap"
}

// ServiceDescs returns the gRPC service descriptors exposed by the bootstrap service.
func (s *BootstrapService) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	return []*grpc.ServiceDesc{&bootstrappb.Workspace_ServiceDesc}, nil
}

// RegisterGRPC registers the bootstrap workspace server on the provided gRPC server.
func (s *BootstrapService) RegisterGRPC(server *grpc.Server) error {
	bootstrappb.RegisterWorkspaceServer(server, s.workspaceServer)
	return nil
}

// ServiceScripts reports the JavaScript runtime scripts required by the bootstrap service.
func (s *BootstrapService) ServiceScripts() []*jsengine.JsScript {
	return nil
}

// WebHandlers returns the HTTP handlers that serve the bootstrap UI routes.
func (s *BootstrapService) WebHandlers() (map[string]http.Handler, error) {
	return map[string]http.Handler{
		"/":           http.RedirectHandler("/bootstrap/", http.StatusFound),
		"/bootstrap":  s.webHandler,
		"/bootstrap/": s.webHandler,
	}, nil
}

type workspaceServer struct {
	bootstrappb.UnimplementedWorkspaceServer
	coordinator  workspaceCoordinator
	runtimeScope scope.Scope
}

func newWorkspaceServer(runtimeScope scope.Scope, switchModeFunc func(ctx context.Context) error, runtimeReadyFunc func(ctx context.Context) error, lockerFactory statepkg.LockerFactory) *workspaceServer {
	c := newCoordinator(runtimeScope)
	if switchModeFunc != nil {
		c.switchMode = switchModeFunc
	}
	if runtimeReadyFunc != nil {
		c.validateRuntimeReady = runtimeReadyFunc
	}
	if lockerFactory != nil {
		c.lockerFactory = lockerFactory
	}

	return &workspaceServer{
		coordinator:  c,
		runtimeScope: runtimeScope,
	}
}
