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
  rpc SearchTerms(SearchTermsReq) returns (SearchTermsResp);
  rpc UpdateTerm(UpdateTermReq) returns (UpdateTermResp);
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
  // S7: non-literal kinds (module → scope → kind → src → value). Literal stays in terms_by_module.
  google.protobuf.Struct metadata_by_module = 5;
}

message SearchTermsReq {
  string lang = 1;
  repeated string modules = 2;
  string q = 3;
  int32 limit = 4;
  int32 offset = 5;
}

message TermItem {
  string application = 1;
  string module = 2;
  string scope = 3;
  string src = 4;
  string value = 5;
  string kind = 6;
  string source = 7;
  string status = 8;
}

message SearchTermsResp {
  string lang = 1;
  repeated TermItem items = 2;
  int32 total = 3;
  int32 limit = 4;
  int32 offset = 5;
}

message UpdateTermReq {
  string module = 1;
  string lang = 2;
  string scope = 3;
  string src = 4;
  string value = 5;
  string kind = 6;
}

message UpdateTermResp {
  TermItem item = 1;
  string hash = 2;
}
`

const (
	MethodGetTranslations = "GetTranslations"
	MethodSearchTerms     = "SearchTerms"
	MethodUpdateTerm      = "UpdateTerm"
)

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
	methods map[string]protoreflect.MethodDescriptor
}

// ProtoPath returns the synthetic proto path for an application I18n service.
func ProtoPath(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "i18n"
	}
	return fmt.Sprintf(i18nProtoPathFmt, name)
}

// FullMethod returns /{app}.I18n/{method}.
func FullMethod(appName, method string) string {
	app := strings.TrimSpace(appName)
	method = strings.TrimSpace(method)
	if method == "" {
		method = MethodGetTranslations
	}
	return "/" + app + ".I18n/" + method
}

// FullMethodGetTranslations is the legacy helper for GetTranslations.
func FullMethodGetTranslations(appName string) string {
	return FullMethod(appName, MethodGetTranslations)
}

// ResetDescriptorCacheForTests clears compiled I18n descriptors (unit tests only).
func ResetDescriptorCacheForTests() {
	i18nMu.Lock()
	defer i18nMu.Unlock()
	i18nCache = map[string]*i18nDescriptorSet{}
	i18nErrs = map[string]error{}
}

func descriptors(appName string) (*i18nDescriptorSet, error) {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "i18n"
	}

	i18nMu.Lock()
	if set, ok := i18nCache[name]; ok {
		i18nMu.Unlock()
		return set, nil
	}
	if err, ok := i18nErrs[name]; ok {
		i18nMu.Unlock()
		return nil, err
	}
	i18nMu.Unlock()

	path := ProtoPath(name)
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
	set := &i18nDescriptorSet{methods: make(map[string]protoreflect.MethodDescriptor, methods.Len())}
	for i := 0; i < methods.Len(); i++ {
		md := methods.Get(i)
		set.methods[string(md.Name())] = md
	}
	for _, required := range []string{MethodGetTranslations, MethodSearchTerms, MethodUpdateTerm} {
		if _, ok := set.methods[required]; !ok {
			return storeError(name, fmt.Errorf("i18n proto missing method %s", required))
		}
	}

	i18nMu.Lock()
	i18nCache[name] = set
	delete(i18nErrs, name)
	i18nMu.Unlock()
	return set, nil
}

func methodDescriptors(appName, methodName string) (protoreflect.MethodDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	set, err := descriptors(appName)
	if err != nil {
		return nil, nil, nil, err
	}
	md, ok := set.methods[strings.TrimSpace(methodName)]
	if !ok {
		return nil, nil, nil, fmt.Errorf("i18n method not found: %s", methodName)
	}
	return md, md.Input(), md.Output(), nil
}

func storeError(appName string, err error) (*i18nDescriptorSet, error) {
	i18nMu.Lock()
	i18nErrs[appName] = err
	delete(i18nCache, appName)
	i18nMu.Unlock()
	return nil, err
}
