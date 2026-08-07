// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import moduleresult "github.com/choysum-dev/choysum/internal/module/artifact/result"

// commitStubSplitBuilder is a shared SplitBuilder stub for package lifecycle tests.
// entrySeen/buildErr are exercised by ensureInjected codegen tests; persist* by
// installer commit tests.
type commitStubSplitBuilder struct {
	entrySeen    string
	buildErr     error
	persistCalls int
	persistErr   error
}

func (b *commitStubSplitBuilder) Build() (*moduleresult.BuildResult, error) {
	return &moduleresult.BuildResult{}, nil
}

func (b *commitStubSplitBuilder) BuildWithoutPersist() (*moduleresult.BuildResult, error) {
	if b.buildErr != nil {
		return nil, b.buildErr
	}
	return &moduleresult.BuildResult{}, nil
}

func (b *commitStubSplitBuilder) Persist(result *moduleresult.BuildResult) error {
	b.persistCalls++
	return b.persistErr
}
