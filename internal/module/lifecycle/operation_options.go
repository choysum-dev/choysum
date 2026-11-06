package lifecycle

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/plan"
)

type OperationOptions struct {
	WithDemo bool
	// SkipWebShell disables planner auto-include of the web SPA shell when a
	// module declares entryPoints.web (CLI --no-web).
	SkipWebShell bool
	// WebShellProfile selects optional shell extras after ensureWebShell
	// (minimal|standard|full). Empty means planner default (standard).
	WebShellProfile plan.WebShellProfile
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
	op := OperationOptionsFromContext(ctx)
	var opts []plan.BuildOption
	if op.SkipWebShell {
		opts = append(opts, plan.WithSkipWebShell(true))
	}
	if profile := plan.WebShellProfile(strings.TrimSpace(string(op.WebShellProfile))); profile != "" {
		opts = append(opts, plan.WithWebShellProfile(profile))
	}
	if len(opts) == 0 {
		return nil
	}
	return opts
}
