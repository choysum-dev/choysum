// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/terms"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Minimal google.protobuf.Struct so loader.Global can compile test fixtures
// without relying on WithStandardImports(nil) (which panics on WKT lookup).
const googleProtobufStructProto = `syntax = "proto3";
package google.protobuf;

message Struct {
  map<string, Value> fields = 1;
}

message Value {
  oneof kind {
    NullValue null_value = 1;
    double number_value = 2;
    string string_value = 3;
    bool bool_value = 4;
    Struct struct_value = 5;
    ListValue list_value = 6;
  }
}

message ListValue {
  repeated Value values = 1;
}

enum NullValue {
  NULL_VALUE = 0;
}
`

const authTranslationTermProto = `syntax = "proto3";
package auth;
import "google/protobuf/struct.proto";

service TranslationTerm {
  rpc GetTranslations(GetTranslationsReq) returns (GetTranslationsResp);
  rpc Search(SearchReq) returns (SearchResp);
  rpc Count(CountReq) returns (CountResp);
}

message GetTranslationsReq {
  google.protobuf.Value req = 1;
}
message GetTranslationsResp {
  google.protobuf.Value result = 1;
}

message SearchReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
message SearchResp {
  repeated google.protobuf.Struct result = 1;
}

message CountReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
message CountResp {
  int64 result = 1;
}
`

var registerAuthTranslationTermProtoOnce sync.Once

func registerAuthTranslationTermProtoForTests() {
	registerAuthTranslationTermProtoOnce.Do(func() {
		loader.Global().RegisterProto("google/protobuf/struct.proto", googleProtobufStructProto)
		loader.Global().RegisterProto("auth/translation_term_gateway_test.proto", authTranslationTermProto)
	})
}

type translationTermRPCBehavior struct {
	countTotal  int64
	countErr    error
	searchItems []map[string]any
	searchErr   error
	getHash     string
	getTerms    map[string]any
	getErr      error
	// getResultRaw when non-nil is written as the GetTranslations "result" Value
	// (e.g. a string) instead of the normal catalog object.
	getResultRaw any
}

func newTranslationTermDialer(t *testing.T, behavior *translationTermRPCBehavior) grpcclient.ServiceDialer {
	t.Helper()
	registerAuthTranslationTermProtoForTests()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		fullMethod, ok := grpc.MethodFromServerStream(stream)
		if !ok {
			return errors.New("missing method")
		}
		// /auth.TranslationTerm/Count → auth.TranslationTerm.Count
		trimmed := strings.TrimPrefix(fullMethod, "/")
		descriptorName := strings.ReplaceAll(trimmed, "/", ".")
		md, err := loader.Global().GetMethodDescriptor(descriptorName)
		if err != nil {
			return err
		}
		req := dynamicpb.NewMessage(md.Input())
		if err := stream.RecvMsg(req); err != nil {
			return err
		}
		resp := dynamicpb.NewMessage(md.Output())
		switch {
		case strings.HasSuffix(fullMethod, "/"+translationTermCount):
			if behavior.countErr != nil {
				return behavior.countErr
			}
			if err := converter.MapToMessage(map[string]any{"result": behavior.countTotal}, resp); err != nil {
				return err
			}
		case strings.HasSuffix(fullMethod, "/"+translationTermSearch):
			if behavior.searchErr != nil {
				return behavior.searchErr
			}
			items := make([]any, 0, len(behavior.searchItems))
			for _, item := range behavior.searchItems {
				items = append(items, item)
			}
			if err := converter.MapToMessage(map[string]any{"result": items}, resp); err != nil {
				return err
			}
		case strings.HasSuffix(fullMethod, "/"+translationTermGetTranslations):
			if behavior.getErr != nil {
				return behavior.getErr
			}
			var result any
			if behavior.getResultRaw != nil {
				result = behavior.getResultRaw
			} else {
				payload := map[string]any{"hash": behavior.getHash}
				if behavior.getTerms != nil {
					payload["terms_by_module"] = behavior.getTerms
				}
				result = payload
			}
			if err := converter.MapToMessage(map[string]any{"result": result}, resp); err != nil {
				return err
			}
		default:
			return errors.New("unexpected method: " + fullMethod)
		}
		return stream.SendMsg(resp)
	}))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})
	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.TranslationTerm" {
			return nil, errors.New("unexpected service: " + serviceName)
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(
			ctx,
			"bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}

func TestFetchAppSearchTermsSuccessAndDefaults(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 1,
		searchItems: []map[string]any{{
			"Application": "auth",
			"Module":      "auth",
			"Scope":       "a@b",
			"Src":         "Hi",
			"Value":       "你好",
			"Kind":        "literal",
			"Comments":    "fuzzy",
		}},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))

	got, err := fetchAppSearchTerms(ctx, nil, "tok", "auth", "zh_CN", []string{"auth"}, "", 0, -1)
	if err != nil {
		t.Fatalf("fetchAppSearchTerms: %v", err)
	}
	if got.Total != 1 || got.Limit != 50 || got.Offset != 0 || len(got.Items) != 1 {
		t.Fatalf("got = %#v", got)
	}
	if got.Items[0].Status != "translated" {
		t.Fatalf("status = %q, want translated (Comments are refs, not fuzzy flags)", got.Items[0].Status)
	}
}

func TestCountAndSearchAppTermsPageSuccess(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 2,
		searchItems: []map[string]any{{
			"Module": "auth", "Scope": "a", "Src": "One", "Value": "一",
		}},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))

	total, err := countAppTerms(ctx, "tok", "auth", "zh_CN", []string{"auth", "web"}, "hi")
	if err != nil || total != 2 {
		t.Fatalf("countAppTerms = %d err=%v", total, err)
	}

	page, err := searchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", total, 0, -3)
	if err != nil {
		t.Fatalf("searchAppTermsPage: %v", err)
	}
	if page.Limit != 50 || page.Offset != 0 || page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("page = %#v", page)
	}
}

func TestCountAndSearchAppTermsRequireApplication(t *testing.T) {
	if _, err := countAppTerms(context.Background(), "tok", "  ", "zh_CN", nil, ""); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("countAppTerms empty app: %v", err)
	}
	if _, err := searchAppTermsPage(context.Background(), "tok", "", "zh_CN", nil, "", 0, 10, 0); err == nil || !strings.Contains(err.Error(), "application is required") {
		t.Fatalf("searchAppTermsPage empty app: %v", err)
	}
}

func TestCountAndSearchAppTermsDialFailure(t *testing.T) {
	registerAuthTranslationTermProtoForTests()
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	if _, err := countAppTerms(ctx, "tok", "auth", "zh_CN", nil, ""); err == nil {
		t.Fatal("expected count dial error")
	}
	if _, err := searchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", 1, 10, 0); err == nil {
		t.Fatal("expected search dial error")
	}
}

func TestInvokeTranslationTermCountDescriptorMissing(t *testing.T) {
	_, err := invokeTranslationTermCount(context.Background(), nil, "missingapp.TranslationTerm", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "load Count descriptor") {
		t.Fatalf("err = %v", err)
	}
}

func TestSearchTranslationTermPageDescriptorMissing(t *testing.T) {
	_, err := searchTranslationTermPage(context.Background(), nil, "missingapp.TranslationTerm", "missingapp", "zh_CN", map[string]any{}, 0, 10, 0)
	if err == nil || !strings.Contains(err.Error(), "load Search descriptor") {
		t.Fatalf("err = %v", err)
	}
}

func TestInvokeAndSearchBuildRequestErrors(t *testing.T) {
	registerAuthTranslationTermProtoForTests()
	behavior := &translationTermRPCBehavior{countTotal: 0}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))

	conn, err := grpcclient.Dial(ctx, "auth.TranslationTerm")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	badCondition := map[string]any{"bad": make(chan int)}
	if _, err := invokeTranslationTermCount(ctx, conn, "auth.TranslationTerm", badCondition); err == nil || !strings.Contains(err.Error(), "build Count request") {
		t.Fatalf("count build err = %v", err)
	}
	if _, err := searchTranslationTermPage(ctx, conn, "auth.TranslationTerm", "auth", "zh_CN", badCondition, 0, 10, 0); err == nil || !strings.Contains(err.Error(), "build Search request") {
		t.Fatalf("search build err = %v", err)
	}
}

func TestInvokeAndSearchRPCErrors(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countErr:  errors.New("count failed"),
		searchErr: errors.New("search failed"),
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	if _, err := countAppTerms(ctx, "tok", "auth", "zh_CN", nil, ""); err == nil {
		t.Fatal("expected count invoke error")
	}
	if _, err := searchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", 1, 10, 0); err == nil {
		t.Fatal("expected search invoke error")
	}
}

func TestFetchAppSearchTermsCountError(t *testing.T) {
	behavior := &translationTermRPCBehavior{countErr: errors.New("count boom")}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	if _, err := fetchAppSearchTerms(ctx, nil, "tok", "auth", "zh_CN", nil, "", 10, 0); err == nil {
		t.Fatal("expected count error from fetchAppSearchTerms")
	}
}

func TestFetchAppTranslationsSuccess(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		getHash: "abc123",
		getTerms: map[string]any{
			"auth": map[string]any{
				"a@t": map[string]any{"Hello": "你好"},
			},
		},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	rs := &rpcTestScope{ctx: ctx}
	got, err := fetchAppTranslations(ctx, rs, "auth", "zh_CN", []string{"auth"})
	if err != nil {
		t.Fatalf("fetchAppTranslations: %v", err)
	}
	if got.Hash != "abc123" || got.Terms["auth"]["a@t"]["Hello"] != "你好" {
		t.Fatalf("got = %#v", got)
	}
}

func TestFetchAppTranslationsMalformedResult(t *testing.T) {
	behavior := &translationTermRPCBehavior{getResultRaw: "not-an-object"}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	_, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"})
	if err == nil || !strings.Contains(err.Error(), "result must be an object") {
		t.Fatalf("err = %v, want result must be an object", err)
	}
}

func TestFetchAppTranslationsDialFailureWithDescriptor(t *testing.T) {
	registerAuthTranslationTermProtoForTests()
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	if _, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected dial error after descriptor load")
	}
}

func TestFetchAppTranslationsInvokeError(t *testing.T) {
	behavior := &translationTermRPCBehavior{getErr: errors.New("get boom")}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	if _, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected invoke error")
	}
}

func TestBuildTermSearchConditionBranches(t *testing.T) {
	none := buildTermSearchCondition("zh_CN", []string{"", "  "}, "")
	and, ok := none["And"].([]any)
	if !ok || len(and) != 1 {
		t.Fatalf("no modules: %#v", none)
	}

	one := buildTermSearchCondition("zh_CN", []string{"auth"}, "")
	and, ok = one["And"].([]any)
	if !ok || len(and) != 2 {
		t.Fatalf("one module: %#v", one)
	}

	many := buildTermSearchCondition("zh_CN", []string{"auth", "web"}, "q")
	and, ok = many["And"].([]any)
	if !ok || len(and) != 3 {
		t.Fatalf("many modules + q: %#v", many)
	}
}

func TestTermStatusAndToInt64Edges(t *testing.T) {
	if termStatus("x") != "translated" {
		t.Fatal("expected translated")
	}
	if termStatus("") != "missing" || termStatus("  ") != "missing" {
		t.Fatal("expected missing for blank value")
	}
	if toInt64(int32(3)) != 3 || toInt64(float32(4)) != 4 {
		t.Fatal("expected int32/float32 coercion")
	}
}

func TestCollectAllTermsUsesCountAndSearchAppPaths(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 1,
		searchItems: []map[string]any{{
			"Module": "auth", "Scope": "a@b", "Src": "Hi", "Value": "你好",
		}},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	h := &handler{} // search hook nil → countAppTerms + searchAppTermsPage
	items, truncated, err := h.collectAllTerms(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || truncated || len(items) != 1 {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllTermsCountPathError(t *testing.T) {
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	registerAuthTranslationTermProtoForTests()
	h := &handler{}
	if _, _, err := h.collectAllTerms(ctx, "tok", "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected count path dial error")
	}
}

func TestCollectAllTermsSearchPageError(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 1,
		searchErr:  errors.New("search boom"),
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	h := &handler{}
	if _, _, err := h.collectAllTerms(ctx, "tok", "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected searchAppTermsPage error")
	}
}

func TestCollectAllTermsTruncatesWhenTotalExceedsMax(t *testing.T) {
	oldMax := terms.ExportMaxItems
	oldPage := terms.ExportPageSize
	terms.ExportMaxItems = 2
	terms.ExportPageSize = 2
	t.Cleanup(func() {
		terms.ExportMaxItems = oldMax
		terms.ExportPageSize = oldPage
	})

	behavior := &translationTermRPCBehavior{
		countTotal: 5,
		searchItems: []map[string]any{
			{"Module": "auth", "Scope": "a@1", "Src": "One", "Value": "1"},
			{"Module": "auth", "Scope": "a@2", "Src": "Two", "Value": "2"},
		},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	h := &handler{}
	items, truncated, err := h.collectAllTerms(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || !truncated || len(items) != 2 {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func registerAuthTranslationTermDecodeFailProto(t *testing.T) {
	t.Helper()
	loader.Global().RegisterProto("google/protobuf/struct.proto", googleProtobufStructProto)
	loader.Global().RegisterProto("auth/translation_term_gateway_test.proto", `syntax = "proto3";
package auth;
import "google/protobuf/struct.proto";

service TranslationTerm {
  rpc GetTranslations(GetTranslationsReq) returns (google.protobuf.ListValue);
  rpc Search(SearchReq) returns (google.protobuf.ListValue);
  rpc Count(CountReq) returns (google.protobuf.ListValue);
}

message GetTranslationsReq {
  google.protobuf.Value req = 1;
}
message SearchReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
message CountReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
`)
	t.Cleanup(func() {
		loader.Global().RegisterProto("google/protobuf/struct.proto", googleProtobufStructProto)
		loader.Global().RegisterProto("auth/translation_term_gateway_test.proto", authTranslationTermProto)
	})
}

func newListValueResponseDialer(t *testing.T) grpcclient.ServiceDialer {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		fullMethod, ok := grpc.MethodFromServerStream(stream)
		if !ok {
			return errors.New("missing method")
		}
		trimmed := strings.TrimPrefix(fullMethod, "/")
		descriptorName := strings.ReplaceAll(trimmed, "/", ".")
		md, err := loader.Global().GetMethodDescriptor(descriptorName)
		if err != nil {
			return err
		}
		req := dynamicpb.NewMessage(md.Input())
		if err := stream.RecvMsg(req); err != nil {
			return err
		}
		// Output is google.protobuf.ListValue → MessageToMap fails ("not a map").
		resp := dynamicpb.NewMessage(md.Output())
		return stream.SendMsg(resp)
	}))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})
	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(
			ctx,
			"bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}

func TestDecodeResponseErrors(t *testing.T) {
	registerAuthTranslationTermDecodeFailProto(t)
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newListValueResponseDialer(t))

	if _, err := countAppTerms(ctx, "tok", "auth", "zh_CN", nil, ""); err == nil || !strings.Contains(err.Error(), "decode Count response") {
		t.Fatalf("count decode err = %v", err)
	}
	if _, err := searchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", 1, 10, 0); err == nil || !strings.Contains(err.Error(), "decode Search response") {
		t.Fatalf("search decode err = %v", err)
	}
	if _, err := fetchAppTranslations(ctx, nil, "auth", "zh_CN", []string{"auth"}); err == nil || !strings.Contains(err.Error(), "decode GetTranslations response") {
		t.Fatalf("get decode err = %v", err)
	}
}

func TestFetchAppTranslationsBuildRequestError(t *testing.T) {
	loader.Global().RegisterProto("google/protobuf/struct.proto", googleProtobufStructProto)
	loader.Global().RegisterProto("auth/translation_term_gateway_test.proto", `syntax = "proto3";
package auth;
import "google/protobuf/struct.proto";

service TranslationTerm {
  rpc GetTranslations(BadGetReq) returns (GetTranslationsResp);
  rpc Search(SearchReq) returns (SearchResp);
  rpc Count(CountReq) returns (CountResp);
}
message BadGetReq {
  // ListValue cannot accept the map payload MapToMessage sends for "req".
  google.protobuf.ListValue req = 1;
}
message GetTranslationsResp {
  google.protobuf.Value result = 1;
}
message SearchReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
message SearchResp {
  repeated google.protobuf.Struct result = 1;
}
message CountReq {
  google.protobuf.Struct condition = 1;
  google.protobuf.Struct options = 2;
}
message CountResp {
  int64 result = 1;
}
`)
	t.Cleanup(func() {
		loader.Global().RegisterProto("google/protobuf/struct.proto", googleProtobufStructProto)
		loader.Global().RegisterProto("auth/translation_term_gateway_test.proto", authTranslationTermProto)
	})
	_, err := fetchAppTranslations(context.Background(), nil, "auth", "zh_CN", []string{"auth"})
	if err == nil || !strings.Contains(err.Error(), "build GetTranslations request") {
		t.Fatalf("err = %v", err)
	}
}
