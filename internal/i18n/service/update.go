// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nservice

import (
	"fmt"
	"strings"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

func (s *Service) handleUpdateTerm(reqMap map[string]any) (any, error) {
	if s.appName == "" || s.appName == "core" {
		return nil, status.Error(codes.FailedPrecondition, "core application has no terminology table")
	}

	module := strings.TrimSpace(fmt.Sprintf("%v", reqMap["module"]))
	lang := strings.TrimSpace(fmt.Sprintf("%v", reqMap["lang"]))
	scopeKey := strings.TrimSpace(fmt.Sprintf("%v", reqMap["scope"]))
	src := strings.TrimSpace(fmt.Sprintf("%v", reqMap["src"]))
	value := fmt.Sprintf("%v", reqMap["value"])
	if value == "<nil>" {
		value = ""
	}
	kind := strings.TrimSpace(fmt.Sprintf("%v", reqMap["kind"]))
	if kind == "" || kind == "<nil>" {
		kind = i18nmodels.KindLiteral
	}

	if module == "" || module == "<nil>" {
		return nil, status.Error(codes.InvalidArgument, "module is required")
	}
	if lang == "" || lang == "<nil>" {
		return nil, status.Error(codes.InvalidArgument, "lang is required")
	}
	if scopeKey == "" || scopeKey == "<nil>" {
		return nil, status.Error(codes.InvalidArgument, "scope is required")
	}
	if src == "" || src == "<nil>" {
		return nil, status.Error(codes.InvalidArgument, "src is required")
	}

	if app, ok := s.registry.ApplicationForModule(module); ok && app != "" && app != s.appName {
		return nil, status.Error(codes.PermissionDenied, "module belongs to another application")
	}

	ts := s.registry.StoreFor(s.appName)
	item, err := ts.UpsertOverride(module, lang, scopeKey, src, kind, value)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.registry.RememberModuleApplication(module, s.appName)

	return s.buildUpdateTermResp(item, ts.TermHash(lang))
}

func (s *Service) buildUpdateTermResp(item store.TermListItem, hash string) (any, error) {
	_, _, respDesc, err := methodDescriptors(s.appName, MethodUpdateTerm)
	if err != nil {
		return nil, err
	}
	respMsg := dynamicpb.NewMessage(respDesc)
	payload := map[string]any{
		"item": map[string]any{
			"application": item.Application,
			"module":      item.Module,
			"scope":       item.Scope,
			"src":         item.Src,
			"value":       item.Value,
			"kind":        item.Kind,
			"source":      item.Source,
			"status":      item.Status,
		},
		"hash": hash,
	}
	if err := converter.MapToMessage(payload, respMsg); err != nil {
		return nil, err
	}
	return respMsg, nil
}
