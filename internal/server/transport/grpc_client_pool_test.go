// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
)

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

func TestGRPCClientPoolLifecycle(t *testing.T) {
	pool, err := NewGRPCClientPool(GRPCClientPoolOptions{Address: "127.0.0.1:9527", MaxCachedConn: 1})
	if err != nil {
		t.Fatalf("NewGRPCClientPool() error = %v", err)
	}
	if pool.maxConns != 1 || pool.creds == nil {
		t.Fatalf("unexpected pool state: %#v", pool)
	}

	first, err := pool.Dial(context.Background(), "auth.User")
	if err != nil {
		t.Fatalf("Dial() first error = %v", err)
	}
	second, err := pool.Dial(context.Background(), "auth.User")
	if err != nil {
		t.Fatalf("Dial() second error = %v", err)
	}
	if first != second {
		t.Fatal("expected Dial() to reuse cached grpc connection")
	}

	if _, err := pool.Dial(context.Background(), "bad service"); err == nil {
		t.Fatal("expected invalid service name error")
	}
	_, err = pool.Dial(context.Background(), "auth.Role")
	var cacheFull *grpcclient.ConnCacheFullError
	if !errors.As(err, &cacheFull) {
		t.Fatalf("expected cache full error, got %v", err)
	}

	interceptor := pool.UnaryServerInterceptor()
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/Get"}, func(ctx context.Context, req any) (any, error) {
		dialer, ok := grpcclient.ServiceDialerFromContext(ctx)
		if !ok {
			t.Fatal("expected service dialer in interceptor context")
		}
		if dialer == nil {
			t.Fatal("expected non-nil service dialer")
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("UnaryServerInterceptor() error = %v", err)
	}

	pool.CloseAll()
	if len(pool.conns) != 0 {
		t.Fatalf("expected CloseAll() to clear cache, got %#v", pool.conns)
	}
}

func TestGRPCClientPoolTLSValidation(t *testing.T) {
	if _, err := NewGRPCClientPool(GRPCClientPoolOptions{Address: "127.0.0.1:9527", EnableTLS: true}); err == nil {
		t.Fatal("expected missing CA file error")
	}

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, []byte("not a pem"), 0o644); err != nil {
		t.Fatalf("write ca file: %v", err)
	}
	if _, err := NewGRPCClientPool(GRPCClientPoolOptions{Address: "127.0.0.1:9527", EnableTLS: true, TLSCaFile: caPath}); err == nil {
		t.Fatal("expected invalid CA parse error")
	}
}

func TestGRPCClientPoolTLSAndRegistryPaths(t *testing.T) {
	caPath, certPath, keyPath := writeTestTLSFiles(t)
	pool, err := NewGRPCClientPool(GRPCClientPoolOptions{
		Registry:      fakeRegistry{},
		Address:       "127.0.0.1:9527",
		EnableTLS:     true,
		TLSCaFile:     caPath,
		TLSCertFile:   certPath,
		TLSKeyFile:    keyPath,
		TLSServerName: " example.internal ",
	})
	if err != nil {
		t.Fatalf("NewGRPCClientPool() error = %v", err)
	}
	if pool.maxConns != 128 {
		t.Fatalf("maxConns = %d, want 128", pool.maxConns)
	}
	tlsCreds, ok := pool.creds.(interface {
		Info() credentials.ProtocolInfo
	})
	if !ok {
		t.Fatalf("expected TLS transport credentials, got %T", pool.creds)
	}
	if got := tlsCreds.Info().ServerName; got != "example.internal" {
		t.Fatalf("server name = %q, want example.internal", got)
	}

	conn, err := pool.Dial(context.Background(), "auth.User")
	if err != nil {
		t.Fatalf("Dial() with registry error = %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil grpc connection")
	}
	if len(pool.conns) != 1 {
		t.Fatalf("cached conns = %d, want 1", len(pool.conns))
	}
	pool.CloseAll()
	if len(pool.conns) != 0 {
		t.Fatalf("expected CloseAll() to clear registry cache, got %#v", pool.conns)
	}
}

func writeTestTLSFiles(t *testing.T) (caPath string, certPath string, keyPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.internal"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.internal"},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	dir := t.TempDir()
	caPath = filepath.Join(dir, "ca.pem")
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatalf("WriteFile(ca.pem) error = %v", err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatalf("WriteFile(cert.pem) error = %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}), 0o600); err != nil {
		t.Fatalf("WriteFile(key.pem) error = %v", err)
	}

	return caPath, certPath, keyPath
}

func TestHTTPServerAccessURL(t *testing.T) {
	tests := []struct {
		name              string
		configuredAddress string
		listenAddress     string
		scheme            string
		want              string
	}{
		{
			name:              "preserves explicit host",
			configuredAddress: "127.0.0.1:9527",
			listenAddress:     "127.0.0.1:9527",
			scheme:            "http",
			want:              "http://127.0.0.1:9527",
		},
		{
			name:              "maps wildcard host to localhost",
			configuredAddress: "0.0.0.0:9527",
			listenAddress:     "[::]:9527",
			scheme:            "https",
			want:              "https://localhost:9527",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := httpServerAccessURL(tt.configuredAddress, tt.listenAddress, tt.scheme); got != tt.want {
				t.Fatalf("httpServerAccessURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
