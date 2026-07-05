// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	modulesPath             string
	distPath                string
	compileBundleMode       string
	bindAddress             string
	port                    int
	enableGzip              bool
	enabledTLS              bool
	tlsCaFile               string
	tlsServerName           string
	tlsCertFile             string
	tlsKeyFile              string
	enableGrpcWebProxy      bool
	hotReload               bool
	hotReloadQueueSize      int
	authEnabled             bool
	httpAuthEnabled         bool
	grpcClientMaxCachedConn int
	cspEnabled              bool
	csrfEnabled             bool
}

func defaultRuntimeOptions() runtimeOptions {
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false)
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool) runtimeOptions {
	serverDefaults := config.NewDefaultServerConfig()
	compileDefaults := config.NewDefaultCompileConfig()

	opts := runtimeOptions{
		compileBundleMode:       compileDefaults.BundleMode,
		bindAddress:             serverDefaults.BindAddress,
		port:                    serverDefaults.Port,
		enableGzip:              serverDefaults.EnableGzip,
		enabledTLS:              serverDefaults.EnabledTLS,
		tlsCaFile:               serverDefaults.TLSCaFile,
		tlsServerName:           serverDefaults.TLSServerName,
		tlsCertFile:             serverDefaults.TLSCertFile,
		tlsKeyFile:              serverDefaults.TLSKeyFile,
		enableGrpcWebProxy:      serverDefaults.EnableGrpcWebProxy,
		hotReload:               serverDefaults.HotReload,
		hotReloadQueueSize:      serverDefaults.HotReloadQueueSize,
		grpcClientMaxCachedConn: serverDefaults.GrpcClient.MaxCachedConns,
		cspEnabled:              serverDefaults.Security != nil && serverDefaults.Security.CSP != nil && serverDefaults.Security.CSP.Enabled,
		csrfEnabled:             serverDefaults.Security != nil && serverDefaults.Security.CSRF != nil && serverDefaults.Security.CSRF.Enabled,
	}

	if hasPathOpts {
		opts.modulesPath = pathOpts.ModulesPath
		opts.distPath = pathOpts.DistPath
	}

	if hasCompileOpts && strings.TrimSpace(compileOpts.BundleMode) != "" {
		opts.compileBundleMode = compileOpts.BundleMode
	}

	if hasServerOpts {
		if strings.TrimSpace(serverOpts.BindAddress) != "" {
			opts.bindAddress = serverOpts.BindAddress
		}
		if serverOpts.Port > 0 {
			opts.port = serverOpts.Port
		}
		opts.enableGzip = serverOpts.EnableGzip
		opts.enabledTLS = serverOpts.EnabledTLS
		opts.tlsCaFile = serverOpts.TLSCaFile
		opts.tlsServerName = serverOpts.TLSServerName
		opts.tlsCertFile = serverOpts.TLSCertFile
		opts.tlsKeyFile = serverOpts.TLSKeyFile
		opts.enableGrpcWebProxy = serverOpts.EnableGrpcWebProxy
		opts.hotReload = serverOpts.HotReload
		if serverOpts.HotReloadQueueSize > 0 {
			opts.hotReloadQueueSize = serverOpts.HotReloadQueueSize
		}
		if serverOpts.GrpcClientMaxCachedConns > 0 {
			opts.grpcClientMaxCachedConn = serverOpts.GrpcClientMaxCachedConns
		}
		if serverOpts.CSP != nil {
			opts.cspEnabled = serverOpts.CSP.Enabled
		} else if !serverOpts.SecurityMissing {
			opts.cspEnabled = false
		}
		if serverOpts.CSRF != nil {
			opts.csrfEnabled = serverOpts.CSRF.Enabled
		} else if !serverOpts.SecurityMissing {
			opts.csrfEnabled = false
		}
	}

	if hasAuthOpts {
		opts.authEnabled = authOpts.Enabled
		opts.httpAuthEnabled = authOpts.HttpAuth != nil && authOpts.HttpAuth.Enabled
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return defaultRuntimeOptions()
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.bindAddress) != ""
}

func (s *GRPCWebServer) resolvedRuntimeOptions() runtimeOptions {
	if s != nil && hasRuntimeOptions(s.runtimeOptions) {
		return s.runtimeOptions
	}
	if s != nil && s.runtimeScope != nil {
		return runtimeOptionsFromScope(s.runtimeScope)
	}
	if s != nil {
		return s.runtimeOptions
	}
	return defaultRuntimeOptions()
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("server runtime options: modulesPath is required")
	}
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("server runtime options: distPath is required")
	}
	if strings.TrimSpace(o.compileBundleMode) == "" {
		return xfmt.Errorf("server runtime options: compileBundleMode is required")
	}
	if strings.TrimSpace(o.bindAddress) == "" {
		return xfmt.Errorf("server runtime options: bindAddress is required")
	}
	if o.port <= 0 {
		return xfmt.Errorf("server runtime options: port must be positive")
	}
	if o.grpcClientMaxCachedConn <= 0 {
		return xfmt.Errorf("server runtime options: grpcClientMaxCachedConn must be positive")
	}
	return nil
}
