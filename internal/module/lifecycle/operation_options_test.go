// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"testing"
)

func TestOperationOptionsContextAndPlanBuildOptions(t *testing.T) {
	if got := OperationOptionsFromContext(nil); got.WithDemo || got.SkipWebShell {
		t.Fatalf("nil ctx options = %#v", got)
	}
	if got := OperationOptionsFromContext(context.Background()); got.SkipWebShell {
		t.Fatalf("empty ctx SkipWebShell = %v", got.SkipWebShell)
	}

	ctx := WithOperationOptions(nil, OperationOptions{WithDemo: true, SkipWebShell: true})
	got := OperationOptionsFromContext(ctx)
	if !got.WithDemo || !got.SkipWebShell {
		t.Fatalf("stored options = %#v", got)
	}

	if opts := planBuildOptionsFromContext(context.Background()); opts != nil {
		t.Fatalf("expected nil plan opts, got %#v", opts)
	}
	opts := planBuildOptionsFromContext(ctx)
	if len(opts) != 1 || opts[0] == nil {
		t.Fatalf("expected one SkipWebShell build option, got %#v", opts)
	}
}
