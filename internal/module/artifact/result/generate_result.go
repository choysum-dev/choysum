package result

import "context"

type GeneratorResult struct {
	Name     string
	OutPaths []string
}

type Generator interface {
	Generate() ([]*GeneratorResult, error)
}

type GeneratorToTargets interface {
	GenerateToTargetsCtx(
		ctx context.Context,
		addonsProtoDir string,
		addonsWebDir string,
		addonsServiceDir string,
		distAppDir string,
	) ([]*GeneratorResult, error)
}
