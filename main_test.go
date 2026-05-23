// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"testing"
)

type fakeCommander struct {
	execute func() error
}

func (f fakeCommander) Execute() error {
	if f.execute != nil {
		return f.execute()
	}
	return nil
}

func TestMainExecutesCommander(t *testing.T) {
	originalNewCommander := newCommander
	originalExit := exitFunc
	t.Cleanup(func() {
		newCommander = originalNewCommander
		exitFunc = originalExit
	})

	called := false
	exitCode := -1
	newCommander = func(ctx context.Context) interface{ Execute() error } {
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		return fakeCommander{execute: func() error {
			called = true
			return nil
		}}
	}
	exitFunc = func(code int) {
		exitCode = code
	}

	main()

	if !called {
		t.Fatal("expected commander Execute to be called")
	}
	if exitCode != -1 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
}

func TestMainExitsOnCommanderError(t *testing.T) {
	originalNewCommander := newCommander
	originalExit := exitFunc
	t.Cleanup(func() {
		newCommander = originalNewCommander
		exitFunc = originalExit
	})

	called := false
	exitCode := -1
	newCommander = func(context.Context) interface{ Execute() error } {
		return fakeCommander{execute: func() error {
			called = true
			return errors.New("boom")
		}}
	}
	exitFunc = func(code int) {
		exitCode = code
	}

	main()

	if !called {
		t.Fatal("expected commander Execute to be called")
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}
