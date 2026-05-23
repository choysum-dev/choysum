// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcauth

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	authEnabled          bool
	grpcAuthentication   bool
	entryAuthSkipMethods []string
	jobTokenAllowedSANs  []string
	internalKey          string
	serverEnvironment    string
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	opts := runtimeOptions{}
	if runtimeScope == nil {
		return opts
	}

	if authOpts, ok := scope.AuthRuntimeOptionsFromScope(runtimeScope); ok {
		opts.authEnabled = authOpts.Enabled
		opts.grpcAuthentication = authOpts.GrpcAuthentication
		opts.internalKey = authOpts.InternalKey
		if len(authOpts.JobTokenAllowedSANs) > 0 {
			opts.jobTokenAllowedSANs = append([]string(nil), authOpts.JobTokenAllowedSANs...)
		}
		if authOpts.GrpcEntryPolicy != nil {
			methods := make([]string, 0, len(authOpts.GrpcEntryPolicy))
			for methodKey, methodCfg := range authOpts.GrpcEntryPolicy {
				if methodCfg == nil || !methodCfg.SkipAuthentication {
					continue
				}
				methodKey = strings.TrimSpace(methodKey)
				if methodKey == "" {
					continue
				}
				methods = append(methods, methodKey)
			}
			opts.entryAuthSkipMethods = methods
		}
	}

	if serverOpts, ok := scope.ServerRuntimeOptionsFromScope(runtimeScope); ok {
		opts.serverEnvironment = serverOpts.Environment
	}

	return opts
}
