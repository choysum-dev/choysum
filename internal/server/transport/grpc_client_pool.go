// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"

	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/registry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClientPoolOptions struct {
	Registry      registry.Registry
	Address       string
	MaxCachedConn int
	EnableTLS     bool
	TLSCaFile     string
	TLSServerName string
	TLSCertFile   string
	TLSKeyFile    string
}

type GRPCClientPool struct {
	registry registry.Registry
	address  string
	creds    credentials.TransportCredentials
	maxConns int

	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
}

func NewGRPCClientPool(opts GRPCClientPoolOptions) (*GRPCClientPool, error) {
	maxConns := opts.MaxCachedConn
	if maxConns <= 0 {
		maxConns = 128
	}

	var backendCreds credentials.TransportCredentials
	if opts.EnableTLS {
		caFile := strings.TrimSpace(opts.TLSCaFile)
		if caFile == "" {
			return nil, fmt.Errorf("missing TLS CA file for grpc client")
		}
		caBytes, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("failed to load TLS CA file")
		}
		tlsCfg := &tls.Config{
			RootCAs:    rootCAs,
			ServerName: strings.TrimSpace(opts.TLSServerName),
		}
		certFile := strings.TrimSpace(opts.TLSCertFile)
		keyFile := strings.TrimSpace(opts.TLSKeyFile)
		if certFile != "" && keyFile != "" {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, err
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
		backendCreds = credentials.NewTLS(tlsCfg)
	} else {
		backendCreds = insecure.NewCredentials()
	}

	return &GRPCClientPool{
		registry: opts.Registry,
		address:  opts.Address,
		creds:    backendCreds,
		maxConns: maxConns,
		conns:    map[string]*grpc.ClientConn{},
	}, nil
}

func (p *GRPCClientPool) MaxConns() int {
	if p == nil {
		return 0
	}
	return p.maxConns
}

func (p *GRPCClientPool) Dial(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	if err := grpcclient.ValidateServiceName(serviceName); err != nil {
		return nil, err
	}

	p.mu.Lock()
	if p.conns == nil {
		p.conns = map[string]*grpc.ClientConn{}
	}
	if cc, ok := p.conns[serviceName]; ok {
		p.mu.Unlock()
		return cc, nil
	}
	current := len(p.conns)
	if current >= p.maxConns {
		p.mu.Unlock()
		return nil, &grpcclient.ConnCacheFullError{ServiceName: serviceName, Max: p.maxConns, Current: current}
	}
	p.mu.Unlock()

	target := p.address
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(p.creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
	if p.registry != nil {
		target = fmt.Sprintf("%s:///%s", p.registry.Scheme(), serviceName)
		dialOptions = append(dialOptions,
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
			grpc.WithResolvers(p.registry.Resolver()),
		)
	}

	cc, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if existing, ok := p.conns[serviceName]; ok {
		p.mu.Unlock()
		_ = cc.Close()
		return existing, nil
	}
	if p.conns == nil {
		p.conns = map[string]*grpc.ClientConn{}
	}
	p.conns[serviceName] = cc
	p.mu.Unlock()
	return cc, nil
}

func (p *GRPCClientPool) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = grpcclient.ContextWithServiceDialer(ctx, p.Dial)
		return handler(ctx, req)
	}
}

func (p *GRPCClientPool) CloseAll() {
	p.mu.Lock()
	conns := p.conns
	p.conns = map[string]*grpc.ClientConn{}
	p.mu.Unlock()

	for _, cc := range conns {
		_ = cc.Close()
	}
}
