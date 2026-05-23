// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package loader

import (
	"strings"
	"testing"
)

const greeterProtoV1 = `syntax = "proto3";
package demo;

service Greeter {
  rpc SayHello (SayHelloReq) returns (SayHelloResp);
}

message SayHelloReq {
  string name = 1;
}

message SayHelloResp {
  string message = 1;
}
`

const greeterProtoV2 = `syntax = "proto3";
package demo;

service Greeter {
  rpc WaveHello (WaveHelloReq) returns (WaveHelloResp);
}

message WaveHelloReq {
  string name = 1;
}

message WaveHelloResp {
  string message = 1;
}
`

func TestGlobalReturnsSingleton(t *testing.T) {
	first := Global()
	second := Global()

	if first == nil || second == nil {
		t.Fatal("expected global loader instances")
	}
	if first != second {
		t.Fatalf("expected singleton loader, got %p and %p", first, second)
	}
}

func TestGetMethodDescriptor(t *testing.T) {
	t.Run("returns descriptor for registered proto", func(t *testing.T) {
		loader := New()
		loader.RegisterProto("demo/greeter.proto", greeterProtoV1)

		md, err := loader.GetMethodDescriptor("demo.Greeter.SayHello")
		if err != nil {
			t.Fatalf("GetMethodDescriptor returned error: %v", err)
		}
		if got := string(md.FullName()); got != "demo.Greeter.SayHello" {
			t.Fatalf("unexpected full method name: %s", got)
		}
		if got := md.Input().FullName(); string(got) != "demo.SayHelloReq" {
			t.Fatalf("unexpected input message: %s", got)
		}

		cached, ok := loader.cache.Load("demo.Greeter.SayHello")
		if !ok {
			t.Fatal("expected descriptor to be cached")
		}
		if cached != md {
			t.Fatal("expected cached descriptor to match returned descriptor")
		}
	})

	t.Run("rejects invalid method format", func(t *testing.T) {
		loader := New()

		_, err := loader.GetMethodDescriptor("Greeter/SayHello")
		if err == nil {
			t.Fatal("expected invalid format error")
		}
		if !strings.Contains(err.Error(), "invalid method name format") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns missing app error when no protos registered", func(t *testing.T) {
		loader := New()

		_, err := loader.GetMethodDescriptor("demo.Greeter.SayHello")
		if err == nil {
			t.Fatal("expected missing proto error")
		}
		if !strings.Contains(err.Error(), "no registered proto files for app demo") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRegisterProtoInvalidatesCacheOnContentChange(t *testing.T) {
	loader := New()
	loader.RegisterProto("demo/greeter.proto", greeterProtoV1)

	first, err := loader.GetMethodDescriptor("demo.Greeter.SayHello")
	if err != nil {
		t.Fatalf("GetMethodDescriptor returned error: %v", err)
	}

	loader.RegisterProto("demo/greeter.proto", greeterProtoV1)
	if cached, ok := loader.cache.Load("demo.Greeter.SayHello"); !ok || cached != first {
		t.Fatal("expected cache to stay warm when proto content is unchanged")
	}

	loader.RegisterProto("demo/greeter.proto", greeterProtoV2)
	if _, ok := loader.cache.Load("demo.Greeter.SayHello"); ok {
		t.Fatal("expected cache to be invalidated when proto content changes")
	}

	if _, err := loader.GetMethodDescriptor("demo.Greeter.SayHello"); err == nil || !strings.Contains(err.Error(), "method not found") {
		t.Fatalf("expected old method lookup to fail after reload, got %v", err)
	}

	md, err := loader.GetMethodDescriptor("demo.Greeter.WaveHello")
	if err != nil {
		t.Fatalf("expected new method lookup to succeed: %v", err)
	}
	if got := string(md.FullName()); got != "demo.Greeter.WaveHello" {
		t.Fatalf("unexpected full method name: %s", got)
	}
}