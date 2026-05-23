// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package reload

import (
	"errors"
	"testing"
)

func TestTriggerWithoutRegisteredHookReturnsNil(t *testing.T) {
	Register(nil)
	t.Cleanup(func() { Register(nil) })

	if err := Trigger(); err != nil {
		t.Fatalf("Trigger() error = %v, want nil", err)
	}
}

func TestRegisterAndTriggerHook(t *testing.T) {
	t.Cleanup(func() { Register(nil) })

	called := 0
	Register(func() error {
		called++
		return nil
	})

	if err := Trigger(); err != nil {
		t.Fatalf("Trigger() error = %v, want nil", err)
	}
	if called != 1 {
		t.Fatalf("hook called = %d, want 1", called)
	}
}

func TestTriggerReturnsHookError(t *testing.T) {
	t.Cleanup(func() { Register(nil) })

	wantErr := errors.New("reload failed")
	Register(func() error { return wantErr })

	if err := Trigger(); !errors.Is(err, wantErr) {
		t.Fatalf("Trigger() error = %v, want %v", err, wantErr)
	}
}
