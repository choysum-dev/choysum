// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"sync"

	"github.com/choysum-dev/choysum/internal/import/plan"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Writer persists planned units for one profile.
type Writer interface {
	Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error
}

var (
	writerMu sync.RWMutex
	writers  = make(map[importpkg.Profile]Writer)
)

// RegisterWriter binds a profile to a writer implementation.
func RegisterWriter(profile importpkg.Profile, w Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	writers[profile] = w
}

// WriterFor returns the writer for profile.
func WriterFor(profile importpkg.Profile) (Writer, error) {
	writerMu.RLock()
	w, ok := writers[profile]
	writerMu.RUnlock()
	if !ok || w == nil {
		return nil, importpkg.ErrWriterNotRegistered
	}
	return w, nil
}

// ResetWritersForTest clears registered writers. Tests only.
func ResetWritersForTest() {
	writerMu.Lock()
	writers = make(map[importpkg.Profile]Writer)
	writerMu.Unlock()
}
