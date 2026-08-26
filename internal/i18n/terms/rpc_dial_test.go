// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/dynamicpb"
)

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
  rpc Search(SearchReq) returns (SearchResp);
  rpc Count(CountReq) returns (CountResp);
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
		loader.Global().RegisterProto("auth/translation_term_terms_test.proto", authTranslationTermProto)
	})
}

type translationTermRPCBehavior struct {
	countTotal  int64
	countErr    error
	searchItems []map[string]any
	searchErr   error
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

func TestFetchAppSearchTermsDialPath(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 1,
		searchItems: []map[string]any{{
			"Module": "auth", "Scope": "a@b", "Src": "Hi", "Value": "你好", "Kind": "menu",
		}},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	got, err := FetchAppSearchTerms(ctx, nil, "tok", "auth", "zh_CN", []string{"auth"}, "", 0, -1)
	if err != nil {
		t.Fatalf("FetchAppSearchTerms: %v", err)
	}
	if got.Limit != 50 || len(got.Items) != 1 || got.Items[0].Kind != "menu" {
		t.Fatalf("got = %#v", got)
	}
}

func TestFetchAppSearchTermsRequiresApplication(t *testing.T) {
	if _, err := FetchAppSearchTerms(context.Background(), nil, " ", "auth", "zh_CN", nil, "", 10, 0); err == nil {
		t.Fatal("expected application error")
	}
}

func TestCountAndSearchAppTermsDialPath(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countTotal: 2,
		searchItems: []map[string]any{{
			"Module": "auth", "Scope": "a", "Src": "One", "Value": "一",
		}},
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	total, err := CountAppTerms(ctx, "tok", "auth", "zh_CN", []string{"auth"}, "")
	if err != nil || total != 2 {
		t.Fatalf("CountAppTerms = %d err=%v", total, err)
	}
	page, err := SearchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", total, 0, -3)
	if err != nil || page.Limit != 50 || len(page.Items) != 1 {
		t.Fatalf("page = %#v err=%v", page, err)
	}
}

func TestCountAndSearchAppTermsDialFailure(t *testing.T) {
	registerAuthTranslationTermProtoForTests()
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), func(context.Context, string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial boom")
	})
	if _, err := CountAppTerms(ctx, "tok", "auth", "zh_CN", nil, ""); err == nil {
		t.Fatal("expected count dial error")
	}
	if _, err := SearchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", 1, 10, 0); err == nil {
		t.Fatal("expected search dial error")
	}
}

func TestInvokeTranslationTermCountDescriptorMissing(t *testing.T) {
	_, err := InvokeTranslationTermCount(context.Background(), nil, "missingapp.TranslationTerm", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "load Count descriptor") {
		t.Fatalf("err = %v", err)
	}
}

func TestSearchTranslationTermPageDescriptorMissing(t *testing.T) {
	_, err := SearchTranslationTermPage(context.Background(), nil, "missingapp.TranslationTerm", "missingapp", "zh_CN", map[string]any{}, 0, 10, 0)
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
	if _, err := InvokeTranslationTermCount(ctx, conn, "auth.TranslationTerm", badCondition); err == nil || !strings.Contains(err.Error(), "build Count request") {
		t.Fatalf("count build err = %v", err)
	}
	if _, err := SearchTranslationTermPage(ctx, conn, "auth.TranslationTerm", "auth", "zh_CN", badCondition, 0, 10, 0); err == nil || !strings.Contains(err.Error(), "build Search request") {
		t.Fatalf("search build err = %v", err)
	}
}

func TestInvokeAndSearchRPCErrors(t *testing.T) {
	behavior := &translationTermRPCBehavior{
		countErr:  errors.New("count failed"),
		searchErr: errors.New("search failed"),
	}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	if _, err := CountAppTerms(ctx, "tok", "auth", "zh_CN", nil, ""); err == nil {
		t.Fatal("expected count invoke error")
	}
	if _, err := SearchAppTermsPage(ctx, "tok", "auth", "zh_CN", nil, "", 1, 10, 0); err == nil {
		t.Fatal("expected search invoke error")
	}
}

func TestFetchAppSearchTermsCountError(t *testing.T) {
	behavior := &translationTermRPCBehavior{countErr: errors.New("count boom")}
	ctx := grpcclient.ContextWithServiceDialer(context.Background(), newTranslationTermDialer(t, behavior))
	if _, err := FetchAppSearchTerms(ctx, nil, "tok", "auth", "zh_CN", nil, "", 10, 0); err == nil {
		t.Fatal("expected count error")
	}
}
