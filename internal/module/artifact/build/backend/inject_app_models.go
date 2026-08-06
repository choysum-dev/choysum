// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func (b *ModuleBuilder) injectAppModels(prebuildResult *module.BuildResult) error {
	if b == nil {
		return nil
	}
	sess := b.ensureInjectSession()
	fx, err := injectappmodel.InjectAppModels(sess, module.ParserResults(prebuildResult))
	if err != nil {
		b.releaseInjectSchedules()
		return err
	}
	b.applyInjectEffects(fx)
	return nil
}

func (b *ModuleBuilder) supersedeInjectAppModels() error {
	if b == nil {
		return nil
	}
	return injectappmodel.SupersedeInjectAppModels(b.ensureInjectSession())
}

func (b *ModuleBuilder) validateInjectAppModels(buildResult *module.BuildResult) error {
	if b == nil {
		return nil
	}
	return injectappmodel.ValidateInjectAppModels(b.ensureInjectSession(), module.ParserResults(buildResult))
}

// BundleInjectAppModels registers inject sources for all Specs (multi-app bundles).
func (b *ModuleBuilder) BundleInjectAppModels(modules []*meta.Module) error {
	if b == nil {
		return nil
	}
	fx, err := injectappmodel.BundleInjectAppModels(b.ensureInjectSession(), modules)
	if err != nil {
		return err
	}
	b.applyInjectEffects(fx)
	return nil
}
