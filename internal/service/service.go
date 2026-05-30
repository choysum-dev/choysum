// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ApplicationService struct {
	runtimeScope     scope.Scope
	name             string
	runtimeOptions   runtimeOptions
	appDistPath      string
	protoRootDir     string
	bundleMode       string
	jsExecutor       jsexecutor.ScriptExecutor
	protoImportPaths []string
	hasGrpcMethod    func(fullMethod string) bool
}

type Option func(*ApplicationService)

func WithHasGrpcMethod(fn func(fullMethod string) bool) Option {
	return func(s *ApplicationService) {
		s.hasGrpcMethod = fn
	}
}

// WithBundleMode overrides the dist layout mode for runtime path resolution.
// Allowed: "bundle" | "application".
func WithBundleMode(mode string) Option {
	return func(s *ApplicationService) {
		s.bundleMode = mode
	}
}

func (s *ApplicationService) Name() string {
	return s.name
}

func getJsReqMeta(jsCtx map[string]interface{}) (map[string]any, bool) {
	rm, ok := jsCtx["req"].(map[string]any)
	if !ok || rm == nil {
		return nil, false
	}
	return rm, true
}

func getJsReqDepth(rm map[string]any) int {
	if rm == nil {
		return 0
	}
	if v, ok := rm["depth"].(int); ok && v >= 0 {
		return v
	}
	return 0
}

func (s *ApplicationService) handleError(err error) (any, error) {
	return s.runtime().handleError(err)
}

func (s *ApplicationService) enforceMethodAccess(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string) error {
	return s.guard().enforceMethodAccess(ctx, runtimeScope, jsCtx, fullMethod)
}

// enforceMethodAccessStrict enforces ACL for top-level requests without honoring entry-policy skips.
// This is used by TaskWorker.ExecuteJob to prevent bypass via skipAuthentication/skipMethodAccess.
func (s *ApplicationService) enforceMethodAccessStrict(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string) error {
	return s.guard().enforceMethodAccessStrict(ctx, runtimeScope, jsCtx, fullMethod)
}

func (s *ApplicationService) executeUnary(
	ctx context.Context,
	runtimeScope scope.Scope,
	jsCtx map[string]interface{},
	routing *jsengine.JsExecutionRouting,
	packageName protoreflect.Name,
	serviceName protoreflect.Name,
	methodName protoreflect.Name,
	outputMsgDesc protoreflect.MessageDescriptor,
	reqMsg *dynamicpb.Message,
) (*dynamicpb.Message, error) {
	return s.runtime().executeUnary(ctx, runtimeScope, jsCtx, routing, packageName, serviceName, methodName, outputMsgDesc, reqMsg)
}

// buildJsContext builds the JS request context for the hardened protocol.
//
//	{
//	  ctx:      final business context assembled and sanitized by Go
//	  identity: minimal identity and authorization facts
//	  req:      request metadata such as trace, kind, and depth
//	}
func (s *ApplicationService) buildJsContext(ctx context.Context) map[string]interface{} {
	return s.runtime().buildJsContext(ctx)
}

// injectMethodMetaAndEntryPolicy mutates jsCtx["req"] with trusted method identity
// and per-method entry policy (depth=0 only).
func (s *ApplicationService) injectMethodMetaAndEntryPolicy(jsCtx map[string]interface{}, fullMethod string) {
	s.guard().injectMethodMetaAndEntryPolicy(jsCtx, fullMethod)
}

func (s *ApplicationService) unaryHandler(packageName protoreflect.Name, serviceName protoreflect.Name, methodName protoreflect.Name, outputMsgDesc protoreflect.MessageDescriptor) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		runtime := s.runtime()
		runtimeScope := s.runtimeScope.WithContext(ctx)
		var outMsg *dynamicpb.Message
		if err := runtimeScope.Transactor().Required(ctx, func(txScope scope.Scope, tx scope.Transaction) error {
			txCtx := tx.Context()
			fullMethod := fmt.Sprintf("/%s.%s/%s", packageName, serviceName, methodName)
			jsCtx := runtime.buildJsContext(txCtx)

			routing, err := s.guard().authorizeUnary(txCtx, txScope, jsCtx, fullMethod)
			if err != nil {
				return err
			}

			msg, err := runtime.executeUnary(txCtx, txScope, jsCtx, routing, packageName, serviceName, methodName, outputMsgDesc, req.(*dynamicpb.Message))
			if err != nil {
				return err
			}
			outMsg = msg
			return nil
		}); err != nil {
			return runtime.handleError(err)
		}

		if outMsg == nil {
			return nil, status.Error(codes.Internal, "outMsg is nil")
		}

		return outMsg, nil
	}
}

func (s *ApplicationService) methodHandler(packageName protoreflect.Name, serviceName protoreflect.Name, methodDescriptor protoreflect.MethodDescriptor) func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	unaryHandler := s.unaryHandler(packageName, serviceName, methodDescriptor.Name(), methodDescriptor.Output())
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		reqMsg := serviceCodec.newMessage(methodDescriptor.Input())
		if err := dec(reqMsg); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request format: %v", err))
		}
		if interceptor == nil {
			return unaryHandler(ctx, reqMsg)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: fmt.Sprintf("/%s.%s/%s", packageName, serviceName, methodDescriptor.Name()),
		}
		return interceptor(ctx, reqMsg, info, unaryHandler)
	}
}

func (s *ApplicationService) parseProtoFiles(protoFiles []string) ([]*grpc.ServiceDesc, error) {
	if len(s.protoImportPaths) == 0 {
		return nil, nil
	}
	// Use the first import root as the base for relative proto paths.
	// (We keep protoImportPaths as a slice to allow future multi-root support.)
	importRoot := s.protoImportPaths[0]
	protoFileNames := make([]string, 0, len(protoFiles))
	for _, file := range protoFiles {
		rel, err := filepath.Rel(importRoot, file)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		protoFileNames = append(protoFileNames, rel)
	}
	if len(protoFileNames) == 0 {
		return nil, nil
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: s.protoImportPaths,
		}),
	}
	fileDescriptors, err := compiler.Compile(s.runtimeScope.Context(), protoFileNames...)
	if err != nil {
		return nil, xfmt.Errorf("error parsing proto files: %w", err)
	}

	var serviceDescs []*grpc.ServiceDesc
	for _, fileDescriptor := range fileDescriptors {
		serviceDescriptors := fileDescriptor.Services()
		for i := 0; i < serviceDescriptors.Len(); i++ {
			serviceDescriptor := serviceDescriptors.Get(i)
			methodDescriptors := serviceDescriptor.Methods()

			var methodDescs []grpc.MethodDesc
			var streamDescs []grpc.StreamDesc

			for j := 0; j < methodDescriptors.Len(); j++ {
				methodDescriptor := methodDescriptors.Get(j)
				if methodDescriptor.IsPlaceholder() || methodDescriptor.IsStreamingClient() || methodDescriptor.IsStreamingServer() {
					continue
				}

				methodDescs = append(methodDescs, grpc.MethodDesc{
					MethodName: string(methodDescriptor.Name()),
					Handler:    s.methodHandler(fileDescriptor.Package().Name(), serviceDescriptor.Name(), methodDescriptor),
				})
			}

			serviceDescs = append(serviceDescs, &grpc.ServiceDesc{
				ServiceName: fmt.Sprintf("%s.%s", fileDescriptor.Package().Name(), serviceDescriptor.Name()),
				Metadata:    fileDescriptor.Name(),
				HandlerType: (*interface{})(nil),
				Methods:     methodDescs,
				Streams:     streamDescs,
			})
		}
	}
	return serviceDescs, nil
}

func (s *ApplicationService) ServiceDescs() ([]*grpc.ServiceDesc, error) {
	protDir := s.protoRootDir
	if strings.TrimSpace(protDir) == "" {
		protDir = filepath.Join(s.appDistPath, "assets")
	}
	if _, err := os.Stat(protDir); os.IsNotExist(err) {
		return nil, nil
	}

	protoFiles := make([]string, 0)
	if err := filepath.WalkDir(protDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	serviceDescs, err := s.parseProtoFiles(protoFiles)
	if err != nil {
		return nil, err
	}

	// Register app protos into the global loader for ExecuteJob routing.
	if len(protoFiles) > 0 && len(s.protoImportPaths) > 0 {
		importRoot := s.protoImportPaths[0]
		for _, file := range protoFiles {
			rel, err := filepath.Rel(importRoot, file)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if rel == "." || strings.HasPrefix(rel, "..") {
				continue
			}
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			loader.Global().RegisterProto(rel, string(content))
		}
	}

	// Inject TaskWorker descriptor for each app (non-web).
	if s.name != "web" {
		if tw, err := s.taskWorkerServiceDesc(); err == nil && tw != nil {
			serviceDescs = append(serviceDescs, tw)
		}
	}

	return serviceDescs, nil
}

func (s *ApplicationService) ServiceScripts() []*jsengine.JsScript {
	if s.name == "web" {
		return nil
	}
	scriptPath := filepath.Join(s.appDistPath, "index.js")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil
	}
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil
	}
	return []*jsengine.JsScript{
		{
			Content:  string(scriptContent),
			FileName: scriptPath,
		},
	}
}

func (s *ApplicationService) staticFileHandler(webPath string, stripPrefix string) http.Handler {
	fileServer := http.FileServer(http.Dir(webPath))
	fileHandler := http.StripPrefix(stripPrefix, fileServer)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, stripPrefix+"assets/") {
			s.serveAssetRequest(w, r, fileHandler)
			return
		}

		if path != stripPrefix {
			if filePath, ok := safeStaticPath(webPath, path, stripPrefix); ok {
				if fileInfo, err := os.Stat(filePath); err == nil && fileInfo.Mode().IsRegular() {
					fileHandler.ServeHTTP(w, r)
					return
				}
			}

			indexPath := filepath.Join(webPath, "index.html")
			indexContent, err := os.ReadFile(indexPath)
			if err == nil {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write(indexContent)
				return
			}
			s.runtimeScope.Logger().Debug("web request redirected", "path", path, "strip_prefix", stripPrefix)
			http.Redirect(w, r, stripPrefix, http.StatusFound)
			return
		}

		fileHandler.ServeHTTP(w, r)
	})
}

func safeStaticPath(webPath string, requestPath string, stripPrefix string) (string, bool) {
	if !strings.HasPrefix(requestPath, stripPrefix) {
		return "", false
	}

	relPath := strings.TrimPrefix(requestPath, stripPrefix)
	relPath = strings.TrimLeft(relPath, "/")
	relPath = filepath.Clean(relPath)

	baseAbsPath, err := filepath.Abs(webPath)
	if err != nil {
		return "", false
	}

	candidatePath := filepath.Join(baseAbsPath, relPath)
	candidateAbsPath, err := filepath.Abs(candidatePath)
	if err != nil {
		return "", false
	}

	relToBase, err := filepath.Rel(baseAbsPath, candidateAbsPath)
	if err != nil {
		return "", false
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(os.PathSeparator)) {
		return "", false
	}

	return candidateAbsPath, true
}

func (s *ApplicationService) serveAssetRequest(w http.ResponseWriter, r *http.Request, handler http.Handler) {
	writer := &assetStatusResponseWriter{ResponseWriter: w, status: http.StatusOK}
	handler.ServeHTTP(writer, r)
	s.logAssetRequestResult(r, writer.status)
}

func (s *ApplicationService) logAssetRequestResult(r *http.Request, status int) {
	attrs := []any{"method", r.Method, "path", r.URL.Path, "status", status}
	if status < http.StatusBadRequest {
		s.runtimeScope.Logger().Debug("web asset served", attrs...)
		return
	}
	if status >= http.StatusInternalServerError {
		s.runtimeScope.Logger().Error("web asset request failed", attrs...)
		return
	}
	s.runtimeScope.Logger().Warn("web asset request failed", attrs...)
}

type assetStatusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *assetStatusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *assetStatusResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (s *ApplicationService) WebHandlers() (map[string]http.Handler, error) {
	// if the service is web, serve the dist folder
	if s.name == "web" {
		opts := s.resolvedRuntimeOptions()
		handlers := make(map[string]http.Handler)
		webBaseUrl := strings.TrimSuffix(opts.webBaseURL, "/") + "/"
		handlers[webBaseUrl] = s.staticFileHandler(s.appDistPath, webBaseUrl)
		// Handle root-path redirection.
		if webBaseUrl != "/" {
			redirectTarget := webBaseUrl // Default redirect target is WebBaseURL.
			// Use RootRedirectURL when configured.
			if opts.rootRedirectURL != "" {
				redirectTarget = strings.TrimSuffix(opts.rootRedirectURL, "/") + "/"
			}
			handlers["/"] = http.RedirectHandler(redirectTarget, http.StatusFound)
		}
		return handlers, nil
	}

	return nil, nil
}

func NewApplicationService(runtimeScope scope.Scope, name string, jsExecutor jsexecutor.ScriptExecutor, opts ...Option) (*ApplicationService, error) {
	service := &ApplicationService{
		runtimeScope:   runtimeScope,
		name:           name,
		jsExecutor:     jsExecutor,
		runtimeOptions: runtimeOptionsFromScope(runtimeScope),
	}

	for _, opt := range opts {
		opt(service)
	}

	mode := strings.ToLower(strings.TrimSpace(service.bundleMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(service.runtimeOptions.bundleMode))
	}
	if mode == "" {
		mode = "bundle"
	}

	distPath := service.runtimeOptions.distPath

	// Resolve dist paths for runtime.
	if name == "web" {
		service.appDistPath = filepath.Join(distPath, "web")
		service.protoImportPaths = nil
		service.protoRootDir = ""
	} else if mode == "bundle" {
		service.appDistPath = filepath.Join(distPath, "bundles")
		service.protoRootDir = config.APIAppProtoDir(distPath, name)
		service.protoImportPaths = []string{config.APIAppProtoDir(distPath, name)}
	} else {
		service.appDistPath = filepath.Join(distPath, "apps", name)
		service.protoRootDir = config.APIAppProtoDir(distPath, name)
		service.protoImportPaths = []string{config.APIAppProtoDir(distPath, name)}
	}

	if _, err := os.Stat(service.appDistPath); os.IsNotExist(err) {
		return nil, err
	}

	return service, nil
}
