// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scope

import (
	"context"
	"log/slog"
)

type Scope interface {
	Run(fn func(scope Scope) error) error
	Session() *Session
	Transactor() Transactor
	WithContext(ctx context.Context) Scope
	Context() context.Context
	Logger() *slog.Logger
}

type FactoryInputCarrier interface {
	FactoryInput() FactoryInput
}

func FactoryInputFromScope(runtimeScope Scope) FactoryInput {
	if runtimeScope == nil {
		return nil
	}
	carrier, ok := runtimeScope.(FactoryInputCarrier)
	if !ok {
		return nil
	}
	return carrier.FactoryInput()
}
