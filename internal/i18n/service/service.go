// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package i18nservice implements Go-native {app}.I18n RPCs (GetTranslations).
// Handlers never enter QuickJS; they read TermStore only.
package i18nservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Service is the per-application terminology read API.
type Service struct {
	appName      string
	runtimeScope scope.Scope
	registry     *store.Registry
}

// New creates an I18n service for one application.
func New(appName string, runtimeScope scope.Scope) *Service {
	return &Service{
		appName:      strings.TrimSpace(appName),
		runtimeScope: runtimeScope,
		registry:     store.RegistryFor(runtimeScope),
	}
}

// FullMethod returns /{app}.I18n/GetTranslations.
func FullMethod(appName string) string {
	return "/" + strings.TrimSpace(appName) + ".I18n/GetTranslations"
}

// NewRequestMessage builds an empty GetTranslationsReq for appName.
func NewRequestMessage(appName string) (*dynamicpb.Message, error) {
	_, reqDesc, _, err := descriptors(appName)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessage(reqDesc), nil
}

// NewResponseMessage builds an empty GetTranslationsResp for appName.
func NewResponseMessage(appName string) (*dynamicpb.Message, error) {
	_, _, respDesc, err := descriptors(appName)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessage(respDesc), nil
}

// ServiceDesc builds the gRPC service descriptor for {app}.I18n.
func (s *Service) ServiceDesc() (*grpc.ServiceDesc, error) {
	md, _, _, err := descriptors(s.appName)
	if err != nil {
		return nil, err
	}
	return &grpc.ServiceDesc{
		ServiceName: s.appName + ".I18n",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: string(md.Name()),
			Handler:    s.methodHandler(),
		}},
		Streams:  []grpc.StreamDesc{},
		Metadata: ProtoPath(s.appName),
	}, nil
}

func (s *Service) methodHandler() func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	unaryHandler := s.unaryHandler()
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		_, reqDesc, _, err := descriptors(s.appName)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		reqMsg := dynamicpb.NewMessage(reqDesc)
		if err := dec(reqMsg); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request format: %v", err))
		}
		if interceptor == nil {
			return unaryHandler(ctx, reqMsg)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: FullMethod(s.appName),
		}
		return interceptor(ctx, reqMsg, info, unaryHandler)
	}
}

func (s *Service) unaryHandler() grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		reqMsg, ok := req.(*dynamicpb.Message)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid request type")
		}
		reqMap, err := converter.MessageToMap(reqMsg)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		lang := strings.TrimSpace(fmt.Sprintf("%v", reqMap["lang"]))
		if lang == "" || lang == "<nil>" {
			return nil, status.Error(codes.InvalidArgument, "lang is required")
		}
		clientHash := strings.TrimSpace(fmt.Sprintf("%v", reqMap["hash"]))
		if clientHash == "<nil>" {
			clientHash = ""
		}
		moduleNames := parseStringList(reqMap["module_names"])

		result, err := s.getTranslations(lang, moduleNames, clientHash)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return s.buildResp(result)
	}
}

type getTranslationsResult struct {
	Lang          string
	Hash          string
	Unchanged     bool
	TermsByModule map[string]map[string]map[string]string
}

func (s *Service) getTranslations(lang string, moduleNames []string, clientHash string) (*getTranslationsResult, error) {
	if s.appName == "" || s.appName == "core" {
		hash := store.EmptyTermHash()
		return &getTranslationsResult{
			Lang:          lang,
			Hash:          hash,
			Unchanged:     clientHash != "" && clientHash == hash,
			TermsByModule: map[string]map[string]map[string]string{},
		}, nil
	}

	ts := s.registry.StoreFor(s.appName)
	if err := ts.WarmLanguage(lang); err != nil {
		return nil, err
	}
	termHash := ts.TermHash(lang)
	if termHash == "" {
		termHash = store.EmptyTermHash()
	}

	if clientHash != "" && clientHash == termHash {
		return &getTranslationsResult{
			Lang:          lang,
			Hash:          termHash,
			Unchanged:     true,
			TermsByModule: nil,
		}, nil
	}

	owned := s.filterOwnedModules(moduleNames)
	terms := ts.TermsByModules(lang, owned)
	if terms == nil {
		terms = map[string]map[string]map[string]string{}
	}
	return &getTranslationsResult{
		Lang:          lang,
		Hash:          termHash,
		Unchanged:     false,
		TermsByModule: terms,
	}, nil
}

func (s *Service) filterOwnedModules(moduleNames []string) []string {
	out := make([]string, 0, len(moduleNames))
	seen := map[string]struct{}{}
	for _, name := range moduleNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		if app, ok := s.registry.ApplicationForModule(name); ok && app != "" && app != s.appName {
			continue // belongs to another application
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func (s *Service) buildResp(result *getTranslationsResult) (any, error) {
	_, _, respDesc, err := descriptors(s.appName)
	if err != nil {
		return nil, err
	}
	respMsg := dynamicpb.NewMessage(respDesc)
	payload := map[string]any{
		"lang":      result.Lang,
		"hash":      result.Hash,
		"unchanged": result.Unchanged,
	}
	if !result.Unchanged {
		payload["terms_by_module"] = termsToStructMap(result.TermsByModule)
	}
	if err := converter.MapToMessage(payload, respMsg); err != nil {
		return nil, err
	}
	return respMsg, nil
}

func termsToStructMap(terms map[string]map[string]map[string]string) map[string]any {
	out := make(map[string]any, len(terms))
	for mod, byScope := range terms {
		modMap := make(map[string]any, len(byScope))
		for scopeKey, bySrc := range byScope {
			srcMap := make(map[string]any, len(bySrc))
			for src, val := range bySrc {
				srcMap[src] = val
			}
			modMap[scopeKey] = srcMap
		}
		out[mod] = modMap
	}
	return out
}

func parseStringList(v any) []string {
	if v == nil {
		return nil
	}
	switch typed := v.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
