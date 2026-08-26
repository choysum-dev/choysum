// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"sync"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

var (
	readerMu sync.RWMutex
	readers  = make(map[exportpkg.Profile]Reader)
)

// Register binds a profile to a reader implementation.
func Register(profile exportpkg.Profile, r Reader) {
	readerMu.Lock()
	defer readerMu.Unlock()
	readers[profile] = r
}

// ReaderFor returns the reader for profile.
func ReaderFor(profile exportpkg.Profile) (Reader, error) {
	readerMu.RLock()
	r, ok := readers[profile]
	readerMu.RUnlock()
	if !ok || r == nil {
		return nil, exportpkg.ErrReaderNotRegistered
	}
	return r, nil
}

// ResetForTest clears registered readers. Tests only.
func ResetForTest() {
	readerMu.Lock()
	readers = make(map[exportpkg.Profile]Reader)
	readerMu.Unlock()
}
