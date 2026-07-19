// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nservice

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	defaultSearchLimit = 50
	// maxSearchLimit bounds editor UI pages and gateway PO export page size.
	maxSearchLimit = 500
)

func (s *Service) handleSearchTerms(reqMap map[string]any) (any, error) {
	lang := strings.TrimSpace(fmt.Sprintf("%v", reqMap["lang"]))
	if lang == "" || lang == "<nil>" {
		return nil, status.Error(codes.InvalidArgument, "lang is required")
	}
	q := strings.TrimSpace(fmt.Sprintf("%v", reqMap["q"]))
	if q == "<nil>" {
		q = ""
	}
	modules := parseStringList(reqMap["modules"])
	limit := parseInt32(reqMap["limit"], defaultSearchLimit)
	offset := parseInt32(reqMap["offset"], 0)
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	if offset < 0 {
		offset = 0
	}

	if s.appName == "" || s.appName == "core" {
		return s.buildSearchTermsResp(lang, nil, 0, limit, offset)
	}

	owned := s.filterOwnedModules(modules)
	// Empty modules filter means "all modules owned by this application".
	ts := s.registry.StoreFor(s.appName)
	items, total, err := ts.SearchTerms(lang, owned, q, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return s.buildSearchTermsResp(lang, items, total, limit, offset)
}

func (s *Service) buildSearchTermsResp(lang string, items []store.TermListItem, total int64, limit, offset int) (any, error) {
	_, _, respDesc, err := methodDescriptors(s.appName, MethodSearchTerms)
	if err != nil {
		return nil, err
	}
	respMsg := dynamicpb.NewMessage(respDesc)
	payloadItems := make([]any, 0, len(items))
	for _, item := range items {
		payloadItems = append(payloadItems, map[string]any{
			"application": item.Application,
			"module":      item.Module,
			"scope":       item.Scope,
			"src":         item.Src,
			"value":       item.Value,
			"kind":        item.Kind,
			"source":      item.Source,
			"status":      item.Status,
		})
	}
	payload := map[string]any{
		"lang":   lang,
		"items":  payloadItems,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	if err := converter.MapToMessage(payload, respMsg); err != nil {
		return nil, err
	}
	return respMsg, nil
}
