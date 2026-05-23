// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

type serverTestScope struct {
	ctx    context.Context
	cfg    *config.Config
	logger *slog.Logger
}

type fakeTelemetry struct{}

func (fakeTelemetry) ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{grpc.MaxRecvMsgSize(1024)}
}

func (fakeTelemetry) Shutdown(context.Context) error { return nil }

type fakeRegistry struct{}

func (fakeRegistry) Scheme() string                                                 { return "fake" }
func (fakeRegistry) Register(string, *resolver.Address) (*registry.Endpoint, error) { return nil, nil }
func (fakeRegistry) UnRegister(*registry.Endpoint) error                            { return nil }
func (fakeRegistry) UnRegisterAll() error                                           { return nil }
func (fakeRegistry) ListServices() ([]*registry.Endpoint, error)                    { return nil, nil }
func (fakeRegistry) GetService(string) ([]*registry.Endpoint, error)                { return nil, nil }
func (fakeRegistry) Resolver() resolver.Builder                                     { return fakeResolverBuilder{} }

type fakeResolverBuilder struct{}

func (fakeResolverBuilder) Scheme() string { return "fake" }

func (fakeResolverBuilder) Build(resolver.Target, resolver.ClientConn, resolver.BuildOptions) (resolver.Resolver, error) {
	return fakeResolver{}, nil
}

type fakeResolver struct{}

func (fakeResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (fakeResolver) Close()                                {}

type trackingTaskGarbageCollector struct {
	startCalls int
	stopCalls  int
}

func (g *trackingTaskGarbageCollector) Start() { g.startCalls++ }
func (g *trackingTaskGarbageCollector) Stop()  { g.stopCalls++ }

type fixedResolverBuilder struct {
	scheme string
	addr   string
}

func (b fixedResolverBuilder) Scheme() string { return b.scheme }

func (b fixedResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	if err := cc.UpdateState(resolver.State{Addresses: []resolver.Address{{Addr: b.addr}}}); err != nil {
		return nil, err
	}
	return fixedResolver{}, nil
}

type fixedResolver struct{}

func (fixedResolver) ResolveNow(resolver.ResolveNowOptions) {}
func (fixedResolver) Close()                                {}

type fixedResolverRegistry struct {
	scheme  string
	builder resolver.Builder
}

func (r fixedResolverRegistry) Scheme() string { return r.scheme }

func (r fixedResolverRegistry) Register(string, *resolver.Address) (*registry.Endpoint, error) {
	return nil, nil
}

func (r fixedResolverRegistry) UnRegister(*registry.Endpoint) error { return nil }
func (r fixedResolverRegistry) UnRegisterAll() error                { return nil }

func (r fixedResolverRegistry) ListServices() ([]*registry.Endpoint, error) {
	return nil, nil
}

func (r fixedResolverRegistry) GetService(string) ([]*registry.Endpoint, error) { return nil, nil }
func (r fixedResolverRegistry) Resolver() resolver.Builder                      { return r.builder }

type trackingRegistry struct {
	registerCalls    []string
	registeredAddrs  []*resolver.Address
	registerErr      error
	unregisterAllCnt int
	unregisterAllErr error
}

func (r *trackingRegistry) Scheme() string { return "fake" }

func (r *trackingRegistry) Register(serviceName string, addr *resolver.Address) (*registry.Endpoint, error) {
	r.registerCalls = append(r.registerCalls, serviceName)
	r.registeredAddrs = append(r.registeredAddrs, addr)
	if r.registerErr != nil {
		return nil, r.registerErr
	}
	return &registry.Endpoint{ServiceName: serviceName, Address: addr}, nil
}

func (r *trackingRegistry) UnRegister(*registry.Endpoint) error { return nil }

func (r *trackingRegistry) UnRegisterAll() error {
	r.unregisterAllCnt++
	return r.unregisterAllErr
}

func (r *trackingRegistry) ListServices() ([]*registry.Endpoint, error) { return nil, nil }

func (r *trackingRegistry) GetService(string) ([]*registry.Endpoint, error) {
	return nil, nil
}

func (r *trackingRegistry) Resolver() resolver.Builder { return fakeResolverBuilder{} }

type fakeAuthenticator struct {
	closed     int
	validateFn func(context.Context, string, auth.TokenType, bool) (auth.Identity, error)
}

type serverTestIdentity struct {
	userID   string
	tokenID  string
	metadata map[string]interface{}
	valid    bool
}

func (i serverTestIdentity) GetUserID() string                   { return i.userID }
func (i serverTestIdentity) GetTokenID() string                  { return i.tokenID }
func (i serverTestIdentity) GetMetadata() map[string]interface{} { return i.metadata }
func (i serverTestIdentity) IsValid() bool                       { return i.valid }

func (f *fakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	if f.validateFn != nil {
		return f.validateFn(ctx, token, tokenType, checkRevoked)
	}
	return nil, nil
}

func (f *fakeAuthenticator) CreateTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}

func (f *fakeAuthenticator) RefreshTokens(context.Context, string, map[string]interface{}) (*auth.TokenPair, error) {
	return nil, nil
}

func (f *fakeAuthenticator) RevokeToken(context.Context, string, string) error { return nil }

func (f *fakeAuthenticator) RevokeAllUserTokens(context.Context, string, string, string) (int, error) {
	return 0, nil
}

func (f *fakeAuthenticator) Close() error {
	f.closed++
	return nil
}

type fakeMetricTelemetry struct {
	shutdowns       int
	metricShutdowns int
}

func (f *fakeMetricTelemetry) ServerOptions() []grpc.ServerOption { return nil }

func (f *fakeMetricTelemetry) Shutdown(context.Context) error {
	f.shutdowns++
	return nil
}

func (f *fakeMetricTelemetry) MetricShutdown(context.Context) error {
	f.metricShutdowns++
	return nil
}

type noSessionServerScope struct {
	*serverTestScope
}

func (e *noSessionServerScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *noSessionServerScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *noSessionServerScope) Session() *scope.Session { return nil }

func (e *noSessionServerScope) WithContext(ctx context.Context) scope.Scope {
	return &noSessionServerScope{serverTestScope: &serverTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger}}
}

type fakeWatchedService struct {
	name      string
	descs     []*grpc.ServiceDesc
	descErr   error
	webErr    error
	handlers  map[string]http.Handler
	scripts   []*jsengine.JsScript
	watchDirs []string
	watchErr  error
	mu        sync.Mutex
	callCh    chan watchedCall
	called    []watchedCall
}

type watchedCall struct {
	module string
	file   string
}

func (f *fakeWatchedService) Name() string { return f.name }

func (f *fakeWatchedService) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	return f.descs, f.descErr
}

func (f *fakeWatchedService) ServiceScripts() []*jsengine.JsScript { return f.scripts }

func (f *fakeWatchedService) WebHandlers() (map[string]http.Handler, error) {
	return f.handlers, f.webErr
}

func (f *fakeWatchedService) watchCallback(moduleName string, file string) error {
	call := watchedCall{module: moduleName, file: file}
	f.mu.Lock()
	f.called = append(f.called, call)
	callCh := f.callCh
	f.mu.Unlock()
	if callCh != nil {
		select {
		case callCh <- call:
		default:
		}
	}
	return f.watchErr
}

func (f *fakeWatchedService) watchPlans() []registrationWatchPlan {
	plans := make([]registrationWatchPlan, 0, len(f.watchDirs))
	for _, watchDir := range f.watchDirs {
		plans = append(plans, registrationWatchPlan{
			ServiceName: f.name,
			ModuleName:  filepath.Base(watchDir),
			Root:        watchDir,
		})
	}
	return plans
}

func (f *fakeWatchedService) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.called)
}

func (f *fakeWatchedService) firstCall() (watchedCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.called) == 0 {
		return watchedCall{}, false
	}
	return f.called[0], true
}

func (e *serverTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }

func (e *serverTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *serverTestScope) Session() *scope.Session { return &scope.Session{} }

func (e *serverTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}

func (e *serverTestScope) Context() context.Context { return e.ctx }

func (e *serverTestScope) Logger() *slog.Logger {
	if e.logger != nil {
		return e.logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *serverTestScope) Config() *config.Config { return e.cfg }

func (e *serverTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newServerTestScope() *serverTestScope {
	return &serverTestScope{
		ctx: context.Background(),
		cfg: &config.Config{Server: config.NewDefaultServerConfig()},
	}
}

func newRichServerTestScope(t *testing.T) *serverTestScope {
	t.Helper()
	return &serverTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Server:  config.NewDefaultServerConfig(),
			Auth:    config.NewDefaultAuthConfig(),
			Compile: config.NewDefaultCompileConfig(),
			Log:     config.NewDefaultLogConfig(),
			Db:      config.NewDefaultDbConfig(),
		},
	}
}

func writeTestTLSFiles(t *testing.T) (caPath string, certPath string, keyPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.internal"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"example.internal"},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	dir := t.TempDir()
	caPath = filepath.Join(dir, "ca.pem")
	certPath = filepath.Join(dir, "client.pem")
	keyPath = filepath.Join(dir, "client.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(caPath, certPEM, 0o644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		t.Fatalf("write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	return caPath, certPath, keyPath
}

func mustNewWatcher(t *testing.T) *fsnotify.Watcher {
	t.Helper()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	return watcher
}

func writeServerTestAppDist(t *testing.T, distRoot string, appName string, script string) {
	t.Helper()
	appDir := filepath.Join(distRoot, "apps", appName)
	assetsDir := filepath.Join(appDir, "assets")
	for _, dir := range []string{appDir, assetsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(appDir, "index.js"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(index.js) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "service.proto"), []byte("syntax = \"proto3\";\npackage auth;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(service.proto) error = %v", err)
	}
}
