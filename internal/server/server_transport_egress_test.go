// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

type fakeGRPCStopper struct {
	gracefulStarted chan struct{}
	stopCalls       atomic.Int32
	releaseGraceful chan struct{}
	hangGraceful    bool
	hangStop        bool
}

func (f *fakeGRPCStopper) GracefulStop() {
	if f.gracefulStarted != nil {
		close(f.gracefulStarted)
	}
	if f.hangGraceful {
		if f.releaseGraceful != nil {
			<-f.releaseGraceful
		} else {
			select {}
		}
		return
	}
}

func (f *fakeGRPCStopper) Stop() {
	f.stopCalls.Add(1)
	if f.hangStop {
		select {}
	}
	if f.releaseGraceful != nil {
		select {
		case <-f.releaseGraceful:
		default:
			close(f.releaseGraceful)
		}
	}
}

func TestStopGRPCServerWithTimeout_GracefulCompletes(t *testing.T) {
	stopper := &fakeGRPCStopper{}
	start := time.Now()
	stopGRPCServerWithTimeout(stopper, 200*time.Millisecond, 200*time.Millisecond)
	if elapsed := time.Since(start); elapsed >= 150*time.Millisecond {
		t.Fatalf("expected quick graceful stop, took %v", elapsed)
	}
	if stopper.stopCalls.Load() != 0 {
		t.Fatalf("Stop() calls = %d, want 0", stopper.stopCalls.Load())
	}
}

func TestStopGRPCServerWithTimeout_ForceStopUnblocksGraceful(t *testing.T) {
	stopper := &fakeGRPCStopper{
		gracefulStarted: make(chan struct{}),
		releaseGraceful: make(chan struct{}),
		hangGraceful:    true,
	}
	done := make(chan struct{})
	go func() {
		stopGRPCServerWithTimeout(stopper, 30*time.Millisecond, time.Second)
		close(done)
	}()
	select {
	case <-stopper.gracefulStarted:
	case <-time.After(time.Second):
		t.Fatal("GracefulStop was not started")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("force Stop did not unblock graceful wait")
	}
	if stopper.stopCalls.Load() != 1 {
		t.Fatalf("Stop() calls = %d, want 1", stopper.stopCalls.Load())
	}
}

func TestStopGRPCServerWithTimeout_AbandonsWhenBothHang(t *testing.T) {
	stopper := &fakeGRPCStopper{
		hangGraceful: true,
		hangStop:     true,
	}
	start := time.Now()
	stopGRPCServerWithTimeout(stopper, 20*time.Millisecond, 20*time.Millisecond)
	elapsed := time.Since(start)
	if elapsed < 35*time.Millisecond {
		t.Fatalf("expected both timeouts to elapse, took %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("abandon path hung too long: %v", elapsed)
	}
	if stopper.stopCalls.Load() != 1 {
		t.Fatalf("Stop() calls = %d, want 1", stopper.stopCalls.Load())
	}
}

func TestStopGRPCServerWithTimeout_NilServer(t *testing.T) {
	stopGRPCServerWithTimeout(nil, time.Millisecond, time.Millisecond)
}

func TestStopGRPCTransport_NonReloadUsesTimeoutHelper(t *testing.T) {
	// Real idle grpc.Server GracefulStop returns immediately; covers the
	// non-reload call path that delegates to stopGRPCServerWithTimeout.
	srv := &GRPCWebServer{server: grpc.NewServer()}
	srv.stopGRPCTransport(false)
	if srv.server != nil {
		t.Fatal("expected server to be cleared")
	}
}
