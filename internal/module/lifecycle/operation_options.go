package lifecycle

import "context"

type OperationOptions struct {
	WithDemo bool
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
