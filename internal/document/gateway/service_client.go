// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"google.golang.org/protobuf/types/dynamicpb"
)

func callDocumentAttachmentContentMethod(ctx context.Context, method string, req map[string]any) (map[string]any, error) {
	return callDocumentRPCMethod(ctx, documentAttachmentContentServiceName, method, req)
}

func callDocumentAttachmentBindingMethod(ctx context.Context, method string, req map[string]any) (map[string]any, error) {
	return callDocumentRPCMethod(ctx, documentAttachmentBindingServiceName, method, req)
}

func callDocumentRPCMethod(ctx context.Context, serviceName string, method string, req map[string]any) (map[string]any, error) {
	service := strings.TrimSpace(serviceName)
	if service == "" {
		return nil, fmt.Errorf("document rpc service name is required")
	}

	methodName := strings.TrimSpace(method)
	if methodName == "" {
		return nil, fmt.Errorf("document rpc method name is required")
	}

	descriptorMethod := fmt.Sprintf("%s.%s", service, methodName)
	invokeMethod := fmt.Sprintf("/%s/%s", service, methodName)

	md, err := loader.Global().GetMethodDescriptor(descriptorMethod)
	if err != nil {
		return nil, fmt.Errorf("load document rpc method descriptor: %w", err)
	}

	reqMsg := dynamicpb.NewMessage(md.Input())
	if err := converter.MapToMessage(req, reqMsg); err != nil {
		return nil, fmt.Errorf("convert document rpc request: %w", err)
	}

	conn, err := grpcclient.Dial(ctx, service)
	if err != nil {
		return nil, grpcclient.ToStatusError(err)
	}

	respMsg := dynamicpb.NewMessage(md.Output())
	if err := conn.Invoke(ctx, invokeMethod, reqMsg, respMsg); err != nil {
		return nil, grpcclient.ToStatusError(err)
	}

	respMap, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, fmt.Errorf("convert document rpc response: %w", err)
	}

	return respMap, nil
}

func parseOptionalInt64(value any) (int64, bool) {
	switch vv := value.(type) {
	case int:
		return int64(vv), true
	case int8:
		return int64(vv), true
	case int16:
		return int64(vv), true
	case int32:
		return int64(vv), true
	case int64:
		return vv, true
	case uint:
		return int64(vv), true
	case uint8:
		return int64(vv), true
	case uint16:
		return int64(vv), true
	case uint32:
		return int64(vv), true
	case uint64:
		if vv > ^uint64(0)>>1 {
			return 0, false
		}
		return int64(vv), true
	case float32:
		return int64(vv), true
	case float64:
		return int64(vv), true
	case string:
		text := strings.TrimSpace(vv)
		if text == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err == nil {
			return parsed, true
		}
		parsedFloat, floatErr := strconv.ParseFloat(text, 64)
		if floatErr != nil {
			return 0, false
		}
		return int64(parsedFloat), true
	default:
		return 0, false
	}
}
