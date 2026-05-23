// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"net/http"
	"sync"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
)

type registrationBinding interface {
	Name() string
	ServiceDescs() ([]*grpc.ServiceDesc, error)
	ServiceScripts() []*jsengine.JsScript
	WebHandlers() (map[string]http.Handler, error)
}

type grpcRegistrationBinding interface {
	RegisterGRPC(server *grpc.Server) error
}

type registrationState struct {
	bindings     []registrationBinding
	grpcMethodMu sync.RWMutex
	grpcMethods  map[string]struct{}
}

type registrationBatchResult struct {
	Bindings    []registrationBinding
	GRPCMethods map[string]struct{}
	InitScripts []*jsengine.JsScript
}

type registrationBatch struct {
	owner  *registrationState
	result registrationBatchResult
}

func (r *registrationState) beginBatch() *registrationBatch {
	return &registrationBatch{
		owner: r,
		result: registrationBatchResult{
			GRPCMethods: map[string]struct{}{},
		},
	}
}

func (r *registrationState) hasGrpcMethod(fullMethod string) bool {
	r.grpcMethodMu.RLock()
	idx := r.grpcMethods
	r.grpcMethodMu.RUnlock()
	if idx == nil {
		return false
	}
	_, ok := idx[fullMethod]
	return ok
}

func (r *registrationState) registeredBindings() []registrationBinding {
	return append([]registrationBinding{}, r.bindings...)
}

func (r *registrationState) grpcMethodsSnapshot() map[string]struct{} {
	r.grpcMethodMu.RLock()
	defer r.grpcMethodMu.RUnlock()
	if r.grpcMethods == nil {
		return nil
	}
	clone := make(map[string]struct{}, len(r.grpcMethods))
	for key := range r.grpcMethods {
		clone[key] = struct{}{}
	}
	return clone
}

func (r *registrationState) storeBindings(bindings []registrationBinding, grpcMethods map[string]struct{}) {
	r.bindings = append([]registrationBinding{}, bindings...)
	r.grpcMethodMu.Lock()
	r.grpcMethods = cloneRegistrationMethods(grpcMethods)
	r.grpcMethodMu.Unlock()
}

func (b *registrationBatch) registerBinding(server *GRPCWebServer, binding registrationBinding) error {
	if err := b.registerGRPC(server, binding); err != nil {
		return err
	}
	b.registerWebHandlers(server.mux, binding)
	b.result.Bindings = append(b.result.Bindings, binding)
	b.result.InitScripts = append(b.result.InitScripts, binding.ServiceScripts()...)
	return nil
}

func (b *registrationBatch) registerGRPC(server *GRPCWebServer, binding registrationBinding) error {
	sds, err := binding.ServiceDescs()
	if err != nil {
		return xfmt.Errorf("Failed to get service descriptors: %w", err)
	}

	registeredByCustom := false
	if reg, ok := binding.(grpcRegistrationBinding); ok {
		if err := reg.RegisterGRPC(server.server); err != nil {
			return xfmt.Errorf("Failed to register grpc service implementation: %w", err)
		}
		registeredByCustom = true
	}

	for _, svc := range sds {
		if !registeredByCustom {
			if err := server.registerGRPCServiceDesc(svc, nil); err != nil {
				return xfmt.Errorf("Failed to register service endpoint: %w", err)
			}
		}
		b.indexGRPCMethods(svc)
		if registeredByCustom {
			if err := server.registerGRPCServiceEndpoint(svc.ServiceName); err != nil {
				return xfmt.Errorf("Failed to register service endpoint: %w", err)
			}
		}
	}
	return nil
}

func (b *registrationBatch) registerWebHandlers(mux *http.ServeMux, binding registrationBinding) {
	handlers, err := binding.WebHandlers()
	if err != nil {
		// Web handler discovery is intentionally best-effort.
	}
	for pattern, handler := range handlers {
		mux.Handle(pattern, handler)
	}
}

func (b *registrationBatch) indexGRPCMethods(svc *grpc.ServiceDesc) {
	if b.result.GRPCMethods == nil {
		b.result.GRPCMethods = map[string]struct{}{}
	}
	for _, method := range svc.Methods {
		b.result.GRPCMethods["/"+svc.ServiceName+"/"+method.MethodName] = struct{}{}
	}
	for _, stream := range svc.Streams {
		b.result.GRPCMethods["/"+svc.ServiceName+"/"+stream.StreamName] = struct{}{}
	}
}

func (b *registrationBatch) commit() registrationBatchResult {
	if b == nil || b.owner == nil {
		return registrationBatchResult{}
	}
	result := b.snapshot()
	b.owner.storeBindings(result.Bindings, result.GRPCMethods)
	return result
}

func (b *registrationBatch) snapshot() registrationBatchResult {
	if b == nil {
		return registrationBatchResult{}
	}
	return registrationBatchResult{
		Bindings:    append([]registrationBinding{}, b.result.Bindings...),
		GRPCMethods: cloneRegistrationMethods(b.result.GRPCMethods),
		InitScripts: append([]*jsengine.JsScript{}, b.result.InitScripts...),
	}
}

func cloneRegistrationMethods(methods map[string]struct{}) map[string]struct{} {
	if methods == nil {
		return nil
	}
	clone := make(map[string]struct{}, len(methods))
	for method := range methods {
		clone[method] = struct{}{}
	}
	return clone
}
