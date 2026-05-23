// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package loader

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoLoader struct {
	cache    sync.Map // map[string]protoreflect.MethodDescriptor
	memoryFs sync.Map // map[string]string
	group    singleflight.Group
}

var (
	globalOnce   sync.Once
	globalLoader *ProtoLoader
)

// Global returns the process-wide shared ProtoLoader.
//
// It is safe for concurrent use, and is intended to be shared across all JS
// engines/threads and internal services.
func Global() *ProtoLoader {
	globalOnce.Do(func() {
		globalLoader = New()
	})
	return globalLoader
}

func New() *ProtoLoader { return &ProtoLoader{} }

func (l *ProtoLoader) RegisterProto(path string, content string) {
	if v, ok := l.memoryFs.Load(path); ok {
		if existing, ok := v.(string); ok && existing == content {
			return
		}
	}

	l.memoryFs.Store(path, content)

	// Invalidate all cached method descriptors.
	// This keeps behavior correct under dev hot reload: next call recompiles using
	// the latest registered proto content.
	l.cache.Range(func(key, _ any) bool {
		l.cache.Delete(key)
		return true
	})
}

type memoryResolver struct {
	loader   *ProtoLoader
	fallback protocompile.Resolver
}

func (r *memoryResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if v, ok := r.loader.memoryFs.Load(path); ok {
		return protocompile.SearchResult{
			Source: strings.NewReader(v.(string)),
		}, nil
	}
	if r.fallback != nil {
		return r.fallback.FindFileByPath(path)
	}
	return protocompile.SearchResult{}, os.ErrNotExist
}

func (l *ProtoLoader) GetMethodDescriptor(fullMethodName string) (protoreflect.MethodDescriptor, error) {
	// 1. Check cache
	if v, ok := l.cache.Load(fullMethodName); ok {
		return v.(protoreflect.MethodDescriptor), nil
	}

	// 2. Parse service name to find app name
	// Format: appName.ServiceName.MethodName
	parts := strings.Split(fullMethodName, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid method name format: %s", fullMethodName)
	}
	appName := parts[0]

	// 3. Ensure proto files for app are compiled
	if _, err, _ := l.group.Do(appName, func() (interface{}, error) {
		return nil, l.compileAppProtos(appName)
	}); err != nil {
		return nil, err
	}

	// 4. Load from cache again
	if v, ok := l.cache.Load(fullMethodName); ok {
		return v.(protoreflect.MethodDescriptor), nil
	}

	return nil, fmt.Errorf("method not found: %s", fullMethodName)
}

func (l *ProtoLoader) compileAppProtos(appName string) error {
	protoFiles := l.appProtoFiles(appName)
	if len(protoFiles) == 0 {
		return fmt.Errorf("no registered proto files for app %s", appName)
	}

	resolver := &memoryResolver{
		loader:   l,
		fallback: protocompile.WithStandardImports(nil),
	}

	compiler := protocompile.Compiler{Resolver: resolver}
	fileDescriptors, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		return fmt.Errorf("failed to compile proto files: %w", err)
	}

	for _, fd := range fileDescriptors {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			sd := services.Get(i)
			for j := 0; j < sd.Methods().Len(); j++ {
				md := sd.Methods().Get(j)
				l.cache.Store(string(md.FullName()), md)
			}
		}
	}

	return nil
}

func (l *ProtoLoader) appProtoFiles(appName string) []string {
	prefix := appName + "/"
	files := make([]string, 0)
	l.memoryFs.Range(func(key, _ any) bool {
		path := key.(string)
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return true
	})
	return files
}
