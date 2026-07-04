package serverconfig

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNewDefaultGrpcClientConfig(t *testing.T) {
	cfg := NewDefaultGrpcClientConfig()
	if cfg.MaxCachedConns != 128 {
		t.Fatalf("expected MaxCachedConns=128, got %d", cfg.MaxCachedConns)
	}
}

func TestNewDefaultServerConfig(t *testing.T) {
	cfg := NewDefaultServerConfig()
	if cfg.HotReload {
		t.Fatal("expected HotReload=false by default")
	}
	if cfg.HotReloadQueueSize != 8 {
		t.Fatalf("expected HotReloadQueueSize=8, got %d", cfg.HotReloadQueueSize)
	}
	if cfg.JsExecutorFactory != "default" {
		t.Fatalf("expected JsExecutorFactory=default, got %s", cfg.JsExecutorFactory)
	}
	if cfg.Port != 9527 {
		t.Fatalf("expected Port=9527, got %d", cfg.Port)
	}
	if cfg.GrpcClient == nil {
		t.Fatal("expected GrpcClient to be set")
	}
}

func TestApplyViperDefaults(t *testing.T) {
	t.Run("nil viper", func(t *testing.T) {
		err := ApplyViperDefaults(nil)
		if err == nil {
			t.Fatal("expected error for nil viper")
		}
	})

	t.Run("sets defaults", func(t *testing.T) {
		v := viper.New()
		if err := ApplyViperDefaults(v); err != nil {
			t.Fatalf("ApplyViperDefaults() error = %v", err)
		}
		if v.GetInt("server.hotReloadQueueSize") != 8 {
			t.Fatalf("expected hotReloadQueueSize=8, got %d", v.GetInt("server.hotReloadQueueSize"))
		}
		if v.GetString("server.jsExecutorFactory") != "default" {
			t.Fatalf("expected jsExecutorFactory=default, got %s", v.GetString("server.jsExecutorFactory"))
		}
	})
}
