// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nservice

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const i18nProtoPathFmt = "i18n/%s.proto"

const i18nProtoTemplate = `syntax = "proto3";
package %s;
import "google/protobuf/struct.proto";

service I18n {
  rpc GetTranslations(GetTranslationsReq) returns (GetTranslationsResp);
}

message GetTranslationsReq {
  string lang = 1;
  repeated string module_names = 2;
  string hash = 3;
}

message GetTranslationsResp {
  string lang = 1;
  string hash = 2;
  bool unchanged = 3;
  google.protobuf.Struct terms_by_module = 4;
}
`

type i18nResolver struct {
	path    string
	content string
}

func (r *i18nResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if path == r.path {
		return protocompile.SearchResult{Source: strings.NewReader(r.content)}, nil
	}
	return protocompile.SearchResult{}, fmt.Errorf("proto not found: %s", path)
}

var (
	i18nMu    sync.Mutex
	i18nCache = map[string]*i18nDescriptorSet{}
	i18nErrs  = map[string]error{}
)

type i18nDescriptorSet struct {
	method protoreflect.MethodDescriptor
	req    protoreflect.MessageDescriptor
	resp   protoreflect.MessageDescriptor
}

func ProtoPath(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "i18n"
	}
	return fmt.Sprintf(i18nProtoPathFmt, name)
}

func descriptors(appName string) (protoreflect.MethodDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "i18n"
	}
	path := ProtoPath(name)

	i18nMu.Lock()
	if set, ok := i18nCache[name]; ok {
		i18nMu.Unlock()
		return set.method, set.req, set.resp, nil
	}
	if err, ok := i18nErrs[name]; ok {
		i18nMu.Unlock()
		return nil, nil, nil, err
	}
	i18nMu.Unlock()

	content := fmt.Sprintf(i18nProtoTemplate, name)
	resolver := &i18nResolver{path: path, content: content}
	compiler := protocompile.Compiler{Resolver: protocompile.WithStandardImports(resolver)}
	files, err := compiler.Compile(context.TODO(), path)
	if err != nil {
		return storeError(name, err)
	}
	fd := files[0]
	services := fd.Services()
	if services.Len() == 0 {
		return storeError(name, fmt.Errorf("i18n proto missing service"))
	}
	serviceDesc := services.Get(0)
	methods := serviceDesc.Methods()
	if methods.Len() == 0 {
		return storeError(name, fmt.Errorf("i18n proto missing method"))
	}
	methodDesc := methods.Get(0)
	set := &i18nDescriptorSet{
		method: methodDesc,
		req:    methodDesc.Input(),
		resp:   methodDesc.Output(),
	}

	i18nMu.Lock()
	i18nCache[name] = set
	delete(i18nErrs, name)
	i18nMu.Unlock()
	return set.method, set.req, set.resp, nil
}

func storeError(appName string, err error) (protoreflect.MethodDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	i18nMu.Lock()
	i18nErrs[appName] = err
	delete(i18nCache, appName)
	i18nMu.Unlock()
	return nil, nil, nil, err
}
