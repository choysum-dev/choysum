// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package reload

import "sync/atomic"

type hookFunc func() error

var current atomic.Value

func Register(fn func() error) {
	if fn == nil {
		current.Store(hookFunc(nil))
		return
	}
	current.Store(hookFunc(fn))
}

func Trigger() error {
	if v := current.Load(); v != nil {
		if fn, ok := v.(hookFunc); ok && fn != nil {
			return fn()
		}
	}
	return nil
}
