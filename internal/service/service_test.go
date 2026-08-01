// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"go.opentelemetry.io/otel/baggage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type helperScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
	props   *[]scope.Propagation
}

type serviceTestTransaction struct {
	ctx     context.Context
	session *scope.Session
}

type serviceTestTransactor struct {
	runtimeScope scope.Scope
	run          func(func(scope.Scope) error) error
	props        *[]scope.Propagation
}

func (e *helperScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *helperScope) Transactor() scope.Transactor {
	return &serviceTestTransactor{runtimeScope: e, run: e.Run, props: e.props}
}
func (e *helperScope) Session() *scope.Session { return e.session }
func (e *helperScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *helperScope) Context() context.Context { return e.ctx }
func (e *helperScope) Logger() *slog.Logger     { return e.logger }
func (e *helperScope) Config() *config.Config   { return e.cfg }

func (e *helperScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type controlledScope struct {
	*helperScope
	runFn func(fn func(runtimeScope scope.Scope) error) error
}

func (e *controlledScope) Run(fn func(runtimeScope scope.Scope) error) error {
	if e.runFn != nil {
		return e.runFn(fn)
	}
	return fn(e)
}

func (e *controlledScope) Transactor() scope.Transactor {
	return &serviceTestTransactor{runtimeScope: e, run: e.Run, props: e.helperScope.props}
}

func (e *controlledScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	base := *e.helperScope
	base.ctx = ctx
	clone.helperScope = &base
	return &clone
}

func (u *serviceTestTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if opts.Propagation == "" {
		opts.Propagation = scope.PropagationRequired
	}
	if u.props != nil {
		*u.props = append(*u.props, opts.Propagation)
	}
	txCtx := ctx
	if txCtx == nil {
		txCtx = u.runtimeScope.Context()
	}
	txScope := u.runtimeScope.WithContext(txCtx)
	tx := &serviceTestTransaction{session: txScope.Session()}
	tx.ctx = scope.ContextWithTransaction(txCtx, tx)
	invoke := func() error {
		return fn(txScope.WithContext(tx.ctx), tx)
	}
	if u.run != nil {
		return u.run(func(scope.Scope) error { return invoke() })
	}
	return invoke()
}

func (u *serviceTestTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (u *serviceTestTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (u *serviceTestTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (tx *serviceTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *serviceTestTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *serviceTestTransaction) Savepoint(string) error           { return nil }
func (tx *serviceTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *serviceTestTransaction) ReleaseSavepoint(string) error    { return nil }

type testIdentity struct {
	userID  string
	tokenID string
	meta    map[string]any
}

func (i *testIdentity) GetUserID() string           { return i.userID }
func (i *testIdentity) GetTokenID() string          { return i.tokenID }
func (i *testIdentity) GetMetadata() map[string]any { return i.meta }
func (i *testIdentity) IsValid() bool               { return true }

type aclTestEngine struct {
	execute func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error)
}

func (e *aclTestEngine) Load(_ []*jsengine.JsScript) error { return nil }

func (e *aclTestEngine) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.execute != nil {
		return e.execute(ctx, req)
	}
	return &jsengine.JsResponse{Id: req.Id, Result: true}, nil
}

func (e *aclTestEngine) Close() error { return nil }

// stubJsExecutor is a minimal JsExecutor that returns Execute results verbatim
// (unlike RuntimeExecutor, which always materializes a Routing object).
type stubJsExecutor struct {
	execute func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error)
}

func (e *stubJsExecutor) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.execute != nil {
		return e.execute(ctx, req)
	}
	return &jsengine.JsResponse{Id: req.Id, Result: true}, nil
}
func (e *stubJsExecutor) GetJsScripts() []*jsengine.JsScript      { return nil }
func (e *stubJsExecutor) SetJsScripts(_ []*jsengine.JsScript)     {}
func (e *stubJsExecutor) Reload(_ ...*jsengine.JsScript) error    { return nil }
func (e *stubJsExecutor) AppendJsScripts(_ ...*jsengine.JsScript) {}
func (e *stubJsExecutor) Start() error                            { return nil }
func (e *stubJsExecutor) Stop() error                             { return nil }

func newHelperScope(distPath string) *helperScope {
	return &helperScope{
		ctx: context.Background(),
		cfg: &config.Config{
			DistPath: distPath,
			Compile:  &config.CompileConfig{BundleMode: string(config.BundleModeBundle)},
			Server:   config.NewDefaultServerConfig(),
			Auth:     config.NewDefaultAuthConfig(),
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newACLTestExecutor(t *testing.T, runtimeScope scope.Scope, execute func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error)) jsexecutor.JsExecutor {
	t.Helper()
	executor, err := jsexecutor.NewRuntimeExecutor(runtimeScope, nil, jsexecutor.WithJsEngine(func() (jsengine.JsEngine, error) {
		return &aclTestEngine{execute: execute}, nil
	}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	if err := executor.Start(); err != nil {
		t.Fatalf("start executor: %v", err)
	}
	t.Cleanup(func() {
		if err := executor.Stop(); err != nil {
			t.Fatalf("stop executor: %v", err)
		}
	})
	return executor
}

func TestServiceOptionsAndHelperFunctions(t *testing.T) {
	runtimeScope := newHelperScope(t.TempDir())
	svc := &ApplicationService{runtimeScope: runtimeScope, name: "auth"}

	called := false
	WithHasGrpcMethod(func(fullMethod string) bool {
		called = fullMethod == "/auth.User/Login"
		return called
	})(svc)
	WithBundleMode("application")(svc)

	if svc.Name() != "auth" || svc.bundleMode != "application" {
		t.Fatalf("unexpected service identity/config: name=%q mode=%q", svc.Name(), svc.bundleMode)
	}
	if !svc.hasGrpcMethod("/auth.User/Login") || !called {
		t.Fatal("expected WithHasGrpcMethod to store callback")
	}

	if meta, ok := getJsReqMeta(map[string]interface{}{"req": map[string]any{"depth": 2}}); !ok || meta["depth"] != 2 {
		t.Fatalf("unexpected getJsReqMeta result: meta=%#v ok=%v", meta, ok)
	}
	if _, ok := getJsReqMeta(map[string]interface{}{"req": "bad"}); ok {
		t.Fatal("expected getJsReqMeta to reject non-map req payload")
	}
	if got := getJsReqDepth(map[string]any{"depth": 3}); got != 3 {
		t.Fatalf("getJsReqDepth() = %d, want 3", got)
	}
	for _, reqMeta := range []map[string]any{nil, {"depth": -1}, {"depth": "3"}} {
		if got := getJsReqDepth(reqMeta); got != 0 {
			t.Fatalf("expected invalid depth to fall back to 0, got %d for %#v", got, reqMeta)
		}
	}
}

func TestHandleErrorVariants(t *testing.T) {
	var logBuf bytes.Buffer
	runtimeScope := newHelperScope(t.TempDir())
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	svc := &ApplicationService{runtimeScope: runtimeScope}

	if result, err := svc.handleError(nil); result != nil || err != nil {
		t.Fatalf("handleError(nil) = %#v, %v", result, err)
	}
	if _, err := svc.handleError(context.Canceled); status.Code(err) != codes.Canceled {
		t.Fatalf("context.Canceled mapped to %v, want %v", status.Code(err), codes.Canceled)
	}

	existing := status.Error(codes.NotFound, "missing")
	if _, err := svc.handleError(existing); status.Code(err) != codes.NotFound || status.Convert(err).Message() != "missing" {
		t.Fatalf("existing status error changed unexpectedly: %v", err)
	}

	valid := oerrors.New("auth", "DENY", "no access").WithGrpcCode(codes.PermissionDenied)
	_, err := svc.handleError(valid)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied status, got %v", status.Code(err))
	}
	if len(status.Convert(err).Details()) != 1 {
		t.Fatalf("expected error details to be attached, got %#v", status.Convert(err).Details())
	}

	invalid := oerrors.New("auth", "BROKEN", "fallback")
	invalid.GrpcCode = 0
	_, err = svc.handleError(invalid)
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "fallback" {
		t.Fatalf("expected invalid grpc code to fall back to internal, got %v", err)
	}

	_, err = svc.handleError(errors.New("plain failure"))
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "plain failure" {
		t.Fatalf("expected plain error to map to internal, got %v", err)
	}
}

func TestEmitTaskMetric(t *testing.T) {
	var nilSvc *ApplicationService
	nilSvc.emitTaskMetric("task_inflight", map[string]any{"count": 1})

	runtimeScope := newHelperScope(t.TempDir())
	fields := map[string]any{"count": 2}
	svc := &ApplicationService{runtimeScope: runtimeScope}

	svc.emitTaskMetric("task_inflight", fields)

	if fields["metric"] != "task_inflight" {
		t.Fatalf("expected metric field to be injected, got %#v", fields)
	}
}

func TestBuildJsContext_IgnoresBaggageCompanyScope(t *testing.T) {
	ctx := context.Background()

	id := &testIdentity{
		userID:  "u1",
		tokenID: "t1",
		meta: map[string]any{
			"allowedCompanyIds": []string{"A", "B", "C"},
			"activeCompanyId":   "A",
			"enabledCompanyIds": []string{"A", "B"},
		},
	}
	ctx = auth.ContextWithIdentity(ctx, id)

	mActive, err := baggage.NewMember("ctx.activeCompanyId", "C")
	if err != nil {
		t.Fatalf("new member activeCompanyId: %v", err)
	}
	mEnabled, err := baggage.NewMember("ctx.enabledCompanyIds", "C")
	if err != nil {
		t.Fatalf("new member enabledCompanyIds: %v", err)
	}
	mLang, err := baggage.NewMember("ctx.lang", "zh")
	if err != nil {
		t.Fatalf("new member lang: %v", err)
	}
	bag, err := baggage.New(mActive, mEnabled, mLang)
	if err != nil {
		t.Fatalf("new baggage: %v", err)
	}
	ctx = baggage.ContextWithBaggage(ctx, bag)

	s := &ApplicationService{runtimeScope: &helperScope{ctx: ctx, logger: slog.Default()}}
	jsCtx := s.buildJsContext(ctx)

	ctxMap, ok := jsCtx["ctx"].(map[string]any)
	if !ok {
		t.Fatalf("jsCtx.ctx missing or wrong type")
	}

	if got := ctxMap["activeCompanyId"]; got != "A" {
		t.Fatalf("activeCompanyId mismatch: got=%v want=%v", got, "A")
	}

	gotEnabled, ok := ctxMap["enabledCompanyIds"].([]string)
	if !ok {
		t.Fatalf("enabledCompanyIds missing or wrong type: %T", ctxMap["enabledCompanyIds"])
	}
	if len(gotEnabled) != 2 || gotEnabled[0] != "A" || gotEnabled[1] != "B" {
		t.Fatalf("enabledCompanyIds mismatch: got=%v want=[A B]", gotEnabled)
	}

	if got := ctxMap["lang"]; got != "zh" {
		t.Fatalf("lang should still be allowed from baggage: got=%v want=%v", got, "zh")
	}
}

func TestBuildJsContext_TimezoneFallback(t *testing.T) {
	newCtxWithBaggage := func(meta map[string]any, tzBaggage string) context.Context {
		ctx := context.Background()
		if meta != nil {
			ctx = auth.ContextWithIdentity(ctx, &testIdentity{userID: "u1", tokenID: "t1", meta: meta})
		}
		if tzBaggage != "" {
			m, err := baggage.NewMember("ctx.tz", tzBaggage)
			if err != nil {
				t.Fatalf("new member tz: %v", err)
			}
			bag, err := baggage.New(m)
			if err != nil {
				t.Fatalf("new baggage: %v", err)
			}
			ctx = baggage.ContextWithBaggage(ctx, bag)
		}
		return ctx
	}

	s := &ApplicationService{runtimeScope: &helperScope{ctx: context.Background(), logger: slog.Default()}}

	t.Run("user timezone wins over baggage", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"timezone":          "America/New_York",
			"companyTimezone":   "Asia/Shanghai",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "Europe/Paris")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "America/New_York" {
			t.Fatalf("tz mismatch: got=%v want=America/New_York", got)
		}
		if got := ctxMap["companyTz"]; got != "Asia/Shanghai" {
			t.Fatalf("companyTz mismatch: got=%v want=Asia/Shanghai", got)
		}
	})

	t.Run("empty user uses baggage then company", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"companyTimezone":   "Asia/Tokyo",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "Europe/Berlin")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "Europe/Berlin" {
			t.Fatalf("tz mismatch: got=%v want=Europe/Berlin", got)
		}
		if got := ctxMap["clientTz"]; got != "Europe/Berlin" {
			t.Fatalf("clientTz mismatch: got=%v want=Europe/Berlin", got)
		}
		if got := ctxMap["companyTz"]; got != "Asia/Tokyo" {
			t.Fatalf("companyTz mismatch: got=%v want=Asia/Tokyo", got)
		}
	})

	t.Run("empty user and baggage falls back to company then UTC", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"companyTimezone":   "Asia/Shanghai",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "Asia/Shanghai" {
			t.Fatalf("tz mismatch: got=%v want=Asia/Shanghai", got)
		}
	})

	t.Run("invalid baggage ignored", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"companyTimezone":   "UTC",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "Not/A_Zone")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "UTC" {
			t.Fatalf("tz mismatch: got=%v want=UTC", got)
		}
		if _, ok := ctxMap["clientTz"]; ok {
			t.Fatalf("clientTz should be absent for invalid baggage")
		}
	})

	t.Run("invalid preferred meta alias falls through to valid alias", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"tz":                "Not/A_Zone",
			"timezone":          "Europe/Berlin",
			"companyTimezone":   "Local",
			"companyTz":         "Asia/Tokyo",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "Europe/Berlin" {
			t.Fatalf("tz mismatch: got=%v want=Europe/Berlin", got)
		}
		if got := ctxMap["companyTz"]; got != "Asia/Tokyo" {
			t.Fatalf("companyTz mismatch: got=%v want=Asia/Tokyo", got)
		}
	})

	t.Run("Local is rejected as IANA timezone", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"timezone":          "Local",
			"companyTimezone":   "Local",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		// No valid user/company IANA → UTC fallback for display tz.
		if got := ctxMap["tz"]; got != "UTC" {
			t.Fatalf("tz mismatch: got=%v want=UTC", got)
		}
		if got, ok := ctxMap["companyTz"]; ok && got != nil && got != "" {
			t.Fatalf("companyTz should be empty when only Local is present: got=%v", got)
		}
	})

	t.Run("baggage sets clientTz even when user tz wins", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"timezone":          "America/New_York",
			"companyTimezone":   "Asia/Shanghai",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "Europe/Paris")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "America/New_York" {
			t.Fatalf("tz mismatch: got=%v want=America/New_York", got)
		}
		if got := ctxMap["clientTz"]; got != "Europe/Paris" {
			t.Fatalf("clientTz mismatch: got=%v want=Europe/Paris", got)
		}
	})

	t.Run("no identity defaults to UTC", func(t *testing.T) {
		jsCtx := s.buildJsContext(context.Background())
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "UTC" {
			t.Fatalf("tz mismatch: got=%v want=UTC", got)
		}
		if _, ok := ctxMap["clientTz"]; ok {
			t.Fatalf("clientTz should be absent without baggage")
		}
	})

	t.Run("meta tz alias preferred over timezone", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"tz":                "Asia/Tokyo",
			"timezone":          "America/New_York",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, "")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["tz"]; got != "Asia/Tokyo" {
			t.Fatalf("tz mismatch: got=%v want=Asia/Tokyo", got)
		}
	})

	t.Run("activeCompanyTimezone alias fills companyTz", func(t *testing.T) {
		ctx := newCtxWithBaggage(map[string]any{
			"activeCompanyTimezone": "Europe/Paris",
			"activeCompanyId":       "A",
			"allowedCompanyIds":     []string{"A"},
		}, "")
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if got := ctxMap["companyTz"]; got != "Europe/Paris" {
			t.Fatalf("companyTz mismatch: got=%v want=Europe/Paris", got)
		}
		if got := ctxMap["tz"]; got != "Europe/Paris" {
			t.Fatalf("tz should fall through to company: got=%v", got)
		}
	})

	t.Run("oversized baggage tz is ignored", func(t *testing.T) {
		long := strings.Repeat("a", 65)
		ctx := newCtxWithBaggage(map[string]any{
			"companyTimezone":   "UTC",
			"activeCompanyId":   "A",
			"allowedCompanyIds": []string{"A"},
		}, long)
		jsCtx := s.buildJsContext(ctx)
		ctxMap := jsCtx["ctx"].(map[string]any)
		if _, ok := ctxMap["clientTz"]; ok {
			t.Fatalf("clientTz should be absent for oversized baggage")
		}
		if got := ctxMap["tz"]; got != "UTC" {
			t.Fatalf("tz mismatch: got=%v want=UTC", got)
		}
	})
}

func TestNormalizeIANATimezone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		ok   bool
		want string
	}{
		{name: "empty", in: "", ok: false},
		{name: "whitespace", in: "   ", ok: false},
		{name: "too long", in: strings.Repeat("A", 65), ok: false},
		{name: "Local", in: "Local", ok: false},
		{name: "local fold", in: "local", ok: false},
		{name: "invalid", in: "Not/A_Zone", ok: false},
		{name: "UTC", in: "UTC", ok: true, want: "UTC"},
		{name: "trim", in: "  Europe/Berlin  ", ok: true, want: "Europe/Berlin"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := normalizeIANATimezone(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok mismatch for %q: got=%v want=%v", tc.in, ok, tc.ok)
			}
			if tc.ok && got != tc.want {
				t.Fatalf("value mismatch for %q: got=%q want=%q", tc.in, got, tc.want)
			}
			if !tc.ok && got != "" {
				t.Fatalf("expected empty value on failure for %q, got %q", tc.in, got)
			}
		})
	}
}

func TestServiceCodec(t *testing.T) {
	structMsg, err := structpb.NewStruct(map[string]any{"name": "choysum", "count": 2})
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}
	anyValue, err := serviceCodec.messageToAny(structMsg.ProtoReflect())
	if err != nil {
		t.Fatalf("messageToAny() error = %v", err)
	}
	convertedMap, ok := anyValue.(map[string]interface{})
	if !ok || convertedMap["name"] != "choysum" || convertedMap["count"] != float64(2) {
		t.Fatalf("unexpected messageToAny result: %#v", anyValue)
	}

	structDesc := structMsg.ProtoReflect().Descriptor()
	dynStruct := serviceCodec.newMessage(structDesc)
	if err := serviceCodec.anyToMessage(map[string]interface{}{"enabled": true}, dynStruct); err != nil {
		t.Fatalf("anyToMessage() error = %v", err)
	}
	convertedStruct, err := serviceCodec.messageToAny(dynStruct.ProtoReflect())
	if err != nil {
		t.Fatalf("messageToAny(dynamic struct) error = %v", err)
	}
	if convertedStruct.(map[string]interface{})["enabled"] != true {
		t.Fatalf("unexpected dynamic struct conversion: %#v", convertedStruct)
	}

	listDesc := (&structpb.ListValue{}).ProtoReflect().Descriptor()
	dynList := serviceCodec.newMessage(listDesc)
	if err := serviceCodec.sliceToMessage([]interface{}{"a", 2.0, true}, dynList); err != nil {
		t.Fatalf("sliceToMessage() error = %v", err)
	}
	convertedList, err := serviceCodec.messageToAny(dynList.ProtoReflect())
	if err != nil {
		t.Fatalf("messageToAny(dynamic list) error = %v", err)
	}
	listValues, ok := convertedList.([]interface{})
	if !ok || len(listValues) != 3 || listValues[0] != "a" || listValues[1] != float64(2) || listValues[2] != true {
		t.Fatalf("unexpected list conversion result: %#v", convertedList)
	}

	errorInfoDesc := (&oerrors.ErrorInfo{}).ProtoReflect().Descriptor()
	secondValue := serviceCodec.newMessage(errorInfoDesc)
	if err := serviceCodec.mapToMessage(map[string]interface{}{"domain": "svc", "code": "E001", "message": "boom"}, secondValue); err != nil {
		t.Fatalf("mapToMessage() error = %v", err)
	}
	convertedSecondValue, err := serviceCodec.messageToAny(secondValue.ProtoReflect())
	if err != nil {
		t.Fatalf("messageToAny(second dynamic value) error = %v", err)
	}
	convertedErrorInfo, ok := convertedSecondValue.(map[string]interface{})
	if !ok || convertedErrorInfo["domain"] != "svc" || convertedErrorInfo["code"] != "E001" || convertedErrorInfo["message"] != "boom" {
		t.Fatalf("unexpected mapToMessage conversion result: %#v", convertedSecondValue)
	}

	valueDesc := (&structpb.Value{}).ProtoReflect().Descriptor().Fields().ByName("string_value")
	protoValue, err := serviceCodec.convertToProtoValue("hello", valueDesc)
	if err != nil {
		t.Fatalf("convertToProtoValue() error = %v", err)
	}
	if protoValue.String() != "hello" {
		t.Fatalf("unexpected proto value: %#v", protoValue)
	}
}

func TestServiceScriptsAndWebHandlers(t *testing.T) {
	distDir := t.TempDir()
	appDir := filepath.Join(distDir, "apps", "auth")
	webDir := filepath.Join(distDir, "web")
	assetsDir := filepath.Join(webDir, "assets")
	for _, dir := range []string{appDir, assetsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(appDir, "index.js"), []byte("console.log('auth')"), 0o644); err != nil {
		t.Fatalf("write app script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "about.html"), []byte("about"), 0o644); err != nil {
		t.Fatalf("write about.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	runtimeScope := newHelperScope(distDir)
	runtimeScope.cfg.Server.WebBaseURL = "/web"
	runtimeScope.cfg.Server.RootRedirectURL = "/portal"

	serviceSvc := &ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: appDir}
	scripts := serviceSvc.ServiceScripts()
	if len(scripts) != 1 || scripts[0].FileName != filepath.Join(appDir, "index.js") || scripts[0].Content != "console.log('auth')" {
		t.Fatalf("unexpected service scripts: %#v", scripts)
	}
	if webScripts := (&ApplicationService{runtimeScope: runtimeScope, name: "web", appDistPath: webDir}).ServiceScripts(); webScripts != nil {
		t.Fatalf("expected web service scripts to be nil, got %#v", webScripts)
	}
	if missingScripts := (&ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: filepath.Join(distDir, "apps", "missing")}).ServiceScripts(); missingScripts != nil {
		t.Fatalf("expected missing script path to return nil, got %#v", missingScripts)
	}

	webSvc := &ApplicationService{runtimeScope: runtimeScope, name: "web", appDistPath: webDir}
	handlers, err := webSvc.WebHandlers()
	if err != nil {
		t.Fatalf("WebHandlers() error = %v", err)
	}
	if len(handlers) != 2 {
		t.Fatalf("unexpected handler map: %#v", handlers)
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/web/assets/app.js", nil)
	assetRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(assetRR, assetReq)
	if assetRR.Code != http.StatusOK || assetRR.Body.String() != "asset" {
		t.Fatalf("unexpected asset handler response: code=%d body=%q", assetRR.Code, assetRR.Body.String())
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/web/about.html", nil)
	pageRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(pageRR, pageReq)
	if pageRR.Code != http.StatusOK || pageRR.Body.String() != "about" {
		t.Fatalf("unexpected static page response: code=%d body=%q", pageRR.Code, pageRR.Body.String())
	}

	spaReq := httptest.NewRequest(http.MethodGet, "/web/dashboard", nil)
	spaRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(spaRR, spaReq)
	if spaRR.Code != http.StatusOK || spaRR.Body.String() != "<html>spa</html>" {
		t.Fatalf("unexpected spa fallback response: code=%d body=%q", spaRR.Code, spaRR.Body.String())
	}

	secretPath := filepath.Join(distDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("super-secret"), 0o644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}
	traversalReq := httptest.NewRequest(http.MethodGet, "/web/../secret.txt", nil)
	traversalRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(traversalRR, traversalReq)
	if traversalRR.Code != http.StatusOK || traversalRR.Body.String() != "<html>spa</html>" {
		t.Fatalf("unexpected traversal response: code=%d body=%q", traversalRR.Code, traversalRR.Body.String())
	}

	dirReq := httptest.NewRequest(http.MethodGet, "/web/assets", nil)
	dirRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(dirRR, dirReq)
	if dirRR.Code != http.StatusOK || dirRR.Body.String() != "<html>spa</html>" {
		t.Fatalf("unexpected directory fallback response: code=%d body=%q", dirRR.Code, dirRR.Body.String())
	}

	rootWebReq := httptest.NewRequest(http.MethodGet, "/web/", nil)
	rootWebRR := httptest.NewRecorder()
	handlers["/web/"].ServeHTTP(rootWebRR, rootWebReq)
	if rootWebRR.Code != http.StatusOK || rootWebRR.Body.String() != "<html>spa</html>" {
		t.Fatalf("unexpected web root response: code=%d body=%q", rootWebRR.Code, rootWebRR.Body.String())
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRR := httptest.NewRecorder()
	handlers["/"].ServeHTTP(rootRR, rootReq)
	if rootRR.Code != http.StatusFound || rootRR.Header().Get("Location") != "/portal/" {
		t.Fatalf("unexpected root redirect response: code=%d location=%q", rootRR.Code, rootRR.Header().Get("Location"))
	}

	redirectSvc := &ApplicationService{runtimeScope: runtimeScope, name: "web", appDistPath: filepath.Join(distDir, "empty-web")}
	if err := os.MkdirAll(redirectSvc.appDistPath, 0o755); err != nil {
		t.Fatalf("mkdir empty web dir: %v", err)
	}
	redirectRR := httptest.NewRecorder()
	redirectSvc.staticFileHandler(redirectSvc.appDistPath, "/web/").ServeHTTP(redirectRR, httptest.NewRequest(http.MethodGet, "/web/missing", nil))
	if redirectRR.Code != http.StatusFound || redirectRR.Header().Get("Location") != "/web/" {
		t.Fatalf("unexpected static handler redirect: code=%d location=%q", redirectRR.Code, redirectRR.Header().Get("Location"))
	}

	nonWebHandlers, err := (&ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: appDir}).WebHandlers()
	if err != nil || nonWebHandlers != nil {
		t.Fatalf("expected non-web service to have no handlers, got handlers=%#v err=%v", nonWebHandlers, err)
	}
}

func TestStaticFileHandlerLogsAssetRequestOutcomes(t *testing.T) {
	distDir := t.TempDir()
	webDir := filepath.Join(distDir, "web")
	assetsDir := filepath.Join(webDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", assetsDir, err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("asset"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	var logBuf bytes.Buffer
	runtimeScope := newHelperScope(distDir)
	runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := (&ApplicationService{runtimeScope: runtimeScope, name: "web", appDistPath: webDir}).staticFileHandler(webDir, "/web/")

	successRR := httptest.NewRecorder()
	handler.ServeHTTP(successRR, httptest.NewRequest(http.MethodGet, "/web/assets/app.js", nil))
	if successRR.Code != http.StatusOK || successRR.Body.String() != "asset" {
		t.Fatalf("unexpected asset handler response: code=%d body=%q", successRR.Code, successRR.Body.String())
	}
	if logs := logBuf.String(); !strings.Contains(logs, "web asset served") || !strings.Contains(logs, "status=200") || !strings.Contains(logs, "path=/web/assets/app.js") {
		t.Fatalf("expected successful asset request debug log, got %q", logs)
	}

	missingRR := httptest.NewRecorder()
	handler.ServeHTTP(missingRR, httptest.NewRequest(http.MethodGet, "/web/assets/missing.js", nil))
	if missingRR.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing asset response: code=%d body=%q", missingRR.Code, missingRR.Body.String())
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "web asset request failed") {
		t.Fatalf("expected failed asset request log, got %q", logs)
	}
	if !strings.Contains(logs, "status=404") || !strings.Contains(logs, "path=/web/assets/missing.js") {
		t.Fatalf("expected failed asset log attrs, got %q", logs)
	}
	if !strings.Contains(logs, "web asset served") {
		t.Fatalf("expected success asset debug log to remain present, got %q", logs)
	}
}

func TestSafeStaticPathRejectsParentRoot(t *testing.T) {
	webPath := filepath.Join(t.TempDir(), "web")
	if err := os.MkdirAll(webPath, 0o755); err != nil {
		t.Fatalf("mkdir web path: %v", err)
	}

	if got, ok := safeStaticPath(webPath, "/api/about.html", "/web/"); ok || got != "" {
		t.Fatalf("safeStaticPath(prefix mismatch) = (%q, %v), want (\"\", false)", got, ok)
	}

	if got, ok := safeStaticPath(webPath, "/web/about.html", "/web/"); !ok {
		t.Fatalf("safeStaticPath(valid) ok = %v, want true", ok)
	} else {
		want := filepath.Join(webPath, "about.html")
		gotEval, err := filepath.Abs(got)
		if err != nil {
			t.Fatalf("abs got path: %v", err)
		}
		wantEval, err := filepath.Abs(want)
		if err != nil {
			t.Fatalf("abs want path: %v", err)
		}
		if gotEval != wantEval {
			t.Fatalf("safeStaticPath(valid) = %q, want %q", gotEval, wantEval)
		}
	}

	if got, ok := safeStaticPath(webPath, "/web/..", "/web/"); ok || got != "" {
		t.Fatalf("safeStaticPath() = (%q, %v), want (\"\", false)", got, ok)
	}

	if got, ok := safeStaticPath(webPath, "/web/../../secret.txt", "/web/"); ok || got != "" {
		t.Fatalf("safeStaticPath(deeper traversal) = (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestNewApplicationServiceResolvesPaths(t *testing.T) {
	distDir := t.TempDir()
	authAPIProtoDir := config.APIAppProtoDir(distDir, "auth")
	for _, dir := range []string{
		filepath.Join(distDir, "web"),
		filepath.Join(distDir, "bundles"),
		filepath.Join(distDir, "apps", "auth"),
		authAPIProtoDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	runtimeScope := newHelperScope(distDir)

	bundleSvc, err := NewApplicationService(runtimeScope, "auth", nil, WithHasGrpcMethod(func(string) bool { return true }))
	if err != nil {
		t.Fatalf("NewApplicationService(bundle) error = %v", err)
	}
	if bundleSvc.appDistPath != filepath.Join(distDir, "bundles") || bundleSvc.protoRootDir != authAPIProtoDir || len(bundleSvc.protoImportPaths) != 1 || bundleSvc.protoImportPaths[0] != authAPIProtoDir {
		t.Fatalf("unexpected bundle service paths: %#v", bundleSvc)
	}
	if bundleSvc.hasGrpcMethod == nil {
		t.Fatalf("expected options to be applied on bundle service: %#v", bundleSvc)
	}

	appSvc, err := NewApplicationService(runtimeScope, "auth", nil, WithBundleMode("application"))
	if err != nil {
		t.Fatalf("NewApplicationService(application) error = %v", err)
	}
	if appSvc.appDistPath != filepath.Join(distDir, "apps", "auth") || appSvc.protoRootDir != authAPIProtoDir || appSvc.protoImportPaths[0] != authAPIProtoDir {
		t.Fatalf("unexpected application service paths: %#v", appSvc)
	}

	webSvc, err := NewApplicationService(runtimeScope, "web", nil)
	if err != nil {
		t.Fatalf("NewApplicationService(web) error = %v", err)
	}
	if webSvc.appDistPath != filepath.Join(distDir, "web") || webSvc.protoRootDir != "" || webSvc.protoImportPaths != nil {
		t.Fatalf("unexpected web service paths: %#v", webSvc)
	}
}

func TestBundleMode_ServiceDescs_LoadsOnlyTargetAppProto(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir dist/bundles: %v", err)
	}

	authProtoDir := config.APIAppProtoDir(distRoot, "auth")
	metaProtoDir := config.APIAppProtoDir(distRoot, "meta")
	if err := os.MkdirAll(authProtoDir, 0o755); err != nil {
		t.Fatalf("mkdir auth proto dir: %v", err)
	}
	if err := os.MkdirAll(metaProtoDir, 0o755); err != nil {
		t.Fatalf("mkdir meta proto dir: %v", err)
	}

	authProto := `syntax = "proto3";
package auth;

service AuthService {
  rpc Ping(PingRequest) returns (PingReply);
}

message PingRequest { string msg = 1; }
message PingReply { string msg = 1; }
`
	metaProto := `syntax = "proto3";
package meta;

service Service {
  rpc Ping(PingRequest) returns (PingReply);
}

message PingRequest { string msg = 1; }
message PingReply { string msg = 1; }
`

	if err := os.WriteFile(filepath.Join(authProtoDir, "auth.proto"), []byte(authProto), 0o644); err != nil {
		t.Fatalf("write auth.proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaProtoDir, "meta.proto"), []byte(metaProto), 0o644); err != nil {
		t.Fatalf("write meta.proto: %v", err)
	}

	runtimeScope := newHelperScope(distRoot)

	svc, err := NewApplicationService(runtimeScope, "auth", nil)
	if err != nil {
		t.Fatalf("NewApplicationService(auth): %v", err)
	}
	descs, err := svc.ServiceDescs()
	if err != nil {
		t.Fatalf("auth ServiceDescs: %v", err)
	}
	if len(descs) != 3 {
		t.Fatalf("expected 3 service desc for auth (AuthService+TaskWorker+I18n); got %d", len(descs))
	}
	var hasAuthService bool
	var hasTaskWorker bool
	var hasI18n bool
	for _, desc := range descs {
		if desc.ServiceName == "auth.AuthService" {
			hasAuthService = true
		}
		if desc.ServiceName == "auth.TaskWorker" {
			hasTaskWorker = true
		}
		if desc.ServiceName == "auth.I18n" {
			hasI18n = true
		}
		if strings.Contains(desc.ServiceName, "meta.") {
			t.Fatalf("auth desc should not include meta: %q", desc.ServiceName)
		}
	}
	if !hasAuthService {
		t.Fatalf("missing auth.AuthService descriptor")
	}
	if !hasTaskWorker {
		t.Fatalf("missing auth.TaskWorker descriptor")
	}
	if !hasI18n {
		t.Fatalf("missing auth.I18n descriptor")
	}

	metaSvc, err := NewApplicationService(runtimeScope, "meta", nil)
	if err != nil {
		t.Fatalf("NewApplicationService(meta): %v", err)
	}
	metaDescs, err := metaSvc.ServiceDescs()
	if err != nil {
		t.Fatalf("meta ServiceDescs: %v", err)
	}
	if len(metaDescs) != 3 {
		t.Fatalf("expected 3 service desc for meta (Service+TaskWorker+I18n); got %d", len(metaDescs))
	}
	var hasService bool
	var hasMetaTaskWorker bool
	var hasMetaI18n bool
	for _, desc := range metaDescs {
		if desc.ServiceName == "meta.MetaService" {
			hasService = true
		}
		if desc.ServiceName == "meta.TaskWorker" {
			hasMetaTaskWorker = true
		}
		if desc.ServiceName == "meta.I18n" {
			hasMetaI18n = true
		}
		if strings.Contains(desc.ServiceName, "auth.") {
			t.Fatalf("meta desc should not include auth: %q", desc.ServiceName)
		}
	}
	if !hasService {
		t.Fatalf("missing meta.MetaService descriptor")
	}
	if !hasMetaTaskWorker {
		t.Fatalf("missing meta.TaskWorker descriptor")
	}
	if !hasMetaI18n {
		t.Fatalf("missing meta.I18n descriptor")
	}
	if got := metaDescs[0].ServiceName; got != "meta.MetaService" {
		t.Fatalf("unexpected meta service name: got=%q want=%q", got, "meta.MetaService")
	}
}

func TestInjectMethodMetaAndEntryPolicy(t *testing.T) {
	runtimeScope := newHelperScope(t.TempDir())
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.GrpcCompanyFilter = false
	runtimeScope.cfg.Auth.GrpcRecordRule = true
	runtimeScope.cfg.Auth.GrpcFieldRule = true
	runtimeScope.cfg.Auth.GrpcEntryPolicy = map[string]*config.EntryMethodConfig{
		"auth.User/Login": {
			RecordRuleAllow: []config.EntryRecordRuleAllow{
				{Model: "auth.User", Ops: []string{"read", "create"}},
				{Model: "", Ops: []string{"ignored"}},
			},
			SkipFieldRule: true,
		},
	}
	svc := &ApplicationService{runtimeScope: runtimeScope}
	jsCtx := map[string]interface{}{"req": map[string]any{"depth": 0}}

	svc.injectMethodMetaAndEntryPolicy(jsCtx, "/auth.User/Login")
	rm := jsCtx["req"].(map[string]any)
	if rm["fullMethod"] != "/auth.User/Login" || rm["method"] != "auth.User/Login" {
		t.Fatalf("unexpected method metadata: %#v", rm)
	}
	if rm["companyMode"] != "skip" {
		t.Fatalf("expected global company filter disable to inject skip mode, got %#v", rm)
	}
	if rm["recordRuleMode"] != "allowlist" {
		t.Fatalf("expected record rule allowlist mode, got %#v", rm)
	}
	allow, ok := rm["recordRuleAllow"].([]string)
	if !ok || len(allow) != 2 || allow[0] != "auth.User:read" || allow[1] != "auth.User:create" {
		t.Fatalf("unexpected recordRuleAllow: %#v", rm["recordRuleAllow"])
	}
	if rm["fieldRuleMode"] != "skip" {
		t.Fatalf("expected fieldRuleMode skip from method policy, got %#v", rm)
	}

	jsCtx = map[string]interface{}{"req": map[string]any{"depth": 1}}
	svc.injectMethodMetaAndEntryPolicy(jsCtx, "/auth.User/Login")
	rm = jsCtx["req"].(map[string]any)
	if _, ok := rm["companyMode"]; ok {
		t.Fatalf("expected non-top-level request to skip policy injection, got %#v", rm)
	}

	runtimeScope.cfg.Auth.Enabled = false
	jsCtx = map[string]interface{}{"req": map[string]any{"depth": 0}}
	svc.injectMethodMetaAndEntryPolicy(jsCtx, "/auth.User/Login")
	rm = jsCtx["req"].(map[string]any)
	if rm["companyMode"] != "skip" || rm["recordRuleMode"] != "skip" || rm["fieldRuleMode"] != "skip" {
		t.Fatalf("expected auth disabled to force skip modes, got %#v", rm)
	}

	svc.injectMethodMetaAndEntryPolicy(map[string]interface{}{}, "/auth.User/Login")
}

func TestEnforceMethodAccessShortCircuitsAndErrors(t *testing.T) {
	runtimeScope := newHelperScope(t.TempDir())
	runtimeScope.cfg.Auth.Enabled = true
	runtimeScope.cfg.Auth.GrpcMethodAccess = true
	svc := &ApplicationService{runtimeScope: runtimeScope}

	if err := svc.enforceMethodAccess(context.Background(), runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 1}}, "/auth.User/Login"); err != nil {
		t.Fatalf("expected non-top-level request to bypass ACL, got %v", err)
	}

	runtimeScope.cfg.Auth.GrpcEntryPolicy = map[string]*config.EntryMethodConfig{
		"auth.User/Login": {SkipAuthentication: true},
	}
	if err := svc.enforceMethodAccess(context.Background(), runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login"); err != nil {
		t.Fatalf("expected skipAuthentication policy to bypass ACL, got %v", err)
	}

	runtimeScope.cfg.Auth.GrpcEntryPolicy = map[string]*config.EntryMethodConfig{
		"auth.User/Login": {SkipMethodAccess: true},
	}
	if err := svc.enforceMethodAccess(context.Background(), runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login"); err != nil {
		t.Fatalf("expected skipMethodAccess policy to bypass ACL, got %v", err)
	}

	runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
	for _, fullMethod := range []string{"auth.User/CheckMethodAccess"} {
		if err := svc.enforceMethodAccess(context.Background(), runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, fullMethod); err != nil {
			t.Fatalf("expected excluded method %q to bypass ACL, got %v", fullMethod, err)
		}
	}

	if err := svc.enforceMethodAccess(context.Background(), runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login"); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected missing identity to be unauthenticated, got %v", err)
	}

	ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "", tokenID: "t1"})
	if err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login"); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected blank user id to be unauthenticated, got %v", err)
	}

	ctx = auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})
	err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0, "traceId": "t", "spanId": "s"}, "ctx": map[string]any{"activeCompanyId": "c1"}}, "/auth.User/Login")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected missing service dialer to map to failed precondition, got %v", err)
	}

	err = svc.enforceMethodAccessStrict(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected strict ACL to use same missing dialer mapping, got %v", err)
	}

	runtimeScope.cfg.Auth.Enabled = false
	if err := svc.enforceMethodAccessStrict(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/auth.User/Login"); err != nil {
		t.Fatalf("expected disabled auth to bypass strict ACL, got %v", err)
	}
}

func TestEnforceMethodAccessExecutorPaths(t *testing.T) {
	t.Run("allowed ACL returns thread routing", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		var capturedReq *jsengine.JsRequest
		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			capturedReq = req
			return &jsengine.JsResponse{
				Id:      req.Id,
				Result:  true,
				Routing: &jsengine.JsExecutionRouting{ThreadID: uint32(7)},
			}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})
		jsCtx := map[string]interface{}{
			"req": map[string]any{"depth": 0},
			"ctx": map[string]any{"companyId": "legacy-company"},
		}

		routing, err := svc.guard().authorizeUnary(ctx, runtimeScope, jsCtx, "/sales.Order/Delete")
		if err != nil {
			t.Fatalf("authorizeUnary() error = %v", err)
		}
		if capturedReq == nil {
			t.Fatal("expected ACL request to be executed")
		}
		if capturedReq.Service != "auth.User.CheckMethodAccess" {
			t.Fatalf("unexpected ACL service: %q", capturedReq.Service)
		}
		if len(capturedReq.Args) != 2 || capturedReq.Args[0] != "legacy-company" || capturedReq.Args[1] != "/sales.Order/Delete" {
			t.Fatalf("unexpected ACL args: %#v", capturedReq.Args)
		}
		if routing == nil || routing.ThreadID == 0 {
			t.Fatalf("expected non-zero routing thread id, got %#v", routing)
		}
		if reqMeta, ok := jsCtx["req"].(map[string]any); ok {
			if legacy, exists := reqMeta["__threadId"]; exists {
				t.Fatalf("expected legacy thread key to be absent, got %#v", legacy)
			}
		}
	})

	t.Run("allowed ACL without routing still succeeds", func(t *testing.T) {
		// Bypass defaultjsexecutor (which always fills Routing) so checkMethodAccess's
		// nil-Routing return path is reachable.
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		svc := &ApplicationService{
			runtimeScope: runtimeScope,
			jsExecutor: &stubJsExecutor{execute: func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
				return &jsengine.JsResponse{
					Id: req.Id,
					Result: map[string]any{
						"allowed":    true,
						"reason":     "method_access_allow",
						"hitRuleIds": []string{"ma_no_routing"},
					},
				}, nil
			}},
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})
		routing, err := svc.guard().authorizeUnary(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Browse")
		if err != nil {
			t.Fatalf("authorizeUnary() error = %v", err)
		}
		if routing != nil {
			t.Fatalf("expected nil routing when ACL response omits Routing, got %#v", routing)
		}
	})

	t.Run("invalid ACL result type becomes unavailable", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: req.Id, Result: "bad"}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})

		err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Delete")
		if status.Code(err) != codes.Unavailable || !strings.Contains(status.Convert(err).Message(), "invalid CheckMethodAccess result type") {
			t.Fatalf("expected invalid ACL result type to map to unavailable, got %v", err)
		}
	})

	t.Run("ACL decision map missing allowed becomes unavailable", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: req.Id, Result: map[string]any{"reason": "method_access_deny"}}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})

		err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Delete")
		if status.Code(err) != codes.Unavailable || !strings.Contains(status.Convert(err).Message(), "invalid or missing 'allowed' field") {
			t.Fatalf("expected missing allowed field to map to unavailable, got %v", err)
		}
	})

	t.Run("denied ACL returns permission denied", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: req.Id, Result: false}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})

		err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Delete")
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected denied ACL to map to permission denied, got %v", err)
		}
	})

	t.Run("storage non-whitelist method returns permission denied", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil

		const deniedMethod = "/document.AttachmentContent/RunGarbageCollection"

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			if len(req.Args) != 2 || req.Args[1] != deniedMethod {
				t.Fatalf("unexpected ACL args for storage non-whitelist check: %#v", req.Args)
			}
			return &jsengine.JsResponse{Id: req.Id, Result: false}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})

		err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, deniedMethod)
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected storage non-whitelist ACL deny to map to permission denied, got %v", err)
		}
	})

	t.Run("strict ACL ignores entry policy skips", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = map[string]*config.EntryMethodConfig{
			"sales.Order/Delete": {
				SkipAuthentication: true,
				SkipMethodAccess:   true,
			},
		}

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: req.Id, Result: false}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u1", tokenID: "t1"})

		err := svc.enforceMethodAccessStrict(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Delete")
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected strict ACL to ignore entry policy skips and deny access, got %v", err)
		}
	})
}

func TestEnforceMethodAccessDecisionObservabilityLogging(t *testing.T) {
	t.Run("decision envelope normalizes hitRuleIds from string slice and csv", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
		runtimeScope.cfg.Auth.AuthzDecisionLog = "all"

		cases := []struct {
			name string
			ids  any
			want string
		}{
			{name: "stringSlice", ids: []string{" b ", "", "a", "a"}, want: "hitRuleIds=\"[a b]\""},
			{name: "csv", ids: "b, a, a, ", want: "hitRuleIds=\"[a b]\""},
			{name: "nilIds", ids: nil, want: ""},
			{name: "unsupported", ids: 42, want: ""},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var logBuf bytes.Buffer
				runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
				executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
					return &jsengine.JsResponse{
						Id: req.Id,
						Result: map[string]any{
							"allowed":    true,
							"reason":     "method_access_allow",
							"hitRuleIds": tc.ids,
						},
					}, nil
				})
				svc := &ApplicationService{
					runtimeScope:  runtimeScope,
					jsExecutor:    executor,
					hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
				}
				ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u-hit", tokenID: "t1"})
				err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Allow")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				logs := logBuf.String()
				if tc.want == "" {
					if strings.Contains(logs, "hitRuleIds=") {
						t.Fatalf("did not expect hitRuleIds in logs, got:\n%s", logs)
					}
					return
				}
				if !strings.Contains(logs, tc.want) {
					t.Fatalf("expected log to contain %q, got:\n%s", tc.want, logs)
				}
			})
		}
	})

	t.Run("decision envelope forwards reason and hitRuleIds into decision_summary", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
		runtimeScope.cfg.Auth.AuthzDecisionLog = "all"
		runtimeScope.cfg.Auth.AuthzDecisionAudit = true

		var logBuf bytes.Buffer
		runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{
				Id: req.Id,
				Result: map[string]any{
					"allowed":    false,
					"reason":     "method_access_deny",
					"hitRuleIds": []any{" ma_2 ", "", "ma_1", "ma_1"},
				},
			}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u-e5", tokenID: "t1"})
		jsCtx := map[string]interface{}{
			"req": map[string]any{"depth": 0},
			"ctx": map[string]any{"activeCompanyId": "company-a"},
		}

		err := svc.enforceMethodAccess(ctx, runtimeScope, jsCtx, "/sales.Order/Deny")
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected permission denied, got %v", err)
		}
		logs := logBuf.String()
		for _, want := range []string{
			"event=authz.decision_summary",
			"layer=method_access",
			"decision=deny",
			"basis=acl_denied",
			"reason=method_access_deny",
			"hitRuleIds=\"[ma_1 ma_2]\"",
			"audit=true",
		} {
			if !strings.Contains(logs, want) {
				t.Fatalf("expected log to contain %q, got:\n%s", want, logs)
			}
		}
	})

	t.Run("allow and deny emit camelCase decision_summary with audit", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
		runtimeScope.cfg.Auth.AuthzDecisionLog = "all"
		runtimeScope.cfg.Auth.AuthzDecisionAudit = true

		var logBuf bytes.Buffer
		runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			allowed := false
			if len(req.Args) == 2 {
				if method, ok := req.Args[1].(string); ok && method == "/sales.Order/Allow" {
					allowed = true
				}
			}
			return &jsengine.JsResponse{Id: req.Id, Result: allowed}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u-obs", tokenID: "t1"})
		jsCtx := map[string]interface{}{
			"req": map[string]any{"depth": 0},
			"ctx": map[string]any{
				"activeCompanyId":   "company-a",
				"enabledCompanyIds": []any{"company-a", " ", "company-b"},
			},
		}

		if _, err := svc.guard().authorizeUnary(ctx, runtimeScope, jsCtx, "/sales.Order/Allow"); err != nil {
			t.Fatalf("authorizeUnary(allow) error = %v", err)
		}
		err := svc.enforceMethodAccess(ctx, runtimeScope, jsCtx, "/sales.Order/Deny")
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected deny ACL, got %v", err)
		}

		logs := logBuf.String()
		for _, want := range []string{
			"authz.decision_summary",
			"method_access",
			"fullMethod",
			"userId",
			"activeCompanyId",
			"enabledCompanyIds",
			"acl_allowed",
			"acl_denied",
			"audit=true",
		} {
			if !strings.Contains(logs, want) {
				t.Fatalf("expected decision log to contain %q, got:\n%s", want, logs)
			}
		}
	})

	t.Run("deny mode with companyId fallback and string company list", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
		runtimeScope.cfg.Auth.AuthzDecisionLog = "deny"
		runtimeScope.cfg.Auth.AuthzDecisionAudit = false

		var logBuf bytes.Buffer
		runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return &jsengine.JsResponse{Id: req.Id, Result: false}, nil
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u-obs-2", tokenID: "t2"})
		jsCtx := map[string]interface{}{
			"req": map[string]any{"depth": 0},
			"ctx": map[string]any{
				"companyId":         "legacy-c1",
				"enabledCompanyIds": []string{"legacy-c1", "legacy-c2"},
			},
		}

		err := svc.enforceMethodAccess(ctx, runtimeScope, jsCtx, "/sales.Order/Delete")
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected deny ACL, got %v", err)
		}

		logs := logBuf.String()
		for _, want := range []string{
			"authz.decision_summary",
			"method_access",
			"acl_denied",
			"legacy-c1",
		} {
			if !strings.Contains(logs, want) {
				t.Fatalf("expected decision log to contain %q, got:\n%s", want, logs)
			}
		}
		if strings.Contains(logs, "audit=true") {
			t.Fatalf("did not expect audit log when AuthzDecisionAudit=false, got:\n%s", logs)
		}
	})

	t.Run("check failure emits acl_check_failed extra", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = true
		runtimeScope.cfg.Auth.GrpcMethodAccess = true
		runtimeScope.cfg.Auth.GrpcEntryPolicy = nil
		runtimeScope.cfg.Auth.AuthzDecisionLog = "deny"

		var logBuf bytes.Buffer
		runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return nil, errors.New("acl boom")
		})

		svc := &ApplicationService{
			runtimeScope:  runtimeScope,
			jsExecutor:    executor,
			hasGrpcMethod: func(fullMethod string) bool { return fullMethod == "/auth.User/CheckMethodAccess" },
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "u-obs-3", tokenID: "t3"})

		err := svc.enforceMethodAccess(ctx, runtimeScope, map[string]interface{}{"req": map[string]any{"depth": 0}}, "/sales.Order/Delete")
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("expected check failure unavailable, got %v", err)
		}
		logs := logBuf.String()
		if !strings.Contains(logs, "acl_check_failed") || !strings.Contains(logs, "authz.decision_summary") {
			t.Fatalf("expected acl_check_failed decision summary, got:\n%s", logs)
		}
	})

	t.Run("logAuthzDecisionSummary no-ops on nil scope logger or payload", func(t *testing.T) {
		payload := map[string]any{"event": "authz.decision_summary", "layer": "method_access"}
		// Each OR branch of the early-return guard must be exercised.
		logAuthzDecisionSummary(nil, payload, false)

		nilLoggerScope := newHelperScope(t.TempDir())
		nilLoggerScope.logger = nil
		logAuthzDecisionSummary(nilLoggerScope, payload, true)

		runtimeScope := newHelperScope(t.TempDir())
		var logBuf bytes.Buffer
		runtimeScope.logger = slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
		logAuthzDecisionSummary(runtimeScope, nil, false)
		if logBuf.Len() != 0 {
			t.Fatalf("expected nil payload to skip logging, got %q", logBuf.String())
		}
	})
}

func TestAuthorizeInternalCallerFromContext(t *testing.T) {
	runtimeScope := newHelperScope(t.TempDir())
	runtimeScope.cfg.Auth.InternalKey = "secret"
	runtimeScope.cfg.Server.Environment = "development"

	if authorizeInternalCallerFromContext(context.Background(), nil) {
		t.Fatal("expected nil env to be rejected")
	}
	if authorizeInternalCallerFromContext(context.Background(), runtimeScope) {
		t.Fatal("expected missing metadata to be rejected")
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
	if !authorizeInternalCallerFromContext(ctx, runtimeScope) {
		t.Fatal("expected matching internal key to be accepted")
	}

	runtimeScope.cfg.Server.Environment = "production"
	if authorizeInternalCallerFromContext(ctx, runtimeScope) {
		t.Fatal("expected production environment to reject internal key bypass")
	}
}

func TestParseProtoFilesEdgeCases(t *testing.T) {
	svc := &ApplicationService{runtimeScope: newHelperScope(t.TempDir())}

	descs, err := svc.parseProtoFiles([]string{"ignored.proto"})
	if err != nil {
		t.Fatalf("parseProtoFiles(empty import paths) error = %v", err)
	}
	if descs != nil {
		t.Fatalf("expected nil descs when protoImportPaths is empty, got %#v", descs)
	}

	rootDir := t.TempDir()
	svc.protoImportPaths = []string{rootDir}
	descs, err = svc.parseProtoFiles([]string{filepath.Join(t.TempDir(), "outside.proto")})
	if err != nil {
		t.Fatalf("parseProtoFiles(outside root) error = %v", err)
	}
	if descs != nil {
		t.Fatalf("expected nil descs for files outside import root, got %#v", descs)
	}

	brokenProto := filepath.Join(rootDir, "broken.proto")
	if err := os.WriteFile(brokenProto, []byte("syntax = \"proto3\"; package broken; service S { rpc Ping(Bad) returns (); }"), 0o644); err != nil {
		t.Fatalf("write broken proto: %v", err)
	}
	if _, err := svc.parseProtoFiles([]string{brokenProto}); err == nil || !strings.Contains(err.Error(), "error parsing proto files") {
		t.Fatalf("expected wrapped parseProtoFiles error, got %v", err)
	}
}

func TestMethodHandlerErrorAndInterceptorPaths(t *testing.T) {
	methodDesc, _, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors() error = %v", err)
	}
	svc := &ApplicationService{runtimeScope: newHelperScope(t.TempDir())}
	handler := svc.methodHandler(methodDesc.ParentFile().Package().Name(), methodDesc.Parent().Name(), methodDesc)

	_, err = handler(nil, context.Background(), func(any) error {
		return errors.New("decode failed")
	}, nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument on decode error, got %v", err)
	}

	intercepted := false
	resp, err := handler(nil, context.Background(), func(msg any) error {
		_, ok := msg.(*dynamicpb.Message)
		if !ok {
			t.Fatalf("expected decoder target to be dynamicpb.Message, got %T", msg)
		}
		return nil
	}, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		intercepted = true
		if info.FullMethod != "/task.TaskWorker/ExecuteJob" {
			t.Fatalf("unexpected full method: %s", info.FullMethod)
		}
		return "intercepted", nil
	})
	if err != nil {
		t.Fatalf("methodHandler(interceptor) error = %v", err)
	}
	if !intercepted || resp != "intercepted" {
		t.Fatalf("expected interceptor result, got resp=%#v intercepted=%v", resp, intercepted)
	}
}

func TestExecuteUnaryPropagatesExecutorErrors(t *testing.T) {
	methodDesc, reqDesc, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors() error = %v", err)
	}
	reqMsg := dynamicpb.NewMessage(reqDesc)
	svc := &ApplicationService{runtimeScope: newHelperScope(t.TempDir()), jsExecutor: jsexecutortest.NewUninitializedExecutor()}

	_, err = svc.executeUnary(
		context.Background(),
		svc.runtimeScope,
		map[string]interface{}{"req": map[string]any{"depth": 0}},
		nil,
		methodDesc.ParentFile().Package().Name(),
		methodDesc.Parent().Name(),
		methodDesc.Name(),
		methodDesc.Output(),
		reqMsg,
	)
	if err == nil || !strings.Contains(err.Error(), "executor-not-initialized: thread pool is not initialized") {
		t.Fatalf("expected executeUnary to propagate executor setup error, got %v", err)
	}
}

func TestUnaryHandlerErrorPaths(t *testing.T) {
	methodDesc, _, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors() error = %v", err)
	}

	t.Run("run error is mapped through handleError", func(t *testing.T) {
		runtimeScope := &controlledScope{
			helperScope: newHelperScope(t.TempDir()),
			runFn: func(fn func(runtimeScope scope.Scope) error) error {
				return errors.New("boom")
			},
		}
		svc := &ApplicationService{runtimeScope: runtimeScope}
		handler := svc.unaryHandler(methodDesc.ParentFile().Package().Name(), methodDesc.Parent().Name(), methodDesc.Name(), methodDesc.Output())

		_, err := handler(context.Background(), dynamicpb.NewMessage(methodDesc.Input()))
		if status.Code(err) != codes.Internal || status.Convert(err).Message() != "boom" {
			t.Fatalf("expected unaryHandler to map run error to internal status, got %v", err)
		}
	})

	t.Run("nil output message falls back to internal status", func(t *testing.T) {
		runtimeScope := &controlledScope{
			helperScope: newHelperScope(t.TempDir()),
			runFn: func(fn func(runtimeScope scope.Scope) error) error {
				return nil
			},
		}
		svc := &ApplicationService{runtimeScope: runtimeScope}
		handler := svc.unaryHandler(methodDesc.ParentFile().Package().Name(), methodDesc.Parent().Name(), methodDesc.Name(), methodDesc.Output())

		_, err := handler(context.Background(), dynamicpb.NewMessage(methodDesc.Input()))
		if status.Code(err) != codes.Internal || status.Convert(err).Message() != "outMsg is nil" {
			t.Fatalf("expected outMsg nil fallback, got %v", err)
		}
	})

	t.Run("executeUnary errors are mapped through handleError", func(t *testing.T) {
		runtimeScope := newHelperScope(t.TempDir())
		runtimeScope.cfg.Auth.Enabled = false
		svc := &ApplicationService{runtimeScope: runtimeScope, jsExecutor: jsexecutortest.NewUninitializedExecutor()}
		handler := svc.unaryHandler(methodDesc.ParentFile().Package().Name(), methodDesc.Parent().Name(), methodDesc.Name(), methodDesc.Output())

		_, err := handler(context.Background(), dynamicpb.NewMessage(methodDesc.Input()))
		if status.Code(err) != codes.Internal || !strings.Contains(status.Convert(err).Message(), "executor-not-initialized: thread pool is not initialized") {
			t.Fatalf("expected executeUnary failure to map to internal status, got %v", err)
		}
	})
}

func TestServiceDescsRegistersLoaderAndSkipsTaskWorkerForWeb(t *testing.T) {
	t.Run("non-web service registers proto in global loader", func(t *testing.T) {
		root := t.TempDir()
		protoDir := filepath.Join(root, "loaderapp")
		if err := os.MkdirAll(protoDir, 0o755); err != nil {
			t.Fatalf("mkdir proto dir: %v", err)
		}
		protoText := `syntax = "proto3";
package loaderapp;

service LoaderService {
  rpc Ping(PingRequest) returns (PingReply);
}

message PingRequest { string msg = 1; }
message PingReply { string msg = 1; }
`
		if err := os.WriteFile(filepath.Join(protoDir, "loader.proto"), []byte(protoText), 0o644); err != nil {
			t.Fatalf("write loader proto: %v", err)
		}

		runtimeScope := newHelperScope(root)
		svc := &ApplicationService{runtimeScope: runtimeScope, name: "loaderapp", appDistPath: root, protoRootDir: root, protoImportPaths: []string{root}}
		descs, err := svc.ServiceDescs()
		if err != nil {
			t.Fatalf("ServiceDescs(loaderapp) error = %v", err)
		}
		if len(descs) != 3 {
			t.Fatalf("expected loaderapp desc plus TaskWorker plus I18n, got %#v", descs)
		}
		var hasLoaderI18n bool
		for _, desc := range descs {
			if desc.ServiceName == "loaderapp.I18n" {
				hasLoaderI18n = true
			}
		}
		if !hasLoaderI18n {
			t.Fatalf("missing loaderapp.I18n descriptor in %#v", descs)
		}
		md, err := loader.Global().GetMethodDescriptor("loaderapp.LoaderService.Ping")
		if err != nil {
			t.Fatalf("global loader missing registered method: %v", err)
		}
		if string(md.FullName()) != "loaderapp.LoaderService.Ping" {
			t.Fatalf("unexpected method descriptor: %s", md.FullName())
		}
	})

	t.Run("web service does not inject TaskWorker", func(t *testing.T) {
		root := t.TempDir()
		protoDir := filepath.Join(root, "web")
		if err := os.MkdirAll(protoDir, 0o755); err != nil {
			t.Fatalf("mkdir web proto dir: %v", err)
		}
		protoText := `syntax = "proto3";
package web;

service WebService {
  rpc Ping(PingRequest) returns (PingReply);
}

message PingRequest { string msg = 1; }
message PingReply { string msg = 1; }
`
		if err := os.WriteFile(filepath.Join(protoDir, "web.proto"), []byte(protoText), 0o644); err != nil {
			t.Fatalf("write web proto: %v", err)
		}

		runtimeScope := newHelperScope(root)
		svc := &ApplicationService{runtimeScope: runtimeScope, name: "web", appDistPath: root, protoRootDir: root, protoImportPaths: []string{root}}
		descs, err := svc.ServiceDescs()
		if err != nil {
			t.Fatalf("ServiceDescs(web) error = %v", err)
		}
		var hasWebService, hasTaskWorker, hasI18n bool
		for _, desc := range descs {
			switch desc.ServiceName {
			case "web.WebService":
				hasWebService = true
			case "web.TaskWorker":
				hasTaskWorker = true
			case "web.I18n":
				hasI18n = true
			}
		}
		if !hasWebService || hasTaskWorker || !hasI18n || len(descs) != 2 {
			t.Fatalf("expected web.WebService+web.I18n without TaskWorker, got %#v", descs)
		}
	})
}

func TestServiceDescsEdgeCases(t *testing.T) {
	t.Run("missing proto directory still injects TaskWorker and I18n", func(t *testing.T) {
		root := t.TempDir()
		runtimeScope := newHelperScope(root)
		svc := &ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: root, protoRootDir: filepath.Join(root, "missing")}

		descs, err := svc.ServiceDescs()
		if err != nil {
			t.Fatalf("ServiceDescs(missing) error = %v", err)
		}
		var hasTaskWorker, hasI18n bool
		for _, desc := range descs {
			switch desc.ServiceName {
			case "auth.TaskWorker":
				hasTaskWorker = true
			case "auth.I18n":
				hasI18n = true
			}
		}
		if !hasTaskWorker || !hasI18n || len(descs) != 2 {
			t.Fatalf("expected TaskWorker+I18n for missing proto dir, got %#v", descs)
		}
	})

	t.Run("non-web empty proto dir still injects TaskWorker and I18n", func(t *testing.T) {
		root := t.TempDir()
		protoDir := filepath.Join(root, "assets")
		if err := os.MkdirAll(protoDir, 0o755); err != nil {
			t.Fatalf("mkdir proto dir: %v", err)
		}
		runtimeScope := newHelperScope(root)
		svc := &ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: root, protoRootDir: protoDir}

		descs, err := svc.ServiceDescs()
		if err != nil {
			t.Fatalf("ServiceDescs(empty) error = %v", err)
		}
		var hasTaskWorker, hasI18n bool
		for _, desc := range descs {
			switch desc.ServiceName {
			case "auth.TaskWorker":
				hasTaskWorker = true
			case "auth.I18n":
				hasI18n = true
			}
		}
		if !hasTaskWorker || !hasI18n || len(descs) != 2 {
			t.Fatalf("expected TaskWorker+I18n for empty proto dir, got %#v", descs)
		}
	})

	t.Run("broken proto returns parse error", func(t *testing.T) {
		root := t.TempDir()
		protoDir := filepath.Join(root, "assets")
		if err := os.MkdirAll(protoDir, 0o755); err != nil {
			t.Fatalf("mkdir proto dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(protoDir, "broken.proto"), []byte("syntax = \"proto3\"; package broken; service S { rpc Ping(Bad) returns (); }"), 0o644); err != nil {
			t.Fatalf("write broken proto: %v", err)
		}
		runtimeScope := newHelperScope(root)
		svc := &ApplicationService{runtimeScope: runtimeScope, name: "auth", appDistPath: root, protoRootDir: protoDir, protoImportPaths: []string{protoDir}}

		if _, err := svc.ServiceDescs(); err == nil || !strings.Contains(err.Error(), "error parsing proto files") {
			t.Fatalf("expected ServiceDescs to surface parse error, got %v", err)
		}
	})
}
