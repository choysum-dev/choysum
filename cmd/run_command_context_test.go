// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/server"
	"github.com/choysum-dev/choysum/pkg/server/defaultserver"
)

func TestRunCommandTreatsContextCanceledAsGracefulStop(t *testing.T) {
	tests := []struct {
		name     string
		serveErr error
	}{
		{name: "direct context canceled", serveErr: context.Canceled},
		{name: "wrapped context canceled", serveErr: fmt.Errorf("wrapped: %w", context.Canceled)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, _, _ := writeTempInitializedRunConfig(t, false)
			stub := &runCommandStubServer{serveErr: tt.serveErr}

			originalFactory := runServerFactory
			runServerFactory = func(runtimeScope scope.Scope, _ ...defaultserver.Option) server.Server {
				if runtimeScope == nil {
					t.Fatal("expected non-nil runtime scope")
				}
				return stub
			}
			t.Cleanup(func() {
				runServerFactory = originalFactory
			})

			cmd := newRunCmd()
			cmd.Flags().String("config", "", "")
			if err := cmd.Flags().Set("config", configPath); err != nil {
				t.Fatalf("set config flag: %v", err)
			}

			output := captureStderr(t, func() {
				cmd.Run(cmd, []string{"web"})
			})

			if !stub.serveCalled {
				t.Fatal("expected server Serve to be called")
			}
			if stub.serveCtx == nil {
				t.Fatal("expected non-nil serve context")
			}
			if len(stub.serveServices) != 1 || stub.serveServices[0] != "web" {
				t.Fatalf("unexpected services passed to Serve: %#v", stub.serveServices)
			}
			if strings.Contains(output, "server starting; NEXT: open") {
				t.Fatalf("did not expect CLI startup hint, got %q", output)
			}
			if strings.Contains(output, "ERROR: server exited unexpectedly") {
				t.Fatalf("expected no error block for context cancellation, got %q", output)
			}
		})
	}
}

type runCommandStubServer struct {
	serveErr      error
	serveCalled   bool
	serveCtx      context.Context
	serveServices []string
}

func (s *runCommandStubServer) Serve(ctx context.Context, services ...string) error {
	s.serveCalled = true
	s.serveCtx = ctx
	s.serveServices = append([]string{}, services...)
	return s.serveErr
}
