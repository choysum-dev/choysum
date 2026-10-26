// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestStartHTTPServer_BaseContextCancelsInFlightRequests(t *testing.T) {
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()

	started := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	handle, err := StartHTTPServer(handler, HTTPServerOptions{
		Address:              "127.0.0.1:0",
		SuppressListeningLog: true,
		BaseContext:          baseCtx,
	})
	if err != nil {
		t.Fatalf("StartHTTPServer() error = %v", err)
	}
	defer func() {
		_ = handle.Server.Close()
	}()

	addr := handle.Listener.Addr().String()
	errCh := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		errCh <- nil
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}

	cancelBase()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("client request error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete after BaseContext cancel")
	}
}
