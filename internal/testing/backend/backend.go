// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/server/middleware/auth/grpcauth"
	internalservice "github.com/choysum-dev/choysum/internal/service"
	cov "github.com/choysum-dev/choysum/internal/testing/coverage"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	grpcloader "github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysumtest"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const inProcessGrpcBufSize = 1024 * 1024

const jsConsoleEmitter = "js_console"

var (
	makeTestScopeHook             = makeTestScope
	newCompilerExecutorHook       = newCompilerExecutor
	startInProcessGrpcHarnessHook = startInProcessGrpcHarness
	prepareBackendHook            backendPrepareHook
	executeBackendHook            backendExecuteHook
	newBackendBuilderHook                   = defaultNewBackendBuilder
	newHarnessServiceHook                   = defaultNewHarnessService
	newQuickJSAuthenticatorHook             = defaultNewQuickJSAuthenticator
	backendProgressWriter         io.Writer = os.Stderr
)

type backendPrepareHook func(ctx context.Context, testRuntimeScope scope.Scope, repoRoot, app string, coverage bool, jsExec jsexecutor.JsExecutor) (func(), error)
type backendExecuteHook func(ctx context.Context, testRuntimeScope scope.Scope, app, pattern string, failFast bool) (any, error)

func writeBackendProgress(format string, args ...any) {
	if backendProgressWriter == nil {
		return
	}
	_, _ = fmt.Fprintf(backendProgressWriter, format, args...)
}

type unitTestRuntimeLogHandler struct {
	next slog.Handler
}

func (h *unitTestRuntimeLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h == nil || h.next == nil {
		return false
	}
	return h.next.Enabled(ctx, level)
}

func (h *unitTestRuntimeLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h == nil || h.next == nil {
		return nil
	}
	if shouldDropUnitTestRuntimeRecord(ctx, h.next, record) {
		return nil
	}
	return h.next.Handle(ctx, record)
}

func (h *unitTestRuntimeLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil || h.next == nil {
		return h
	}
	return &unitTestRuntimeLogHandler{next: h.next.WithAttrs(attrs)}
}

func (h *unitTestRuntimeLogHandler) WithGroup(group string) slog.Handler {
	if h == nil || h.next == nil {
		return h
	}
	return &unitTestRuntimeLogHandler{next: h.next.WithGroup(group)}
}

func unitTestRuntimeLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return slog.New(&unitTestRuntimeLogHandler{next: logger.Handler()})
}

func shouldDropUnitTestRuntimeRecord(ctx context.Context, next slog.Handler, record slog.Record) bool {
	if next == nil {
		return false
	}
	if record.Message != "js console message" {
		return false
	}
	if next.Enabled(ctx, slog.LevelDebug) {
		return false
	}
	if !recordHasStringAttr(record, "emitter", jsConsoleEmitter) {
		return false
	}
	return recordHasBoolAttr(record, "passthrough", true)
}

func recordHasStringAttr(record slog.Record, key string, want string) bool {
	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key != key {
			return true
		}
		found = attr.Value.Kind() == slog.KindString && attr.Value.String() == want
		return false
	})
	return found
}

func recordHasBoolAttr(record slog.Record, key string, want bool) bool {
	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key != key {
			return true
		}
		found = attr.Value.Kind() == slog.KindBool && attr.Value.Bool() == want
		return false
	})
	return found
}

type harnessService interface {
	ServiceDescs() ([]*grpc.ServiceDesc, error)
	ServiceScripts() []*jsengine.JsScript
}

func defaultNewBackendBuilder(runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, mod *meta.Module, entryPoint, outFileName, globalName string) any {
	return internalbackendbuilder.NewModuleBuilder(
		runtimeScope,
		jsExec,
		mod,
		entryPoint,
		internalbackendbuilder.WithPublishDist(true),
		internalbackendbuilder.WithOutFileName(outFileName),
		internalbackendbuilder.WithGlobalName(globalName),
	)
}

func defaultNewHarnessService(runtimeScope scope.Scope, name string, jsExec jsexecutor.JsExecutor, mode string) (harnessService, error) {
	return internalservice.NewApplicationService(runtimeScope, name, jsExec, internalservice.WithBundleMode(mode))
}

func defaultNewQuickJSAuthenticator(runtimeScope scope.Scope) (auth.Authenticator, error) {
	return auth.NewAuthenticator(runtimeScope)
}

func shouldSkipTestScanDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".choysum", "tmp":
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.IsDir()
}

func resolveBundleModeForTests(runtimeScope scope.Scope) string {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	mode := strings.ToLower(strings.TrimSpace(runtimeOpts.compileBundleMode))
	if mode == "" {
		mode = "bundle"
	}
	if mode != "bundle" || !runtimeOpts.hasConfig {
		return mode
	}

	distRoot := runtimeOpts.distPath
	bundlesIndex := config.BundlesIndexJS(distRoot)
	apiRoot := config.APIRootFromDist(distRoot)
	if !fileExists(bundlesIndex) || !dirExists(apiRoot) {
		return "application"
	}
	return mode
}

type inProcessGrpcHarness struct {
	listener      *bufconn.Listener
	server        *grpc.Server
	connMu        sync.Mutex
	conn          *grpc.ClientConn
	connErr       error
	serviceDialer client.ServiceDialer

	runtimeExec   jsexecutor.JsExecutor
	authenticator auth.Authenticator
}

func (h *inProcessGrpcHarness) Close() {
	if h == nil {
		return
	}
	if h.server != nil {
		h.server.Stop()
		h.server = nil
	}
	if h.listener != nil {
		h.listener.Close()
		h.listener = nil
	}
	h.connMu.Lock()
	cc := h.conn
	h.conn = nil
	h.connMu.Unlock()
	if cc != nil {
		_ = cc.Close()
	}
	if h.runtimeExec != nil {
		_ = h.runtimeExec.Stop()
		h.runtimeExec = nil
	}
	if h.authenticator != nil {
		_ = h.authenticator.Close()
		h.authenticator = nil
	}
}

func startInProcessGrpcHarness(ctx context.Context, runtimeScope scope.Scope) (*inProcessGrpcHarness, error) {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if runtimeScope == nil || !runtimeOpts.hasConfig {
		return nil, xfmt.Errorf("invalid scope")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	h := &inProcessGrpcHarness{listener: bufconn.Listen(inProcessGrpcBufSize)}

	// Authenticator (optional).
	if runtimeOpts.authEnabled {
		authenticator, err := auth.NewAuthenticator(runtimeScope)
		if err != nil {
			h.Close()
			return nil, err
		}
		h.authenticator = authenticator
	}

	// Runtime JS executor for gRPC service handlers.
	runtimeExec, err := newRuntimeExecutor(runtimeScope, h.authenticator)
	if err != nil {
		h.Close()
		return nil, err
	}
	h.runtimeExec = runtimeExec

	mode := resolveBundleModeForTests(runtimeScope)

	// Load service scripts.
	var initScripts []*jsengine.JsScript
	if mode == "application" {
		appsDir := filepath.Join(runtimeOpts.distPath, "apps")
		appEntries, err := os.ReadDir(appsDir)
		if err != nil {
			h.Close()
			return nil, xfmt.Errorf("read apps dist dir: %w", err)
		}
		for _, entry := range appEntries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			svc, err := newHarnessServiceHook(runtimeScope, name, runtimeExec, mode)
			if err != nil {
				h.Close()
				return nil, xfmt.Errorf("create service %s: %w", name, err)
			}
			initScripts = append(initScripts, svc.ServiceScripts()...)
		}
	} else {
		// Bundle mode: scripts come from dist/bundles/index.js and are shared.
		// We still create per-app services for proto descriptors.
		apiRoot := config.APIRootFromDist(runtimeOpts.distPath)
		assetApps, err := os.ReadDir(apiRoot)
		if err != nil {
			h.Close()
			return nil, xfmt.Errorf("read api root dir: %w", err)
		}
		picked := ""
		for _, e := range assetApps {
			if !e.IsDir() {
				continue
			}
			if dirExists(filepath.Join(apiRoot, e.Name(), "proto")) {
				picked = e.Name()
				break
			}
		}
		if strings.TrimSpace(picked) == "" {
			// No backend apps/protos built.
			picked = "web"
		}
		svc, err := newHarnessServiceHook(runtimeScope, picked, runtimeExec, mode)
		if err != nil {
			h.Close()
			return nil, xfmt.Errorf("create bundle service %s: %w", picked, err)
		}
		initScripts = append(initScripts, svc.ServiceScripts()...)
	}
	initScripts, err = jsengine.DedupeInitScripts(initScripts)
	if err != nil {
		h.Close()
		return nil, xfmt.Errorf("dedupe init scripts: %w", err)
	}
	runtimeExec.SetJsScripts(initScripts)
	if err := runtimeExec.Start(); err != nil {
		h.Close()
		return nil, xfmt.Errorf("start runtime executor: %w", err)
	}

	// Lazy shared in-process client conn.
	h.serviceDialer = func(ctx context.Context, _ string) (*grpc.ClientConn, error) {
		h.connMu.Lock()
		defer h.connMu.Unlock()
		if h.conn != nil || h.connErr != nil {
			return h.conn, h.connErr
		}
		// Use passthrough resolver to avoid DNS lookup on logical target names.
		// The actual connection is provided by bufconn.
		cc, err := grpc.DialContext(
			ctx,
			"passthrough:///bufnet",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return h.listener.Dial()
			}),
		)
		if err != nil {
			h.connErr = err
			return nil, err
		}
		h.conn = cc
		return cc, nil
	}

	// gRPC server: auth (if enabled) + inject dialer for nested calls.
	interceptors := []grpc.UnaryServerInterceptor{
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			ctx = client.ContextWithServiceDialer(ctx, h.serviceDialer)
			return handler(ctx, req)
		},
	}
	if h.authenticator != nil {
		interceptors = append([]grpc.UnaryServerInterceptor{grpcauth.AuthInterceptorFromConfig(runtimeScope, h.authenticator)}, interceptors...)
	}
	h.server = grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))

	// Register gRPC service descriptors.
	if mode == "application" {
		appsDir := filepath.Join(runtimeOpts.distPath, "apps")
		appEntries, err := os.ReadDir(appsDir)
		if err != nil {
			h.Close()
			return nil, xfmt.Errorf("read apps dist dir: %w", err)
		}
		for _, entry := range appEntries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			svc, err := newHarnessServiceHook(runtimeScope, name, runtimeExec, mode)
			if err != nil {
				h.Close()
				return nil, xfmt.Errorf("create service %s: %w", name, err)
			}
			sds, err := svc.ServiceDescs()
			if err != nil {
				h.Close()
				return nil, xfmt.Errorf("service descs %s: %w", name, err)
			}
			for _, sd := range sds {
				h.server.RegisterService(sd, nil)
			}
		}
	} else {
		apiRoot := config.APIRootFromDist(runtimeOpts.distPath)
		assetApps, err := os.ReadDir(apiRoot)
		if err != nil {
			h.Close()
			return nil, xfmt.Errorf("read api root dir: %w", err)
		}
		for _, entry := range assetApps {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !dirExists(filepath.Join(apiRoot, name, "proto")) {
				continue
			}
			svc, err := newHarnessServiceHook(runtimeScope, name, runtimeExec, mode)
			if err != nil {
				h.Close()
				return nil, xfmt.Errorf("create service %s: %w", name, err)
			}
			sds, err := svc.ServiceDescs()
			if err != nil {
				h.Close()
				return nil, xfmt.Errorf("service descs %s: %w", name, err)
			}
			for _, sd := range sds {
				h.server.RegisterService(sd, nil)
			}
		}
	}

	go func() {
		_ = h.server.Serve(h.listener)
	}()

	return h, nil
}

type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr,omitempty"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       string           `xml:"time,attr"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	ClassName string        `xml:"classname,attr,omitempty"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr,omitempty"`
	Body    string `xml:",chardata"`
}

// shouldSkipWebShellForUnitApp skips SPA shell install for non-auth unit shards.
func shouldSkipWebShellForUnitApp(app string) bool {
	return !strings.EqualFold(strings.TrimSpace(app), "auth")
}

// shouldInstallMetaForUnitApp installs meta for auth (gRPC) and web (UserFilter ModelId).
func shouldInstallMetaForUnitApp(app string) bool {
	app = strings.TrimSpace(app)
	return strings.EqualFold(app, "auth") || strings.EqualFold(app, "web")
}

type unitAppInstaller interface {
	Install(ctx context.Context, req lifecycle.InstallRequest) error
}

// installUnitAppModules installs the unit shard (optionally skipping the web shell) then meta when needed.
func installUnitAppModules(ctx context.Context, installer unitAppInstaller, app string) error {
	if err := installer.Install(ctx, lifecycle.InstallRequest{
		Name:         app,
		SkipWebShell: shouldSkipWebShellForUnitApp(app),
	}); err != nil {
		return err
	}
	return ensureMetaInstalledForUnitApp(ctx, installer, app)
}

// ensureMetaInstalledForUnitApp installs meta when the shard needs MetaModel/gRPC services.
func ensureMetaInstalledForUnitApp(ctx context.Context, installer unitAppInstaller, app string) error {
	if !shouldInstallMetaForUnitApp(app) {
		return nil
	}
	return installer.Install(ctx, lifecycle.InstallRequest{Name: "meta", SkipWebShell: true})
}

// jsContextWithUnitTestIdentity seeds bootstrap admin into JsRequest.Context when auth is installed.
func jsContextWithUnitTestIdentity(ctx context.Context, testScope scope.Scope) (map[string]interface{}, error) {
	jsCtx := map[string]interface{}{}
	identity, ok, idErr := resolveUnitTestDefaultIdentity(ctx, testScope)
	if idErr != nil {
		return nil, xfmt.Errorf("resolve unit test default identity: %w", idErr)
	}
	if ok {
		// When auth is in the install closure, seed bootstrap admin so domain
		// fixtures are not anonymous-denied by record rules. choysumtest
		// re-applies this before each case (tests may clear identity).
		jsCtx = unitTestJsRequestContext(identity)
	}
	return jsCtx, nil
}

// unitTestIdentityContextFn resolves JsRequest.Context for backend unit runs (overridable in tests).
var unitTestIdentityContextFn = jsContextWithUnitTestIdentity

// loadUnitAppTestContext resolves JsRequest.Context, surfacing identity resolution failures.
func loadUnitAppTestContext(ctx context.Context, testScope scope.Scope) (map[string]interface{}, error) {
	jsCtx, idErr := unitTestIdentityContextFn(ctx, testScope)
	if idErr != nil {
		return nil, idErr
	}
	return jsCtx, nil
}

func RunOneAppBackendTests(
	ctx context.Context,
	baseScope scope.Scope,
	app string,
	repoRoot string,
	dbDialect string,
	dbFile string,
	dbDSN string,
	keep bool,
	junitPath string,
	pattern string,
	failFast bool,
	coverage bool,
) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}
	if coverage && prepareBackendHook == nil {
		if err := cov.PreflightInstrumentationPrerequisites(repoRoot); err != nil {
			return false, err
		}
	}
	// Proto registrations are process-global in production, but each unit-test
	// application gets its own database, API assets, and gRPC harness. Without
	// this reset, an earlier app can leave a method descriptor visible to a
	// later app whose harness does not register that service.
	grpcloader.ResetGlobalForTests()
	prepareStarted := time.Now()
	writeBackendProgress("# prepare runtime %s\n", app)

	testScope, cleanup, err := makeTestScopeHook(ctx, baseScope, app, dbDialect, dbFile, dbDSN, keep)
	if err != nil {
		return false, err
	}
	defer cleanup()
	testOpts := runtimeOptionsFromScope(testScope)
	if testOpts.hasConfig {
		distRoot := strings.TrimSpace(testOpts.distPath)
		if distRoot != "" {
			// Safe because makeTestScope isolates DistPath under a temp runRoot;
			// APIRootFromDist is the sibling <runRoot>/api, not developer ~/.choysum/api.
			apiRoot := filepath.Clean(config.APIRootFromDist(distRoot))
			if apiRoot == "." || apiRoot == string(filepath.Separator) {
				return false, xfmt.Errorf("invalid api root for backend tests: %s", apiRoot)
			}
			if err := os.RemoveAll(apiRoot); err != nil {
				return false, xfmt.Errorf("reset runtime api root %s: %w", apiRoot, err)
			}
		}
	}

	// Use the test scope for install/migrations and bundling so the temporary DB
	// has the business tables required by JS-side DB calls.
	jsExec, err := newCompilerExecutorHook(testScope)
	if err != nil {
		return false, err
	}
	if jsExec != nil {
		defer jsExec.Stop()
	}

	// 1) Build app service bundle (index.js) + tests bundle (tests.js)
	// Ensure a DB session is available for build-time plugins.
	cleanupGeneratedEntry := func() {}
	if prepareBackendHook != nil {
		cleanupFn, err := prepareBackendHook(ctx, testScope, repoRoot, app, coverage, jsExec)
		if err != nil {
			return false, err
		}
		if cleanupFn != nil {
			cleanupGeneratedEntry = cleanupFn
		}
	} else {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		// Let module installation manage its own transactional/lease lifecycle.
		// The outer test transaction is only needed for bundle/test execution state.
		//
		// Skip the SPA shell for most domain shards: entryPoints.web would otherwise
		// pull web→document→auth into e.g. base. Auth is the exception — its BE
		// suite needs global web build to persist declared MetaUiResource rows
		// (PermissionState smoke uses auth.route.token_list, etc.).
		moduleLifecycle := lifecycle.NewService(testScope, jsExec)
		// Auth backend tests rely on meta gRPC services (Model/Application).
		// Web UserFilter tests dial meta.MetaModel for effective ModelId (SF12).
		if err := installUnitAppModules(ctx, moduleLifecycle, app); err != nil {
			return false, err
		}

		txCtx := testScope.Context()
		if err := testScope.Transactor().Required(txCtx, func(runtimeScope scope.Scope, _ scope.Transaction) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			runtimeOpts := runtimeOptionsFromScope(runtimeScope)

			serviceEntry, err := resolveServiceEntryPoint(runtimeScope, app)
			if err != nil {
				return err
			}
			if err := buildAppBundle(ctx, runtimeScope, jsExec, app, "index.js", app, serviceEntry); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}

			// Auto-discover tests under modules/<app>/service/**/*.test.ts and generate a temporary entry.
			entry, cleanup, err := resolveOrGenerateTestsEntryPoint(ctx, runtimeScope, app)
			if err != nil {
				return err
			}
			if keep {
				cleanupGeneratedEntry = func() {}
			} else {
				cleanupGeneratedEntry = cleanup
			}
			if err := buildAppBundle(ctx, runtimeScope, jsExec, app, "tests.js", "__tests__", entry); err != nil {
				return err
			}
			if coverage {
				if err := cov.InstrumentDistBundleWithTmpRoot(ctx, repoRoot, runtimeOpts.distPath, app, runtimeOpts.tmpPath); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return false, err
		}
	}
	writeBackendProgress("# prepare runtime %s ok (%s)\n", app, time.Since(prepareStarted).Round(100*time.Millisecond))
	defer cleanupGeneratedEntry()
	if err := ctx.Err(); err != nil {
		return true, err
	}

	var rawResult any
	if executeBackendHook != nil {
		rawResult, err = executeBackendHook(ctx, testScope, app, pattern, failFast)
		if err != nil {
			return true, xfmt.Errorf("execute tests: %w", err)
		}
	} else {
		// 3) Run in QuickJS via existing RPC entrypoint
		// Provide authenticator to QuickJS so $choysum.auth is available.
		// Without this, JS code that references $choysum.auth (e.g. Token.CreateTokenPair)
		// will crash with TypeError.
		var quickjsAuthenticator auth.Authenticator
		if testOpts.authEnabled {
			jwtAuth, err := newQuickJSAuthenticatorHook(testScope)
			if err != nil {
				return false, err
			}
			quickjsAuthenticator = jwtAuth
			defer jwtAuth.Close()
		}

		factory := jsengine.NewJsEngineFactory(testScope, quickjsAuthenticator)
		if factory == nil {
			return false, xfmt.Errorf("JavaScript engine factory is not registered: %s", testOpts.serverJsEngineFactory)
		}
		engine, err := factory()
		if err != nil {
			return false, xfmt.Errorf("create js engine: %w", err)
		}
		defer engine.Close()

		distRoot := testOpts.distPath
		configuredMode := strings.ToLower(strings.TrimSpace(testOpts.compileBundleMode))
		if configuredMode == "" {
			configuredMode = "bundle"
		}
		mode := resolveBundleModeForTests(testScope)

		// In application mode, load per-app dist/apps/<app>/index.js.
		// In bundle mode, load dist/bundles/index.js.
		indexPath := ""
		testsPath := ""
		if mode == "application" {
			indexPath = config.AppIndexJS(distRoot, app)
		} else {
			indexPath = config.BundlesIndexJS(distRoot)
		}

		// Tests bundle is generated according to the configured bundle mode.
		// Prefer the configured layout, but fall back to whichever file exists.
		var testsCandidates []string
		if configuredMode == "bundle" {
			testsCandidates = append(testsCandidates, filepath.Join(config.BundlesDir(distRoot), "tests.js"))
			testsCandidates = append(testsCandidates, filepath.Join(config.AppDir(distRoot, app), "tests.js"))
		} else {
			testsCandidates = append(testsCandidates, filepath.Join(config.AppDir(distRoot, app), "tests.js"))
			testsCandidates = append(testsCandidates, filepath.Join(config.BundlesDir(distRoot), "tests.js"))
		}
		for _, candidate := range testsCandidates {
			if fileExists(candidate) {
				testsPath = candidate
				break
			}
		}
		if testsPath == "" && len(testsCandidates) > 0 {
			testsPath = testsCandidates[0]
		}

		appJS, err := os.ReadFile(indexPath)
		if err != nil {
			return false, xfmt.Errorf("read dist index.js failed (bundleMode=%s, path=%s; check dist layout under %s): %w", mode, indexPath, distRoot, err)
		}
		testsJS, err := os.ReadFile(testsPath)
		if err != nil {
			return false, xfmt.Errorf("read dist tests.js failed (bundleMode=%s, path=%s; check dist layout under %s): %w", mode, testsPath, distRoot, err)
		}

		if err := engine.Load([]*jsengine.JsScript{
			{FileName: "scripts/choysumtest/choysumtest.js", Content: choysumtest.ChoysumTestScript},
			{FileName: indexPath, Content: string(appJS)},
			{FileName: testsPath, Content: string(testsJS)},
		}); err != nil {
			return false, xfmt.Errorf("load scripts: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return true, err
		}

		jsCtx, err := loadUnitAppTestContext(ctx, testScope)
		if err != nil {
			return false, err
		}
		req := &jsengine.JsRequest{
			Id:      fmt.Sprintf("test-%s-%d", app, time.Now().UnixNano()),
			Service: "__tests__.Run",
			Args: []interface{}{map[string]any{
				"pattern":  pattern,
				"failFast": failFast,
			}},
			Context: jsCtx,
		}

		// Execute tests within a DB session context so $choysum.db bridges work.
		var res *jsengine.JsResponse
		txCtx := testScope.Context()
		if err := testScope.Transactor().Required(txCtx, func(runtimeScope scope.Scope, _ scope.Transaction) error {
			execCtx := scope.ContextWithScope(ctx, runtimeScope)
			runtimeOpts := runtimeOptionsFromScope(runtimeScope)

			// Start an in-process gRPC server so JS-side $choysum.grpc calls can resolve
			// logical service names (e.g. meta.Model) without requiring an external server.
			grpcHarness, err := startInProcessGrpcHarnessHook(execCtx, runtimeScope)
			if err != nil {
				return err
			}
			defer grpcHarness.Close()

			// Seed incoming metadata so outbound JS->gRPC bridge marks nested calls with depth>0.
			// Without this, internal calls look like depth=0 and can trigger top-level ACL/policy enforcement.
			execCtx = metadata.NewIncomingContext(execCtx, metadata.Pairs("x-choysum-depth", "0"))

			// Provide an access token in trusted (Go-only) context for the JS->gRPC bridge.
			// This token is not exposed to JS but is required to attach the Authorization header.
			// Use a signed JWT token when auth is enabled; otherwise a dummy value is sufficient.
			if runtimeOpts.authEnabled {
				pair, err := quickjsAuthenticator.CreateTokens(execCtx, "choysum_test_runner", map[string]interface{}{"purpose": "choysum test"})
				if err != nil {
					return err
				}
				execCtx = auth.ContextWithAccessToken(execCtx, pair.AccessToken)
			} else {
				execCtx = auth.ContextWithAccessToken(execCtx, "choysum_test_dummy")
			}

			// Inject service dialer so $choysum.grpc.unary can dial logical service names.
			execCtx = client.ContextWithServiceDialer(execCtx, grpcHarness.serviceDialer)

			r, err := engine.Execute(execCtx, req)
			if err != nil {
				return err
			}
			res = r
			return nil
		}); err != nil {
			return true, xfmt.Errorf("execute tests: %w", err)
		}
		rawResult = res.Result
	}

	report, err := parseReport(rawResult)
	if err != nil {
		return true, xfmt.Errorf("parse test report: %w", err)
	}

	writeTAP(os.Stdout, report)

	failed, junitErr := writeJUnitIfNeeded(app, report, junitPath)
	if junitErr != nil {
		return failed, junitErr
	}

	if coverage {
		runID := cov.CoverageRunIDFromContext(ctx)
		if err := cov.WriteCoverageJSONWithRunIDAndTmpRoot(repoRoot, app, runID, report.coverageJSON, testOpts.tmpPath); err != nil {
			return true, err
		}
	}

	if failed {
		// keep running remaining apps in "all" mode
		baseScope.Logger().Warn("backend tests failed", slog.String("app", app))
	}
	return failed, nil
}

func resolveOrGenerateTestsEntryPoint(ctx context.Context, runtimeScope scope.Scope, app string) (string, func(), error) {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	modulesPath := runtimeOpts.modulesPath
	serviceDir := filepath.Join(modulesPath, app, "service")

	files, err := listTestFiles(serviceDir)
	if err != nil {
		return "", func() {}, xfmt.Errorf("scan test files: %w", err)
	}
	if len(files) == 0 {
		return "", func() {}, xfmt.Errorf("no test files found under %s", serviceDir)
	}

	sort.Strings(files)

	genDir, err := backendTestsIndexTmpDir(ctx, runtimeOpts.modulesPath, runtimeOpts.tmpPath, app)
	if err != nil {
		return "", func() {}, xfmt.Errorf("resolve tests index tmp dir: %w", err)
	}
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return "", func() {}, xfmt.Errorf("mkdir %s: %w", genDir, err)
	}
	genPath := filepath.Join(genDir, fmt.Sprintf("index.%d.ts", time.Now().UnixNano()))

	var b strings.Builder
	for _, abs := range files {
		imp := filepath.ToSlash(filepath.Clean(abs))
		b.WriteString("import '")
		b.WriteString(imp)
		b.WriteString("';\n")
	}
	b.WriteString("\n")
	b.WriteString("export async function Run(options?: { pattern?: string; failFast?: boolean }) {\n")
	b.WriteString("  // The runner is injected by `choysum test` (QuickJS init script).\n")
	b.WriteString("  // eslint-disable-next-line @typescript-eslint/no-explicit-any\n")
	b.WriteString("  const fn = (globalThis as any).__choysum_test_run__;\n")
	b.WriteString("  if (typeof fn !== 'function') {\n")
	b.WriteString("    throw new Error('Missing __choysum_test_run__ in globalThis');\n")
	b.WriteString("  }\n")
	b.WriteString("  return await fn(options);\n")
	b.WriteString("}\n")

	if err := os.WriteFile(genPath, []byte(b.String()), 0o644); err != nil {
		return "", func() {}, xfmt.Errorf("write generated tests entry: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(genPath)
		_ = os.Remove(genDir)
	}
	return genPath, cleanup, nil
}

func listTestFiles(serviceDir string) ([]string, error) {
	st, err := os.Stat(serviceDir)
	if err != nil || !st.IsDir() {
		return nil, nil
	}
	var out []string
	err = filepath.WalkDir(serviceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipTestScanDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".test.ts") {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			out = append(out, abs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func backendTmpDir(ctx context.Context, modulesPath string, tmpRoot string) (string, error) {
	workspaceRoot := strings.TrimSpace(modulesPath)
	if workspaceRoot != "" {
		workspaceRoot = filepath.Dir(workspaceRoot)
	}
	resolvedTmpRoot := strings.TrimSpace(tmpRoot)
	if resolvedTmpRoot == "" {
		resolvedTmpRoot = os.TempDir()
	}
	return testingpathing.ResolveTestingTmpDirFromContext(ctx, workspaceRoot, resolvedTmpRoot, "backend")
}

func backendTestsIndexTmpDir(ctx context.Context, modulesPath string, tmpRoot string, app string) (string, error) {
	backendTmpRoot, err := backendTmpDir(ctx, modulesPath, tmpRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(backendTmpRoot, "tests-index", sanitizeUnitAppToken(app)), nil
}

func unitTestDBTmpDir(ctx context.Context, modulesPath string, tmpRoot string, app string) (string, error) {
	workspaceRoot := strings.TrimSpace(modulesPath)
	if workspaceRoot != "" {
		workspaceRoot = filepath.Dir(workspaceRoot)
	}
	resolvedTmpRoot := testingpathing.EffectiveCLITestTmpRoot(ctx, tmpRoot)
	if resolvedTmpRoot == "" {
		resolvedTmpRoot = os.TempDir()
	}
	testDBTmpDir, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, workspaceRoot, resolvedTmpRoot, "testdb")
	if err != nil {
		return "", err
	}
	return filepath.Join(testDBTmpDir, sanitizeUnitAppToken(app)), nil
}

// unitTestRunRoot returns a per-app temporary run directory for backend unit tests.
// DistPath is placed at <runRoot>/dist so APIRootFromDist resolves to the sibling
// <runRoot>/api — mirroring e2e runDir isolation and avoiding mutation of the
// developer ~/.choysum/dist and ~/.choysum/api trees.
func unitTestRunRoot(ctx context.Context, modulesPath string, tmpRoot string, app string) (string, error) {
	workspaceRoot := strings.TrimSpace(modulesPath)
	if workspaceRoot != "" {
		workspaceRoot = filepath.Dir(workspaceRoot)
	}
	resolvedTmpRoot := testingpathing.EffectiveCLITestTmpRoot(ctx, tmpRoot)
	if resolvedTmpRoot == "" {
		resolvedTmpRoot = os.TempDir()
	}
	unitTmpDir, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, workspaceRoot, resolvedTmpRoot, "unit")
	if err != nil {
		return "", err
	}
	return filepath.Join(unitTmpDir, fmt.Sprintf("%s-%d", sanitizeUnitAppToken(app), time.Now().UnixNano())), nil
}

func sanitizeUnitAppToken(app string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token := strings.TrimSpace(replacer.Replace(app))
	if token == "" {
		return "app"
	}
	return token
}

func cleanupSQLiteFiles(sqlitePath string) {
	_ = os.Remove(sqlitePath)
	_ = os.Remove(sqlitePath + "-wal")
	_ = os.Remove(sqlitePath + "-shm")
}

func joinCleanups(fns ...func()) func() {
	return func() {
		for i := len(fns) - 1; i >= 0; i-- {
			if fns[i] != nil {
				fns[i]()
			}
		}
	}
}

// makeTestScope builds an isolated scope for one backend unit-test app.
// It always overrides DistPath to a temp runRoot (sibling api via APIRootFromDist),
// matching e2e runDir isolation so Install/RemoveAll never touch developer ~/.choysum assets.
func makeTestScope(ctx context.Context, baseScope scope.Scope, app string, dbDialect string, dbFile string, dbDSN string, keep bool) (scope.Scope, func(), error) {
	if baseScope == nil {
		return nil, func() {}, xfmt.Errorf("invalid base scope")
	}
	baseOpts := runtimeOptionsFromScope(baseScope)
	if !baseOpts.hasConfig {
		return nil, func() {}, xfmt.Errorf("invalid base scope")
	}
	baseDBOpts, hasBaseDBOpts := scope.DatabaseRuntimeOptionsFromScope(baseScope)
	if !hasBaseDBOpts || strings.TrimSpace(baseDBOpts.Dialect) == "" || strings.TrimSpace(baseDBOpts.DSN) == "" {
		return nil, func() {}, xfmt.Errorf("config missing db")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tmpRoot := testingpathing.EffectiveCLITestTmpRoot(ctx, baseOpts.tmpPath)
	runRoot, err := unitTestRunRoot(ctx, baseOpts.modulesPath, tmpRoot, app)
	if err != nil {
		return nil, func() {}, xfmt.Errorf("resolve unit run root: %w", err)
	}
	isolatedDist := filepath.Join(runRoot, "dist")
	if err := os.MkdirAll(isolatedDist, 0o755); err != nil {
		return nil, func() {}, xfmt.Errorf("create isolated dist dir: %w", err)
	}

	dbCopy := baseDBOpts
	var cleanupDB func()

	switch strings.ToLower(strings.TrimSpace(dbDialect)) {
	case "sqlite", "":
		sqlitePath := strings.TrimSpace(dbFile)
		if sqlitePath == "" {
			tmpDir, err := unitTestDBTmpDir(ctx, baseOpts.modulesPath, tmpRoot, app)
			if err != nil {
				_ = os.RemoveAll(runRoot)
				return nil, func() {}, xfmt.Errorf("resolve tmp dir: %w", err)
			}
			_ = os.MkdirAll(tmpDir, 0o755)
			sqlitePath = filepath.Join(tmpDir, fmt.Sprintf("%s-%d.sqlite", app, time.Now().UnixNano()))
		}
		dbCopy.Dialect = "sqlite"
		dbCopy.DSN = fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=10000&_journal_mode=WAL", sqlitePath)
		if !keep {
			cleanupDB = func() { cleanupSQLiteFiles(sqlitePath) }
		}
	case "postgres":
		dsn := strings.TrimSpace(dbDSN)
		if dsn == "" {
			dsn = strings.TrimSpace(os.Getenv("CHOYSUM_TEST_POSTGRES_DSN"))
		}
		if dsn == "" {
			_ = os.RemoveAll(runRoot)
			return nil, func() {}, xfmt.Errorf("--db=postgres requires --db-dsn or env CHOYSUM_TEST_POSTGRES_DSN")
		}

		adminDSN, err := setPostgresDatabaseInDSN(dsn, "postgres")
		if err != nil {
			_ = os.RemoveAll(runRoot)
			return nil, func() {}, xfmt.Errorf("invalid postgres dsn: %w", err)
		}

		// Create a dedicated DB per run for isolation.
		testDBName := makePostgresTestDBName(app)
		testDSN, err := setPostgresDatabaseInDSN(dsn, testDBName)
		if err != nil {
			_ = os.RemoveAll(runRoot)
			return nil, func() {}, xfmt.Errorf("invalid postgres dsn: %w", err)
		}

		adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		if err != nil {
			_ = os.RemoveAll(runRoot)
			return nil, func() {}, xfmt.Errorf("connect postgres (admin): %w", err)
		}
		sqlDB, _ := adminDB.DB()
		if sqlDB != nil {
			defer sqlDB.Close()
		}

		if err := adminDB.WithContext(ctx).Exec(fmt.Sprintf("CREATE DATABASE %s", quotePostgresIdent(testDBName))).Error; err != nil {
			_ = os.RemoveAll(runRoot)
			return nil, func() {}, xfmt.Errorf("create postgres test database %s: %w", testDBName, err)
		}
		fmt.Fprintf(os.Stderr, "choysum test: created postgres test database %s (keep=%v)\n", testDBName, keep)

		dbCopy.Dialect = "postgres"
		dbCopy.DSN = testDSN
		if !keep {
			cleanupDB = func() { _ = dropPostgresDatabase(ctx, adminDSN, testDBName) }
		}
	default:
		_ = os.RemoveAll(runRoot)
		return nil, func() {}, xfmt.Errorf("unsupported --db %s (supported: sqlite, postgres)", dbDialect)
	}

	scopeInput := newTestRuntimeScopeInputFromScope(baseScope, dbCopy).withDistPath(isolatedDist)
	if runHome := testingpathing.EffectiveCLITestRunHome(ctx); runHome != "" {
		scopeInput = scopeInput.withDefaultChoysumPath(runHome)
	}
	testScope := scope.NewScope(baseScope.Context(), scopeInput, unitTestRuntimeLogger(baseScope.Logger()))
	if testScope == nil {
		if cleanupDB != nil {
			cleanupDB()
		}
		_ = os.RemoveAll(runRoot)
		return nil, func() {}, xfmt.Errorf("failed to initialize test scope")
	}

	cleanupDist := func() {}
	if !keep {
		cleanupDist = func() { _ = os.RemoveAll(runRoot) }
	} else {
		fmt.Fprintf(os.Stderr, "choysum test: kept unit run dir: %s\n", runRoot)
		if runHome := testingpathing.EffectiveCLITestRunHome(ctx); runHome != "" {
			fmt.Fprintf(os.Stderr, "choysum test: kept unit shared home: %s\n", runHome)
		}
	}
	return testScope, joinCleanups(cleanupDB, cleanupDist), nil
}

func makePostgresTestDBName(app string) string {
	// Postgres identifiers are limited to 63 bytes.
	// We keep a stable prefix and a timestamp suffix for uniqueness.
	safe := strings.ToLower(strings.TrimSpace(app))
	if safe == "" {
		safe = "app"
	}
	var b strings.Builder
	for _, r := range safe {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteByte('_')
			continue
		}
		b.WriteByte('_')
	}
	namePart := strings.Trim(b.String(), "_")
	if namePart == "" {
		namePart = "app"
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	base := "choysum_test_" + namePart + "_" + suffix
	if len(base) <= 63 {
		return base
	}
	over := len(base) - 63
	if over >= len(namePart) {
		namePart = "app"
	} else {
		namePart = namePart[:len(namePart)-over]
		namePart = strings.Trim(namePart, "_")
		if namePart == "" {
			namePart = "app"
		}
	}
	base = "choysum_test_" + namePart + "_" + suffix
	if len(base) > 63 {
		base = base[:63]
	}
	return base
}

func quotePostgresIdent(ident string) string {
	// Defensive quoting; our generated names are already safe.
	return "\"" + strings.ReplaceAll(ident, "\"", "\"\"") + "\""
}

func setPostgresDatabaseInDSN(dsn string, dbName string) (string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", xfmt.Errorf("empty dsn")
	}
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		u.Path = "/" + dbName
		return u.String(), nil
	}

	tokens := splitPostgresKVDSN(dsn)
	found := false
	for i, t := range tokens {
		k, _, ok := strings.Cut(t, "=")
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		switch key {
		case "dbname", "database":
			tokens[i] = k + "=" + dbName
			found = true
		}
	}
	if !found {
		tokens = append(tokens, "dbname="+dbName)
	}
	return strings.Join(tokens, " "), nil
}

func splitPostgresKVDSN(dsn string) []string {
	// Split on spaces, keeping quoted values together.
	var out []string
	var b strings.Builder
	inQuote := false
	var quote rune
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range dsn {
		switch {
		case (r == '\'' || r == '"') && !inQuote:
			inQuote = true
			quote = r
			b.WriteRune(r)
		case inQuote && r == quote:
			inQuote = false
			b.WriteRune(r)
		case !inQuote && unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}

func dropPostgresDatabase(ctx context.Context, adminDSN string, dbName string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(adminDSN) == "" {
		return nil
	}
	if !strings.HasPrefix(dbName, "choysum_test_") {
		return xfmt.Errorf("refuse to drop non-test database: %s", dbName)
	}
	adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		return err
	}
	sqlDB, _ := adminDB.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}
	_ = adminDB.WithContext(ctx).Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()", dbName).Error
	return adminDB.WithContext(ctx).Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", quotePostgresIdent(dbName))).Error
}

func buildAppBundle(ctx context.Context, runtimeScope scope.Scope, jsExec jsexecutor.JsExecutor, app string, outFileName string, globalName string, entryPoint string) error {
	st, err := os.Stat(entryPoint)
	if err != nil || st.IsDir() {
		return xfmt.Errorf("entry point not found: %s", entryPoint)
	}

	mod := &meta.Module{
		Name:              app,
		ApplicationStr:    app,
		Path:              filepath.Join(runtimeOptionsFromScope(runtimeScope).modulesPath, app),
		ServiceEntryPoint: entryPoint,
	}

	b := newBackendBuilderHook(runtimeScope, jsExec, mod, entryPoint, outFileName, globalName)

	// Unit-test hardening: when bundleMode=bundle, runtime scripts are expected under dist/bundles.
	// Write test-only bundle (tests.js) to dist/bundles so the QuickJS loader can stay mode-correct.
	mode := strings.ToLower(strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).compileBundleMode))
	if mode == "" {
		mode = "bundle"
	}
	if mode == "bundle" && strings.EqualFold(strings.TrimSpace(outFileName), "tests.js") {
		if bundlerToDir, ok := b.(module.BundlerToDir); ok {
			if _, err := bundlerToDir.BundleToDirCtx(ctx, config.BundlesDir(runtimeOptionsFromScope(runtimeScope).distPath)); err != nil {
				return xfmt.Errorf("bundle %s (%s) to bundles dir: %w", app, outFileName, err)
			}
			return nil
		}
		return xfmt.Errorf("backend builder does not support BundleToDirCtx (required for bundleMode=bundle unit tests)")
	}
	if bundler, ok := b.(module.Bundler); ok {
		if _, err := bundler.Bundle(); err != nil {
			return xfmt.Errorf("bundle %s (%s): %w", app, outFileName, err)
		}
		return nil
	}
	builder, ok := b.(module.Builder)
	if !ok {
		return xfmt.Errorf("backend builder does not support Build")
	}
	if _, err := builder.Build(); err != nil {
		return xfmt.Errorf("build %s (%s): %w", app, outFileName, err)
	}
	return nil
}

func newCompilerExecutor(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
	// Create a compiler executor like install/upgrade.
	jsExec, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		return nil, xfmt.Errorf("create compiler executor: %w", err)
	}
	if err := jsExec.Start(); err != nil {
		return nil, xfmt.Errorf("start compiler executor: %w", err)
	}
	return jsExec, nil
}

func newRuntimeExecutor(runtimeScope scope.Scope, authenticator auth.Authenticator) (jsexecutor.JsExecutor, error) {
	jsExec, err := jsexecutor.NewRuntimeExecutor(runtimeScope, authenticator)
	if err != nil {
		return nil, xfmt.Errorf("create runtime executor: %w", err)
	}
	return jsExec, nil
}

type moduleManifest struct {
	EntryPoints map[string]string `json:"entryPoints"`
}

func resolveServiceEntryPoint(runtimeScope scope.Scope, app string) (string, error) {
	modulesPath := runtimeOptionsFromScope(runtimeScope).modulesPath
	manifestPath := filepath.Join(modulesPath, app, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err == nil {
		var m moduleManifest
		if err := json.Unmarshal(raw, &m); err == nil {
			if m.EntryPoints != nil {
				if rel := strings.TrimSpace(m.EntryPoints["service"]); rel != "" {
					p := rel
					if !filepath.IsAbs(p) {
						p = filepath.Join(modulesPath, app, rel)
					}
					return p, nil
				}
			}
		}
	}

	// Fallback to convention.
	fallback := filepath.Join(modulesPath, app, "service", "index.ts")
	if st, err2 := os.Stat(fallback); err2 == nil && !st.IsDir() {
		return fallback, nil
	}
	if err != nil {
		return "", xfmt.Errorf("manifest not found for %s: %w", app, err)
	}
	return "", xfmt.Errorf("service entry point not found for %s", app)
}

func writeTAP(w *os.File, report parsedReport) {
	// TAP 13; print plan at end so fail-fast remains valid.
	fmt.Fprintln(w, "TAP version 13")

	for i, c := range report.cases {
		n := i + 1
		if c.ok {
			fmt.Fprintf(w, "ok %d - %s\n", n, c.name)
			continue
		}
		fmt.Fprintf(w, "not ok %d - %s\n", n, c.name)
		if strings.TrimSpace(c.errMsg) != "" {
			for _, line := range strings.Split(c.errMsg, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				fmt.Fprintf(w, "# %s\n", line)
			}
		}
		if strings.TrimSpace(c.errStack) != "" {
			for _, line := range strings.Split(c.errStack, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				fmt.Fprintf(w, "# %s\n", line)
			}
		}
	}

	fmt.Fprintf(w, "1..%d\n", len(report.cases))
}

func writeJUnitIfNeeded(app string, report parsedReport, junitPath string) (bool, error) {
	if strings.TrimSpace(junitPath) == "" {
		return report.failed > 0, nil
	}

	suite := junitTestSuite{Name: app, Tests: report.total, Failures: report.failed, Errors: 0, Skipped: 0, Time: fmt.Sprintf("%.3f", report.totalTimeSeconds())}
	for _, c := range report.cases {
		jc := junitTestCase{ClassName: app, Name: c.name, Time: fmt.Sprintf("%.3f", float64(c.durationMs)/1000.0)}
		if !c.ok {
			msg := "failed"
			body := ""
			if c.errMsg != "" {
				msg = c.errMsg
			}
			if c.errStack != "" {
				body = c.errStack
			}
			jc.Failure = &junitFailure{Message: msg, Body: body}
		}
		suite.TestCases = append(suite.TestCases, jc)
	}

	doc := junitTestSuites{
		Name:       app,
		Tests:      suite.Tests,
		Failures:   suite.Failures,
		Errors:     0,
		Time:       suite.Time,
		TestSuites: []junitTestSuite{suite},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return true, xfmt.Errorf("marshal junit: %w", err)
	}
	out = append([]byte(xml.Header), out...)
	if dir := filepath.Dir(junitPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return true, xfmt.Errorf("mkdir junit dir: %w", err)
		}
	}
	if err := os.WriteFile(junitPath, out, 0o644); err != nil {
		return true, xfmt.Errorf("write junit: %w", err)
	}

	return report.failed > 0, nil
}

type parsedReport struct {
	total  int
	passed int
	failed int
	cases  []parsedCase
	// coverageJSON is JSON.stringify(globalThis.__coverage__).
	// Returned as a string (instead of a huge nested object) to avoid QuickJS
	// runtime assertion failures during teardown.
	coverageJSON string
}

type parsedCase struct {
	name       string
	ok         bool
	durationMs int
	errMsg     string
	errStack   string
}

func (r parsedReport) totalTimeSeconds() float64 {
	ms := 0
	for _, c := range r.cases {
		ms += c.durationMs
	}
	return float64(ms) / 1000.0
}

func parseReport(result any) (parsedReport, error) {
	m, ok := result.(map[string]any)
	if !ok {
		return parsedReport{}, xfmt.Errorf("unexpected result type %T", result)
	}

	getInt := func(key string) int {
		v := m[key]
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		default:
			return 0
		}
	}

	rep := parsedReport{
		total:  getInt("total"),
		passed: getInt("passed"),
		failed: getInt("failed"),
	}
	if s, ok := m["coverageJSON"].(string); ok {
		rep.coverageJSON = s
	}

	casesVal, _ := m["cases"]
	casesSlice, _ := casesVal.([]any)
	for _, item := range casesSlice {
		cm, _ := item.(map[string]any)
		pc := parsedCase{}
		if v, ok := cm["name"].(string); ok {
			pc.name = v
		}
		if v, ok := cm["ok"].(bool); ok {
			pc.ok = v
		}
		switch t := cm["durationMs"].(type) {
		case float64:
			pc.durationMs = int(t)
		case int:
			pc.durationMs = t
		case int64:
			pc.durationMs = int(t)
		}
		if errObj, ok := cm["error"].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok {
				pc.errMsg = msg
			}
			if st, ok := errObj["stack"].(string); ok {
				pc.errStack = st
			}
		}
		rep.cases = append(rep.cases, pc)
	}

	return rep, nil
}
