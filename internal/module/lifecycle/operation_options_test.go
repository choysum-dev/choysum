// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"testing"
)

func TestOperationOptionsContextAndPlanBuildOptions(t *testing.T) {
	t.Parallel()
	// Intentionally pass nil: production helpers must treat a missing Context as empty options.
	if got := OperationOptionsFromContext(nil); got.WithDemo || got.SkipWebShell {
		t.Fatalf("nil ctx options = %#v", got)
	}
	if got := OperationOptionsFromContext(context.Background()); got.SkipWebShell {
		t.Fatalf("empty ctx SkipWebShell = %v", got.SkipWebShell)
	}

	ctx := WithOperationOptions(context.Background(), OperationOptions{
		WithDemo:        true,
		SkipWebShell:    true,
		WebShellProfile: "minimal",
	})
	got := OperationOptionsFromContext(ctx)
	if !got.WithDemo || !got.SkipWebShell || got.WebShellProfile != "minimal" {
		t.Fatalf("stored options = %#v", got)
	}

	if opts := planBuildOptionsFromContext(context.Background()); opts != nil {
		t.Fatalf("expected nil plan opts, got %#v", opts)
	}
	opts := planBuildOptionsFromContext(ctx)
	if len(opts) != 2 || opts[0] == nil || opts[1] == nil {
		t.Fatalf("expected SkipWebShell + WebShellProfile build options, got %#v", opts)
	}
}
