package lifecycle

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type InstallRequest struct {
	Name         string
	WithDemo     bool
	SkipWebShell bool
}

type UpgradeRequest struct {
	Input        string
	WithDemo     bool
	SkipWebShell bool
}

type UninstallRequest struct {
	Name string
}

type Service interface {
	Install(ctx context.Context, req InstallRequest) error
	Upgrade(ctx context.Context, req UpgradeRequest) error
	Uninstall(ctx context.Context, req UninstallRequest) error
}

type service struct {
	manager *ModuleManager
}

func NewService(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, opts ...Option) Service {
	return &service{manager: NewModuleManager(runtimeScope, jsExecutor, opts...)}
}

func (s *service) Install(ctx context.Context, req InstallRequest) error {
	ctx = WithOperationOptions(ctx, OperationOptions{WithDemo: req.WithDemo, SkipWebShell: req.SkipWebShell})
	return s.manager.Install(ctx, strings.TrimSpace(req.Name))
}

func (s *service) Upgrade(ctx context.Context, req UpgradeRequest) error {
	ctx = WithOperationOptions(ctx, OperationOptions{WithDemo: req.WithDemo, SkipWebShell: req.SkipWebShell})
	return s.manager.Upgrade(ctx, strings.TrimSpace(req.Input))
}

func (s *service) Uninstall(ctx context.Context, req UninstallRequest) error {
	return s.manager.Uninstall(ctx, strings.TrimSpace(req.Name))
}
