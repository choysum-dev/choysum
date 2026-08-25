// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/pkg/config"
)

func TestServerRestartReplansServeTargetsFromUpdatedDist(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)
	writeRestartReplanManifest(t, runtimeScope.cfg.DistPath, []string{"auth"}, map[string]distmanifest.DistManifestApp{
		"auth": {},
	})
	seedRestartReplanAppProto(t, runtimeScope.cfg.DistPath, "auth")

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	srv.runState.setServeRequestArgs(nil)
	if err := srv.planServe(nil); err != nil {
		t.Fatalf("planServe(nil) error = %v", err)
	}
	assertRunStateTargets(t, srv, []string{"auth", "web"}, "before install")

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) error = %v", err)
	}

	writeRestartReplanManifest(t, runtimeScope.cfg.DistPath, []string{"auth", "partner"}, map[string]distmanifest.DistManifestApp{
		"auth":    {},
		"partner": {},
	})
	seedRestartReplanAppProto(t, runtimeScope.cfg.DistPath, "partner")

	if err := srv.Restart(); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	assertRunStateTargets(t, srv, []string{"auth", "partner", "web"}, "after install restart")
}

func TestServerRestartSurfacesReplanErrors(t *testing.T) {
	newPlannedServer := func(t *testing.T) (*GRPCWebServer, *noSessionServerScope) {
		t.Helper()
		runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
		runtimeScope.cfg.Auth.Enabled = false
		runtimeScope.cfg.DistPath = t.TempDir()
		runtimeScope.cfg.Compile.BundleMode = "bundle"
		assignEphemeralServerPort(t, runtimeScope.cfg)
		runtimeScope.cfg.Server.EnableGrpcWebProxy = false
		runtimeScope.cfg.Server.HotReload = false

		seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)
		writeRestartReplanManifest(t, runtimeScope.cfg.DistPath, []string{"auth"}, map[string]distmanifest.DistManifestApp{
			"auth": {},
		})
		seedRestartReplanAppProto(t, runtimeScope.cfg.DistPath, "auth")

		srv := NewServer(runtimeScope).(*GRPCWebServer)
		srv.runState.setServeRequestArgs(nil)
		if err := srv.planServe(nil); err != nil {
			t.Fatalf("planServe(nil) error = %v", err)
		}
		if err := srv.start(false); err != nil {
			t.Fatalf("start(false) error = %v", err)
		}
		return srv, runtimeScope
	}

	corruptManifest := func(t *testing.T, distRoot string) {
		t.Helper()
		path := filepath.Join(distRoot, distmanifest.DistManifestFileName)
		if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
			t.Fatalf("corrupt dist manifest: %v", err)
		}
	}

	srvRestart, envRestart := newPlannedServer(t)
	t.Cleanup(func() {
		if srvRestart.httpServer != nil || srvRestart.server != nil || srvRestart.listener != nil || srvRestart.grpcClientPool != nil {
			_ = srvRestart.stop(false)
		}
	})
	corruptManifest(t, envRestart.cfg.DistPath)
	err := srvRestart.Restart()
	if err == nil {
		t.Fatal("expected Restart() to fail when replan cannot parse dist manifest")
	}
	if strings.Contains(err.Error(), "Failed to start server") {
		t.Fatalf("Restart() error = %v, want direct replan failure without restart() wrapper", err)
	}
	if !strings.Contains(err.Error(), "parse dist manifest") {
		t.Fatalf("Restart() error = %v, want dist manifest parse failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestart), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "Restart() replan failure diagnostics")

	srvRestartHelper, envRestartHelper := newPlannedServer(t)
	t.Cleanup(func() {
		if srvRestartHelper.httpServer != nil || srvRestartHelper.server != nil || srvRestartHelper.listener != nil || srvRestartHelper.grpcClientPool != nil {
			_ = srvRestartHelper.stop(false)
		}
	})
	corruptManifest(t, envRestartHelper.cfg.DistPath)
	err = srvRestartHelper.restart()
	if err == nil {
		t.Fatal("expected restart() to fail when replan cannot parse dist manifest")
	}
	if !strings.Contains(err.Error(), "Failed to start server") {
		t.Fatalf("restart() error = %v, want wrapped start failure", err)
	}
	if !strings.Contains(err.Error(), "parse dist manifest") {
		t.Fatalf("restart() error = %v, want dist manifest parse failure", err)
	}
	assertRecoveryActionDiagnostics(t, recoveryDiagnosticsForTest(srvRestartHelper), recoveryActionRestart, recoveryActionDiagnostics{Attempts: 1, Failures: 1}, "restart() replan failure diagnostics")
}

func TestServerRestartPreservesExplicitServeTargets(t *testing.T) {
	runtimeScope := &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}
	runtimeScope.cfg.Auth.Enabled = false
	runtimeScope.cfg.DistPath = t.TempDir()
	runtimeScope.cfg.Compile.BundleMode = "bundle"
	assignEphemeralServerPort(t, runtimeScope.cfg)
	runtimeScope.cfg.Server.EnableGrpcWebProxy = false
	runtimeScope.cfg.Server.HotReload = false

	seedBundleModeWebReadyDist(t, runtimeScope.cfg.DistPath)
	writeRestartReplanManifest(t, runtimeScope.cfg.DistPath, []string{"auth", "partner"}, map[string]distmanifest.DistManifestApp{
		"auth":    {},
		"partner": {},
	})
	seedRestartReplanAppProto(t, runtimeScope.cfg.DistPath, "auth")
	seedRestartReplanAppProto(t, runtimeScope.cfg.DistPath, "partner")

	srv := NewServer(runtimeScope).(*GRPCWebServer)
	t.Cleanup(func() {
		if srv.httpServer != nil || srv.server != nil || srv.listener != nil || srv.grpcClientPool != nil {
			_ = srv.stop(false)
		}
	})

	explicit := []string{"auth", "web"}
	srv.runState.setServeRequestArgs(explicit)
	if err := srv.planServe(explicit); err != nil {
		t.Fatalf("planServe(explicit) error = %v", err)
	}
	assertRunStateTargets(t, srv, []string{"auth", "web"}, "before restart with explicit targets")

	if err := srv.start(false); err != nil {
		t.Fatalf("start(false) error = %v", err)
	}
	if err := srv.Restart(); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	assertRunStateTargets(t, srv, []string{"auth", "web"}, "after restart with explicit targets")
	if got := srv.runState.serveRequest(); !reflect.DeepEqual(got, explicit) {
		t.Fatalf("serveRequest() = %#v, want %#v", got, explicit)
	}
}

func writeRestartReplanManifest(t *testing.T, distRoot string, topo []string, apps map[string]distmanifest.DistManifestApp) {
	t.Helper()
	manifest := distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		CompileBundleMode: "bundle",
		HasWeb:            true,
		BackendTopoOrder:  topo,
		Apps:              apps,
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal dist manifest: %v", err)
	}
	path := filepath.Join(distRoot, distmanifest.DistManifestFileName)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write dist manifest: %v", err)
	}
}

func seedRestartReplanAppProto(t *testing.T, distRoot, app string) {
	t.Helper()
	protoDir := config.APIAppProtoDir(distRoot, app)
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", protoDir, err)
	}
}
