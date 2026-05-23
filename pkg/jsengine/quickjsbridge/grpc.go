// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

func attachChoysumErrorInfo(ctx *quickjs.Context, errObj *quickjs.Value, callErr error) {
	if ctx == nil || errObj == nil || callErr == nil {
		return
	}
	st, ok := status.FromError(callErr)
	if !ok {
		return
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*oerrors.ErrorInfo)
		if !ok || info == nil {
			continue
		}

		if value := strings.TrimSpace(info.GetDomain()); value != "" {
			errObj.Set("domain", ctx.String(value))
		}
		if value := strings.TrimSpace(info.GetCode()); value != "" {
			errObj.Set("code", ctx.String(value))
		}
		if value := strings.TrimSpace(info.GetErrorId()); value != "" {
			errObj.Set("errorId", ctx.String(value))
		}
		if info.GetGrpcCode() != 0 {
			errObj.Set("grpcCode", ctx.Int32(info.GetGrpcCode()))
		}
		if metadataMap := info.GetMetadata(); len(metadataMap) != 0 {
			metadataObj := ctx.Object()
			for key, value := range metadataMap {
				if strings.TrimSpace(key) == "" {
					continue
				}
				metadataObj.Set(key, ctx.String(value))
			}
			errObj.Set("metadata", metadataObj)
		}
		return
	}
}

func buildTrustedOutgoingMetadata(execCtx context.Context) (metadata.MD, error) {
	md := metadata.New(nil)

	if token, ok := auth.AccessTokenFromContext(execCtx); ok {
		authorization := "Bearer " + token
		if strings.ContainsAny(authorization, "\r\n") {
			return nil, fmt.Errorf("invalid authorization header")
		}
		md.Set("authorization", authorization)
	} else if key, ok := auth.InternalKeyFromContext(execCtx); ok {
		if strings.ContainsAny(key, "\r\n") {
			return nil, fmt.Errorf("invalid internal key header")
		}
		md.Set("x-choysum-internal-key", key)
	} else {
		return nil, fmt.Errorf("missing access token in host context")
	}

	if incoming, ok := metadata.FromIncomingContext(execCtx); ok {
		for _, key := range []string{"traceparent", "tracestate", "baggage"} {
			values := incoming.Get(key)
			if len(values) > 0 {
				value := values[0]
				if value != "" && !strings.ContainsAny(value, "\r\n") {
					md.Set(key, value)
				}
			}
		}

		depth := 0
		if values := incoming.Get("x-choysum-depth"); len(values) > 0 {
			if parsed, err := strconv.Atoi(strings.TrimSpace(values[0])); err == nil && parsed >= 0 {
				depth = parsed
			}
		}
		md.Set("x-choysum-depth", strconv.Itoa(depth+1))
	}

	if peerInfo, ok := peer.FromContext(execCtx); ok && peerInfo.Addr != nil {
		clientIP := peerInfo.Addr.String()
		if clientIP != "" && !strings.ContainsAny(clientIP, "\r\n") {
			md.Set("x-choysum-client-ip", clientIP)
		}
	}

	md.Set("user-agent", "choysum-quickjs/1.0")
	md.Set("x-choysum-jsclient", "1")

	return md, nil
}

func cloneMetadata(source metadata.MD) metadata.MD {
	if len(source) == 0 {
		return metadata.MD{}
	}
	dup := metadata.MD{}
	for key, values := range source {
		dup[key] = append([]string(nil), values...)
	}
	return dup
}

type grpcMethodNames struct {
	descriptor string
	invoke     string
}

func methodNamesFromServiceParts(service, method string) (grpcMethodNames, error) {
	var names grpcMethodNames
	serviceName := strings.Trim(strings.TrimSpace(service), "/")
	methodName := strings.TrimSpace(method)
	if serviceName == "" {
		return names, fmt.Errorf("service name cannot be empty")
	}
	if methodName == "" {
		return names, fmt.Errorf("method name cannot be empty")
	}
	names.descriptor = fmt.Sprintf("%s.%s", serviceName, methodName)
	names.invoke = fmt.Sprintf("/%s/%s", serviceName, methodName)
	return names, nil
}

func unaryCallFunc(protoLoader *loader.ProtoLoader, engine *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		serviceName := args[0].String()
		methodName := args[1].String()
		dataValue := args[2]

		execCtx := context.Background()
		if engine != nil {
			if hostCtx := engine.ExecContext(); hostCtx != nil {
				execCtx = hostCtx
			}
		}

		trustedMetadata, metadataErr := buildTrustedOutgoingMetadata(execCtx)
		if metadataErr != nil {
			return ctx.ThrowError(metadataErr)
		}
		outgoingMetadata := cloneMetadata(trustedMetadata)

		return ctx.NewPromise(func(resolve func(*quickjs.Value), reject func(*quickjs.Value)) {
			names, err := methodNamesFromServiceParts(serviceName, methodName)
			if err != nil {
				reject(ctx.NewError(err))
				return
			}

			var dataMap map[string]interface{}
			if err := ctx.Unmarshal(dataValue, &dataMap); err != nil {
				reject(ctx.NewError(fmt.Errorf("failed to convert data: %w", err)))
				return
			}

			go func(payload map[string]interface{}, metadataCopy metadata.MD, base context.Context) {
				requestCtx := metadata.NewOutgoingContext(base, metadataCopy)
				response, callErr := doUnaryCall(requestCtx, protoLoader, serviceName, names.descriptor, names.invoke, payload)

				ctx.Schedule(func(inner *quickjs.Context) {
					if callErr != nil {
						errorObj := inner.NewError(callErr)
						attachChoysumErrorInfo(inner, errorObj, callErr)
						defer errorObj.Free()
						reject(errorObj)
						return
					}

					jsResponse, marshalErr := inner.Marshal(response)
					if marshalErr != nil {
						errorObj := inner.NewError(fmt.Errorf("failed to convert result: %v", marshalErr))
						defer errorObj.Free()
						reject(errorObj)
						return
					}
					defer jsResponse.Free()
					resolve(jsResponse)
				})
			}(dataMap, outgoingMetadata, execCtx)
		})
	}
}

func doUnaryCall(ctx context.Context, protoLoader *loader.ProtoLoader, serviceName string, descriptorMethod string, invokeMethod string, data map[string]interface{}) (map[string]interface{}, error) {
	methodDescriptor, err := protoLoader.GetMethodDescriptor(descriptorMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to load method descriptor: %w", err)
	}

	requestMessage := dynamicpb.NewMessage(methodDescriptor.Input())
	err = converter.MapToMessage(data, requestMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	conn, err := client.Dial(ctx, serviceName)
	if err != nil {
		return nil, client.ToStatusError(err)
	}

	responseMessage := dynamicpb.NewMessage(methodDescriptor.Output())
	err = conn.Invoke(ctx, invokeMethod, requestMessage, responseMessage)
	if err != nil {
		return nil, client.ToStatusError(err)
	}

	responseMap, err := converter.MessageToMap(responseMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return responseMap, nil
}

func registerProtoFunc(protoLoader *loader.ProtoLoader) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("registerProto expects 2 arguments: path, content"))
		}
		path := args[0].String()
		content := args[1].String()
		protoLoader.RegisterProto(path, content)
		return ctx.Undefined()
	}
}

func WithGrpc(runtimeScope scope.Scope) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		engine := jsEngine.(*quickjsengine.QuickjsEngine)
		ctx := engine.Ctx
		protoLoader := loader.Global()

		grpcObj := ctx.Object()
		grpcObj.Set("unary", ctx.Function(unaryCallFunc(protoLoader, engine)))
		grpcObj.Set("registerProto", ctx.Function(registerProtoFunc(protoLoader)))

		global := ctx.Globals()
		choysum := global.Get("$choysum")
		if choysum.IsUndefined() {
			choysum = ctx.Object()
		}

		choysum.Set("grpc", grpcObj)
		global.Set("$choysum", choysum)

		return nil
	}
}
