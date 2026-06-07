package origin

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/meta"
)

const (
	OriginTypeLocal    = "local"
	OriginTypeRegistry = "registry"
)

type Binding struct {
	ModuleName      string `json:"moduleName" yaml:"moduleName"`
	OriginType      string `json:"originType" yaml:"originType"`
	OriginRef       string `json:"originRef" yaml:"originRef"`
	ResolvedVersion string `json:"resolvedVersion,omitempty" yaml:"resolvedVersion,omitempty"`
	Integrity       string `json:"integrity,omitempty" yaml:"integrity,omitempty"`
	LocalPath       string `json:"localPath,omitempty" yaml:"localPath,omitempty"`
	UpdatedAt       string `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

type Service interface {
	Peek(ctx context.Context, input string) (*meta.IrModule, error)
	ResolveInstallModule(ctx context.Context, input string) (*meta.IrModule, error)
	Fetch(ctx context.Context, input string) (*meta.IrModule, error)
	Purge(ctx context.Context, moduleName string) error
}
