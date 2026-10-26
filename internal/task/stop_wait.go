// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"sync"
	"time"
)

// stopWaitTimeout bounds how long Dispatcher/Scheduler/GC Stop waits for
// in-flight goroutines. Matches the HTTP/gRPC drain budget so a hung worker
// cannot block process exit indefinitely.
var stopWaitTimeout = 2 * time.Second

// waitWaitGroup waits for wg up to timeout. It returns true when Wait returned
// before the deadline, false when the timeout fired (wg may still be active).
func waitWaitGroup(wg *sync.WaitGroup, timeout time.Duration) bool {
	if wg == nil {
		return true
	}
	if timeout <= 0 {
		wg.Wait()
		return true
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
