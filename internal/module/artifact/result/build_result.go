package result

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/evanw/esbuild/pkg/api"
)

type BuildResult struct {
	Module        *meta.IrModule
	EsbuildResult *api.BuildResult
}

type Builder interface {
	Build() (*BuildResult, error)
}

// SplitBuilder compiles without writing metadata, then persists in a later step.
type SplitBuilder interface {
	Builder
	BuildWithoutPersist() (*BuildResult, error)
	Persist(result *BuildResult) error
}

type Bundler interface {
	Bundle() (*BuildResult, error)
}

type BundlerToDir interface {
	BundleToDirCtx(ctx context.Context, distAppDir string) (*BuildResult, error)
}

type BuilderToDir interface {
	BuildToDirCtx(ctx context.Context, distDir string) (*BuildResult, error)
}
