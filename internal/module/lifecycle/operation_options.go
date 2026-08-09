package lifecycle

import (
	"context"

	"github.com/choysum-dev/choysum/internal/module/plan"
)

type OperationOptions struct {
	WithDemo bool
	// SkipWebShell disables planner auto-include of the web SPA shell when a
	// module declares entryPoints.web (CLI --no-web).
	SkipWebShell bool
}

type operationOptionsKey struct{}

func WithOperationOptions(ctx context.Context, opts OperationOptions) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, operationOptionsKey{}, opts)
}

func OperationOptionsFromContext(ctx context.Context) OperationOptions {
	if ctx == nil {
		return OperationOptions{}
	}
	if value := ctx.Value(operationOptionsKey{}); value != nil {
		if opts, ok := value.(OperationOptions); ok {
			return opts
		}
	}
	return OperationOptions{}
}

func planBuildOptionsFromContext(ctx context.Context) []plan.BuildOption {
	if OperationOptionsFromContext(ctx).SkipWebShell {
		return []plan.BuildOption{plan.WithSkipWebShell(true)}
	}
	return nil
}
