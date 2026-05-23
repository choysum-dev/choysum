// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/dynamicpb"
)

type gatewayTestScope struct {
	ctx     context.Context
	session *scope.Session
	cfg     *config.Config
}

func (e *gatewayTestScope) Run(fn func(runtimeScope scope.Scope) error) error {
	return fn(e)
}

func (e *gatewayTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *gatewayTestScope) Session() *scope.Session {
	return e.session
}

func (e *gatewayTestScope) WithContext(ctx context.Context) scope.Scope {
	return &gatewayTestScope{ctx: ctx, session: e.session}
}

func (e *gatewayTestScope) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

func (e *gatewayTestScope) Logger() *slog.Logger {
	return slog.Default()
}

func (e *gatewayTestScope) Config() *config.Config {
	if e.cfg == nil {
		e.cfg = &config.Config{Document: config.NewDefaultDocumentConfig()}
	}
	return e.cfg
}

func (e *gatewayTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

type gatewayFakeIdentity struct {
	userID   string
	tokenID  string
	metadata map[string]any
}

func (f gatewayFakeIdentity) GetUserID() string {
	return f.userID
}

func (f gatewayFakeIdentity) GetTokenID() string {
	return f.tokenID
}

func (f gatewayFakeIdentity) GetMetadata() map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range f.metadata {
		out[k] = v
	}
	return out
}

func (f gatewayFakeIdentity) IsValid() bool {
	return f.userID != ""
}

func newGatewayTestScope(t *testing.T) *gatewayTestScope {
	t.Helper()

	return &gatewayTestScope{
		ctx:     context.Background(),
		session: &scope.Session{},
		cfg:     &config.Config{Document: config.NewDefaultDocumentConfig()},
	}
}

func newGatewayTestScopeWithBackend(t *testing.T, backend string) *gatewayTestScope {
	t.Helper()
	runtimeScope := newGatewayTestScope(t)
	if runtimeScope.cfg == nil {
		runtimeScope.cfg = &config.Config{Document: config.NewDefaultDocumentConfig()}
	}
	if runtimeScope.cfg.Document == nil {
		runtimeScope.cfg.Document = config.NewDefaultDocumentConfig()
	}
	if runtimeScope.cfg.Document.Attachment == nil {
		runtimeScope.cfg.Document.Attachment = config.NewDefaultDocumentConfig().Attachment
	}
	runtimeScope.cfg.Document.Attachment.Backend = backend
	if runtimeScope.cfg.Document.Attachment.S3 != nil {
		runtimeScope.cfg.Document.Attachment.S3.AccessKey = "test-access"
		runtimeScope.cfg.Document.Attachment.S3.SecretKey = "test-secret"
		runtimeScope.cfg.Document.Attachment.S3.Bucket = "choysum-attachments-test"
		runtimeScope.cfg.Document.Attachment.S3.Endpoint = "127.0.0.1:9000"
		runtimeScope.cfg.Document.Attachment.S3.UseTLS = false
	}
	return runtimeScope
}

func decodeGatewayErrorCode(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	return decodeGatewayErrorPayload(t, rr)["code"].(string)
}

func decodeGatewayErrorPayload(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json body: %v, body=%s", err, rr.Body.String())
	}
	return payload
}

func decodeGatewayErrorStage(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	payload := decodeGatewayErrorPayload(t, rr)
	metadataObj, ok := payload["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	stage, _ := metadataObj["stage"].(string)
	return stage
}

func decodeGatewayErrorMetadata(t *testing.T, rr *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	payload := decodeGatewayErrorPayload(t, rr)
	metadataObj, ok := payload["metadata"].(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for k, v := range metadataObj {
		if sv, ok := v.(string); ok {
			out[k] = sv
		}
	}
	return out
}

const gatewayDocumentAttachmentProto = `syntax = "proto3";

package document;

service AttachmentContent {
	rpc AuthorizeUploadPut (AttachmentContent_AuthorizeUploadPut_Req) returns (AttachmentContent_AuthorizeUploadPut_Resp) {}
	rpc CommitUploadPut (AttachmentContent_CommitUploadPut_Req) returns (AttachmentContent_CommitUploadPut_Resp) {}
}

service AttachmentBinding {
	rpc ResolveDownloadContent (AttachmentBinding_ResolveDownloadContent_Req) returns (AttachmentBinding_ResolveDownloadContent_Resp) {}
}

message PrincipalContext {
	string userId = 1;
	string activeCompanyId = 2;
	repeated string enabledCompanyIds = 3;
}

message RequestMeta {
	string contentType = 1;
	int64 contentLength = 2;
	string checksumSha256 = 3;
}

message AuthorizeUploadPutReq {
	string uploadId = 1;
	PrincipalContext principal = 2;
	RequestMeta requestMeta = 3;
}

message AuthorizeUploadPutResp {
	string uploadId = 1;
	int64 maxUploadBytes = 2;
	string requiredChecksumAlgorithm = 3;
	string expectedChecksumSha256 = 4;
	repeated string allowedMimeTypes = 5;
	string payloadWriteTicket = 6;
}

message PayloadReceipt {
	string payloadId = 1;
	int64 sizeBytes = 2;
	string checksumSha256 = 3;
	string contentType = 4;
}

message CommitUploadPutReq {
	string uploadId = 1;
	PrincipalContext principal = 2;
	PayloadReceipt payloadReceipt = 3;
}

message CommitUploadPutResp {
	string uploadId = 1;
	string attachmentUploadSessionStatus = 2;
	string attachmentContentId = 3;
}

message AttachmentContent_AuthorizeUploadPut_Req {
	AuthorizeUploadPutReq req = 1;
}

message AttachmentContent_AuthorizeUploadPut_Resp {
	AuthorizeUploadPutResp result = 1;
}

message AttachmentContent_CommitUploadPut_Req {
	CommitUploadPutReq req = 1;
}

message AttachmentContent_CommitUploadPut_Resp {
	CommitUploadPutResp result = 1;
}

message ResolveDownloadContentReq {
	string attachmentBindingId = 1;
	PrincipalContext principal = 2;
}

message ResolveDownloadContentResp {
	string attachmentBindingId = 1;
	string payloadReadTicket = 2;
	string mimeType = 3;
	int64 sizeBytes = 4;
	string checksumSha256 = 5;
	string fileName = 6;
	string downloadDisposition = 7;
	string etag = 8;
}

message AttachmentBinding_ResolveDownloadContent_Req {
	ResolveDownloadContentReq req = 1;
}

message AttachmentBinding_ResolveDownloadContent_Resp {
	ResolveDownloadContentResp result = 1;
}
`

type gatewayDocumentAttachmentService interface {
	AuthorizeUploadPut(context.Context, *dynamicpb.Message) (*dynamicpb.Message, error)
	CommitUploadPut(context.Context, *dynamicpb.Message) (*dynamicpb.Message, error)
}

type gatewayDocumentBindingService interface {
	ResolveDownloadContent(context.Context, *dynamicpb.Message) (*dynamicpb.Message, error)
}

type gatewayDocumentAttachmentFixture struct {
	authorizeResult map[string]any
	commitResult    map[string]any
	resolveResult   map[string]any
	authorizeErr    error
	commitErr       error
	resolveErr      error

	authorizeReq map[string]any
	commitReq    map[string]any
	resolveReq   map[string]any

	authorizeMetadata metadata.MD
	commitMetadata    metadata.MD
	resolveMetadata   metadata.MD
}

func newGatewayDocumentAttachmentFixture() *gatewayDocumentAttachmentFixture {
	registerGatewayDocumentProtoForTests()
	return &gatewayDocumentAttachmentFixture{
		authorizeResult: map[string]any{
			"maxUploadBytes":     1024,
			"payloadWriteTicket": "ticket",
		},
		commitResult: map[string]any{
			"attachmentUploadSessionStatus": "uploaded",
		},
		resolveResult: map[string]any{
			"attachmentBindingId": "bnd-001",
			"payloadReadTicket":   `{"attachmentBindingId":"bnd-001","storedContentId":"stored-001"}`,
			"mimeType":            "application/octet-stream",
			"sizeBytes":           int64(0),
			"fileName":            "attachment.bin",
			"downloadDisposition": "attachment",
		},
	}
}

func registerGatewayDocumentProtoForTests() {
	loader.Global().RegisterProto("document/proto/document.proto", gatewayDocumentAttachmentProto)
}

func (f *gatewayDocumentAttachmentFixture) AuthorizeUploadPut(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		f.authorizeMetadata = cloneOutgoingMetadata(md)
	} else {
		f.authorizeMetadata = metadata.MD{}
	}

	reqMap, err := converter.MessageToMap(req)
	if err != nil {
		return nil, err
	}
	f.authorizeReq = reqMap

	if f.authorizeErr != nil {
		return nil, f.authorizeErr
	}

	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentContent.AuthorizeUploadPut")
	if err != nil {
		return nil, err
	}
	resp := dynamicpb.NewMessage(methodDesc.Output())
	if err := converter.MapToMessage(map[string]any{"result": f.authorizeResult}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (f *gatewayDocumentAttachmentFixture) CommitUploadPut(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		f.commitMetadata = cloneOutgoingMetadata(md)
	} else {
		f.commitMetadata = metadata.MD{}
	}

	reqMap, err := converter.MessageToMap(req)
	if err != nil {
		return nil, err
	}
	f.commitReq = reqMap

	if f.commitErr != nil {
		return nil, f.commitErr
	}

	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentContent.CommitUploadPut")
	if err != nil {
		return nil, err
	}
	resp := dynamicpb.NewMessage(methodDesc.Output())
	if err := converter.MapToMessage(map[string]any{"result": f.commitResult}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (f *gatewayDocumentAttachmentFixture) ResolveDownloadContent(ctx context.Context, req *dynamicpb.Message) (*dynamicpb.Message, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		f.resolveMetadata = cloneOutgoingMetadata(md)
	} else {
		f.resolveMetadata = metadata.MD{}
	}

	reqMap, err := converter.MessageToMap(req)
	if err != nil {
		return nil, err
	}
	f.resolveReq = reqMap

	if f.resolveErr != nil {
		return nil, f.resolveErr
	}

	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentBinding.ResolveDownloadContent")
	if err != nil {
		return nil, err
	}
	resp := dynamicpb.NewMessage(methodDesc.Output())
	if err := converter.MapToMessage(map[string]any{"result": f.resolveResult}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

var gatewayDocumentAttachmentServiceDesc = grpc.ServiceDesc{
	ServiceName: documentAttachmentContentServiceName,
	HandlerType: (*gatewayDocumentAttachmentService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: documentAuthorizeUploadPutMethod, Handler: gatewayDocumentAuthorizeUploadPutHandler},
		{MethodName: documentCommitUploadPutMethod, Handler: gatewayDocumentCommitUploadPutHandler},
	},
}

var gatewayDocumentBindingServiceDesc = grpc.ServiceDesc{
	ServiceName: documentAttachmentBindingServiceName,
	HandlerType: (*gatewayDocumentBindingService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: documentResolveDownloadContentMethod, Handler: gatewayDocumentResolveDownloadContentHandler},
	},
}

func gatewayDocumentAuthorizeUploadPutHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	registerGatewayDocumentProtoForTests()
	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentContent.AuthorizeUploadPut")
	if err != nil {
		return nil, err
	}

	in := dynamicpb.NewMessage(methodDesc.Input())
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(gatewayDocumentAttachmentService).AuthorizeUploadPut(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/document.AttachmentContent/AuthorizeUploadPut"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(gatewayDocumentAttachmentService).AuthorizeUploadPut(ctx, req.(*dynamicpb.Message))
	}
	return interceptor(ctx, in, info, handler)
}

func gatewayDocumentCommitUploadPutHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	registerGatewayDocumentProtoForTests()
	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentContent.CommitUploadPut")
	if err != nil {
		return nil, err
	}

	in := dynamicpb.NewMessage(methodDesc.Input())
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(gatewayDocumentAttachmentService).CommitUploadPut(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/document.AttachmentContent/CommitUploadPut"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(gatewayDocumentAttachmentService).CommitUploadPut(ctx, req.(*dynamicpb.Message))
	}
	return interceptor(ctx, in, info, handler)
}

func gatewayDocumentResolveDownloadContentHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	registerGatewayDocumentProtoForTests()
	methodDesc, err := loader.Global().GetMethodDescriptor("document.AttachmentBinding.ResolveDownloadContent")
	if err != nil {
		return nil, err
	}

	in := dynamicpb.NewMessage(methodDesc.Input())
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(gatewayDocumentBindingService).ResolveDownloadContent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/document.AttachmentBinding/ResolveDownloadContent"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(gatewayDocumentBindingService).ResolveDownloadContent(ctx, req.(*dynamicpb.Message))
	}
	return interceptor(ctx, in, info, handler)
}

func newGatewayDocumentDialer(t *testing.T, svc gatewayDocumentAttachmentService) grpcclient.ServiceDialer {
	t.Helper()

	registerGatewayDocumentProtoForTests()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	grpcServer.RegisterService(&gatewayDocumentAttachmentServiceDesc, svc)
	if bindingSvc, ok := svc.(gatewayDocumentBindingService); ok {
		grpcServer.RegisterService(&gatewayDocumentBindingServiceDesc, bindingSvc)
	}
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	var conn *grpc.ClientConn
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})

	return func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != documentAttachmentContentServiceName && serviceName != documentAttachmentBindingServiceName {
			return nil, errors.New("unexpected service")
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(
			ctx,
			"bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
}

func newGatewayDocumentStatusError(grpcCode codes.Code, code string, message string, metadata map[string]string) error {
	st := status.New(grpcCode, message)
	errInfo := &oerrors.ErrorInfo{
		Domain:   "document",
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(message),
		GrpcCode: int32(grpcCode),
		Metadata: metadata,
	}

	stWithDetails, err := st.WithDetails(errInfo)
	if err != nil {
		return st.Err()
	}
	return stWithDetails.Err()
}

func TestDocumentGatewaySkeletonUploadRoute(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux)

	t.Run("put upload route returns not implemented", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/_document/uploads/up_001", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("json unmarshal error: %v", err)
		}
		if body["code"] != "document.SKELETON_NOT_IMPLEMENTED" {
			t.Fatalf("code = %v, want document.SKELETON_NOT_IMPLEMENTED", body["code"])
		}
	})

	t.Run("method mismatch returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_document/uploads/up_001", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}
		if allow := rr.Header().Get("Allow"); allow != http.MethodPut {
			t.Fatalf("allow = %q, want %q", allow, http.MethodPut)
		}
	})

	t.Run("missing upload id returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/_document/uploads/", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})
}

func TestDocumentGatewaySkeletonBindingContentRoute(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux)

	t.Run("get binding content route returns not implemented", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_document/bindings/bnd_001/content", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
		}
	})

	t.Run("method mismatch returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/_document/bindings/bnd_001/content", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}
		if allow := rr.Header().Get("Allow"); allow != http.MethodGet {
			t.Fatalf("allow = %q, want %q", allow, http.MethodGet)
		}
	})

	t.Run("invalid content path returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_document/bindings/bnd_001", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})
}

func TestDocumentGatewayUploadHandlerPersistsUploadedPayload(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.authorizeResult = map[string]any{
		"uploadId":           "up_db_001",
		"maxUploadBytes":     1024,
		"payloadWriteTicket": "ticket-db-001",
	}
	docFixture.commitResult = map[string]any{
		"uploadId":                      "up_db_001",
		"attachmentUploadSessionStatus": "uploaded",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadPut := payloadPutFunc
	var putReq payloadPutRequest
	payloadPutFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
		putReq = req
		return payloadPutReceipt{
			payloadID:      "sc:stored-put-db-001",
			sizeBytes:      int64(len(req.body)),
			checksumSHA256: req.checksumSHA256,
			contentType:    req.contentType,
		}, nil
	}
	t.Cleanup(func() { payloadPutFunc = prevPayloadPut })

	const uploadID = "up_db_001"
	body := []byte("hello-storage")
	checksum := checksumSHA256Hex(body)

	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/"+uploadID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "upload-db-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}

	if putReq.uploadID != uploadID {
		t.Fatalf("payload put uploadID = %q, want %q", putReq.uploadID, uploadID)
	}
	if putReq.payloadWriteTicket != "ticket-db-001" {
		t.Fatalf("payload put ticket = %q, want ticket-db-001", putReq.payloadWriteTicket)
	}
	if putReq.contentType != "text/plain" {
		t.Fatalf("payload put contentType = %q, want text/plain", putReq.contentType)
	}
	if putReq.checksumSHA256 != checksum {
		t.Fatalf("payload put checksum = %q, want %q", putReq.checksumSHA256, checksum)
	}
	if string(putReq.body) != string(body) {
		t.Fatalf("payload put body = %q, want %q", string(putReq.body), string(body))
	}

	authorizeReq := asRecord(docFixture.authorizeReq["req"])
	if authorizeReq == nil {
		t.Fatal("authorize req should not be nil")
	}
	if got := normalizeOptionalText(authorizeReq["uploadId"]); got != uploadID {
		t.Fatalf("authorize uploadId = %q, want %q", got, uploadID)
	}
	principal := asRecord(authorizeReq["principal"])
	if principal == nil {
		t.Fatal("authorize principal should not be nil")
	}
	if got := normalizeOptionalText(principal["userId"]); got != "u1" {
		t.Fatalf("principal.userId = %q, want u1", got)
	}
	if got := normalizeOptionalText(principal["activeCompanyId"]); got != "c1" {
		t.Fatalf("principal.activeCompanyId = %q, want c1", got)
	}
	requestMeta := asRecord(authorizeReq["requestMeta"])
	if requestMeta == nil {
		t.Fatal("authorize requestMeta should not be nil")
	}
	if got := normalizeOptionalText(requestMeta["contentType"]); got != "text/plain" {
		t.Fatalf("requestMeta.contentType = %q, want text/plain", got)
	}
	if got, ok := parseOptionalInt64(requestMeta["contentLength"]); !ok || got != int64(len(body)) {
		t.Fatalf("requestMeta.contentLength = %d, ok=%v, want %d", got, ok, len(body))
	}

	commitReq := asRecord(docFixture.commitReq["req"])
	if commitReq == nil {
		t.Fatal("commit req should not be nil")
	}
	payloadReceipt := asRecord(commitReq["payloadReceipt"])
	if payloadReceipt == nil {
		t.Fatal("commit payloadReceipt should not be nil")
	}
	if got := normalizeOptionalText(payloadReceipt["payloadId"]); got != "sc:stored-put-db-001" {
		t.Fatalf("payloadReceipt.payloadId = %q, want sc:stored-put-db-001", got)
	}
	if got, ok := parseOptionalInt64(payloadReceipt["sizeBytes"]); !ok || got != int64(len(body)) {
		t.Fatalf("payloadReceipt.sizeBytes = %d, ok=%v, want %d", got, ok, len(body))
	}
	if got := normalizeOptionalText(payloadReceipt["checksumSha256"]); got != checksum {
		t.Fatalf("payloadReceipt.checksumSha256 = %q, want %q", got, checksum)
	}

	if authz := docFixture.authorizeMetadata.Get("authorization"); len(authz) == 0 || authz[0] != "Bearer upload-db-token" {
		t.Fatalf("authorize metadata authorization = %v, want Bearer upload-db-token", authz)
	}
}

func TestDocumentGatewayUploadHandlerPersistsS3StagingRefWhenBackendS3(t *testing.T) {
	runtimeScope := newGatewayTestScopeWithBackend(t, "s3")
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.authorizeResult = map[string]any{
		"uploadId":           "up_s3_001",
		"maxUploadBytes":     1024,
		"payloadWriteTicket": `{"uploadId":"up_s3_001","companyId":"c1","activeCompanyId":"c1","userId":"u1","status":"prepared"}`,
	}
	docFixture.commitResult = map[string]any{
		"uploadId":                      "up_s3_001",
		"attachmentUploadSessionStatus": "uploaded",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	const uploadID = "up_s3_001"
	body := []byte("hello-storage-s3")
	checksum := checksumSHA256Hex(body)

	prevPayloadPut := payloadPutFunc
	putCalls := 0
	var gotUploadID string
	var gotContentType string
	var gotBody []byte
	var gotChecksum string
	payloadPutFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
		putCalls += 1
		gotUploadID = req.uploadID
		gotContentType = req.contentType
		gotBody = append([]byte(nil), req.body...)
		gotChecksum = req.checksumSHA256
		return payloadPutReceipt{
			payloadID:      "sc:stored-upload-s3-001",
			sizeBytes:      int64(len(body)),
			checksumSHA256: checksum,
			contentType:    "text/plain",
		}, nil
	}
	t.Cleanup(func() { payloadPutFunc = prevPayloadPut })

	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/"+uploadID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "upload-s3-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if putCalls != 1 {
		t.Fatalf("put calls = %d, want 1", putCalls)
	}
	if gotUploadID != uploadID {
		t.Fatalf("payload put uploadID = %q, want %q", gotUploadID, uploadID)
	}
	if gotContentType != "text/plain" {
		t.Fatalf("payload put contentType = %q, want text/plain", gotContentType)
	}
	if string(gotBody) != string(body) {
		t.Fatalf("payload put body = %q, want %q", string(gotBody), string(body))
	}
	if gotChecksum != checksum {
		t.Fatalf("payload put checksum = %q, want %q", gotChecksum, checksum)
	}

	commitReq := asRecord(docFixture.commitReq["req"])
	if commitReq == nil {
		t.Fatal("commit req should not be nil")
	}
	payloadReceipt := asRecord(commitReq["payloadReceipt"])
	if payloadReceipt == nil {
		t.Fatal("commit payloadReceipt should not be nil")
	}
	payloadID := normalizeOptionalText(payloadReceipt["payloadId"])
	if payloadID != "sc:stored-upload-s3-001" {
		t.Fatalf("payloadReceipt.payloadId = %q, want sc:stored-upload-s3-001", payloadID)
	}
	if got := normalizeOptionalText(payloadReceipt["checksumSha256"]); got != checksum {
		t.Fatalf("payloadReceipt.checksumSha256 = %q, want %q", got, checksum)
	}
	if got, ok := parseOptionalInt64(payloadReceipt["sizeBytes"]); !ok || got != int64(len(body)) {
		t.Fatalf("payloadReceipt.sizeBytes = %d, ok=%v, want %d", got, ok, len(body))
	}
}

func TestDocumentGatewayDownloadHandlerReturnsBindingContent(t *testing.T) {
	for _, backend := range []string{"db", "s3"} {
		backend := backend
		t.Run("backend="+backend, func(t *testing.T) {
			runtimeScope := newGatewayTestScopeWithBackend(t, backend)
			mux := http.NewServeMux()
			RegisterSkeletonHandlers(mux, runtimeScope)

			bindingID := "bnd_download_allow_" + backend
			body := []byte("gateway-download-" + backend)
			ticket := fmt.Sprintf(`{"attachmentBindingId":"%s","attachmentContentId":"att-download-db","storedContentId":"stored-download-db"}`, bindingID)
			if backend == "s3" {
				ticket = fmt.Sprintf(`{"attachmentBindingId":"%s","attachmentContentId":"att-download-s3","storedContentId":"stored-download-s3"}`, bindingID)
			}

			docFixture := newGatewayDocumentAttachmentFixture()
			docFixture.resolveResult = map[string]any{
				"attachmentBindingId": bindingID,
				"payloadReadTicket":   ticket,
				"mimeType":            "text/plain",
				"sizeBytes":           int64(len(body)),
				"checksumSha256":      checksumSHA256Hex(body),
				"fileName":            "doc.txt",
				"downloadDisposition": "attachment",
				"etag":                `"sha256:` + checksumSHA256Hex(body) + `"`,
			}
			dialer := newGatewayDocumentDialer(t, docFixture)

			prevPayloadOpen := payloadOpenFunc
			var openReq payloadOpenRequest
			payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
				openReq = req
				return payloadOpenResult{body: io.NopCloser(bytes.NewReader(body)), sizeBytes: int64(len(body))}, nil
			}
			t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

			req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
			ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
				userID:  "u1",
				tokenID: "tok-1",
				metadata: map[string]any{
					"activeCompanyId": "c1",
				},
			})
			ctx = auth.ContextWithAccessToken(ctx, "download-allow-token")
			ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if rr.Header().Get("Content-Type") != "text/plain" {
				t.Fatalf("content-type = %q, want text/plain", rr.Header().Get("Content-Type"))
			}
			if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("x-content-type-options = %q, want nosniff", rr.Header().Get("X-Content-Type-Options"))
			}
			if rr.Header().Get("Content-Disposition") != "attachment; filename=\"doc.txt\"; filename*=UTF-8''doc.txt" {
				t.Fatalf("content-disposition = %q, want attachment filename", rr.Header().Get("Content-Disposition"))
			}
			if rr.Header().Get("ETag") != `"sha256:`+checksumSHA256Hex(body)+`"` {
				t.Fatalf("etag = %q, want checksum etag", rr.Header().Get("ETag"))
			}
			if rr.Header().Get("Content-Length") != strconv.Itoa(len(body)) {
				t.Fatalf("content-length = %q, want %d", rr.Header().Get("Content-Length"), len(body))
			}
			if rr.Body.String() != string(body) {
				t.Fatalf("body = %q, want %q", rr.Body.String(), string(body))
			}

			if openReq.bindingID != bindingID {
				t.Fatalf("payload open bindingID = %q, want %q", openReq.bindingID, bindingID)
			}
			if openReq.payloadReadTicket != ticket {
				t.Fatalf("payload open ticket = %q, want %q", openReq.payloadReadTicket, ticket)
			}

			resolveReq := asRecord(docFixture.resolveReq["req"])
			if resolveReq == nil {
				t.Fatal("resolve download req should not be nil")
			}
			if got := normalizeOptionalText(resolveReq["attachmentBindingId"]); got != bindingID {
				t.Fatalf("resolve attachmentBindingId = %q, want %q", got, bindingID)
			}
			principal := asRecord(resolveReq["principal"])
			if principal == nil {
				t.Fatal("resolve principal should not be nil")
			}
			if got := normalizeOptionalText(principal["userId"]); got != "u1" {
				t.Fatalf("resolve principal.userId = %q, want u1", got)
			}
			if got := normalizeOptionalText(principal["activeCompanyId"]); got != "c1" {
				t.Fatalf("resolve principal.activeCompanyId = %q, want c1", got)
			}

			if authz := docFixture.resolveMetadata.Get("authorization"); len(authz) == 0 || authz[0] != "Bearer download-allow-token" {
				t.Fatalf("resolve metadata authorization = %v, want Bearer download-allow-token", authz)
			}
		})
	}
}

func TestDocumentGatewayDownloadHandlerReturnsInlineDisposition(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_download_inline"
	body := []byte("gateway-download-inline")
	ticket := `{"attachmentBindingId":"bnd_download_inline","storedContentId":"stored-inline"}`

	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": bindingID,
		"payloadReadTicket":   ticket,
		"mimeType":            "text/plain",
		"sizeBytes":           int64(len(body)),
		"checksumSha256":      checksumSHA256Hex(body),
		"fileName":            "note.txt",
		"downloadDisposition": "inline",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{body: io.NopCloser(bytes.NewReader(body)), sizeBytes: int64(len(body))}, nil
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-inline-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Header().Get("Content-Disposition") != "inline; filename=\"note.txt\"; filename*=UTF-8''note.txt" {
		t.Fatalf("content-disposition = %q, want inline filename", rr.Header().Get("Content-Disposition"))
	}
}

func TestDocumentGatewayDownloadHandlerAcceptsContextSessionWhenRuntimeScopeSessionIsNil(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	runtimeScope.session = nil
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_download_ctx_session"
	body := []byte("gateway-download-ctx-session")
	ticket := `{"attachmentBindingId":"bnd_download_ctx_session","storedContentId":"stored-ctx-session"}`

	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": bindingID,
		"payloadReadTicket":   ticket,
		"mimeType":            "text/plain",
		"sizeBytes":           int64(len(body)),
		"checksumSha256":      checksumSHA256Hex(body),
		"fileName":            "ctx.txt",
		"downloadDisposition": "attachment",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{body: io.NopCloser(bytes.NewReader(body)), sizeBytes: int64(len(body))}, nil
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-ctx-session-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	ctx = scope.ContextWithScope(ctx, &gatewayTestScope{ctx: ctx, session: &scope.Session{}})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != string(body) {
		t.Fatalf("body = %q, want %q", rr.Body.String(), string(body))
	}
}

func TestDocumentGatewayDownloadHandlerReturnsInternalWhenPayloadOpenFails(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_download_payload_open_err"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": bindingID,
		"payloadReadTicket":   `{"attachmentBindingId":"bnd_download_payload_open_err","storedContentId":"stored-open-fail"}`,
		"mimeType":            "text/plain",
		"sizeBytes":           int64(12),
		"fileName":            "doc.txt",
		"downloadDisposition": "attachment",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{}, errors.New("open failed")
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-open-fail-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.INTERNAL" {
		t.Fatalf("code = %q, want document.INTERNAL", code)
	}
}

func TestDocumentGatewayUploadHandlerMapsPayloadInvalidArgument(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.authorizeResult = map[string]any{
		"uploadId":           "up_db_payload_invalid_arg",
		"maxUploadBytes":     1024,
		"payloadWriteTicket": "ticket-invalid-arg",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadPut := payloadPutFunc
	payloadPutFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
		return payloadPutReceipt{}, documentpayload.InvalidArgument("payload write ticket is required")
	}
	t.Cleanup(func() { payloadPutFunc = prevPayloadPut })

	body := []byte("payload")
	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/up_db_payload_invalid_arg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "upload-invalid-arg-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.INVALID_ARGUMENT" {
		t.Fatalf("code = %q, want document.INVALID_ARGUMENT", code)
	}
}

func TestDocumentGatewayDownloadHandlerMapsPayloadNotFound(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_download_payload_not_found"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveResult = map[string]any{
		"attachmentBindingId": bindingID,
		"payloadReadTicket":   `{"attachmentBindingId":"bnd_download_payload_not_found","storedContentId":"stored-missing"}`,
		"mimeType":            "text/plain",
		"sizeBytes":           int64(12),
		"fileName":            "doc.txt",
		"downloadDisposition": "attachment",
	}
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadOpen := payloadOpenFunc
	payloadOpenFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadOpenRequest) (payloadOpenResult, error) {
		return payloadOpenResult{}, documentpayload.NotFound("active stored content not found")
	}
	t.Cleanup(func() { payloadOpenFunc = prevPayloadOpen })

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-not-found-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.NOT_FOUND" {
		t.Fatalf("code = %q, want document.NOT_FOUND", code)
	}
}

func TestDocumentGatewayUploadHandlerRejectsChecksumMismatch(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.authorizeResult = map[string]any{
		"uploadId":           "up_db_checksum_mismatch",
		"maxUploadBytes":     1024,
		"payloadWriteTicket": "ticket-checksum",
	}
	docFixture.commitErr = newGatewayDocumentStatusError(
		codes.FailedPrecondition,
		"CHECKSUM_MISMATCH",
		"checksum mismatch",
		map[string]string{"uploadId": "up_db_checksum_mismatch"},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadPut := payloadPutFunc
	payloadPutFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
		return payloadPutReceipt{
			payloadID:      "sc:stored-checksum",
			sizeBytes:      int64(len(req.body)),
			checksumSHA256: req.checksumSHA256,
			contentType:    req.contentType,
		}, nil
	}
	t.Cleanup(func() { payloadPutFunc = prevPayloadPut })

	const uploadID = "up_db_checksum_mismatch"
	body := []byte("actual-body")

	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/"+uploadID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "upload-checksum-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusUnprocessableEntity, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.CHECKSUM_MISMATCH" {
		t.Fatalf("code = %q, want document.CHECKSUM_MISMATCH", code)
	}
	if metadata := decodeGatewayErrorMetadata(t, rr); metadata["uploadId"] != uploadID {
		t.Fatalf("metadata.uploadId = %q, want %q", metadata["uploadId"], uploadID)
	}
}

func TestDocumentGatewayUploadHandlerRejectsMimeTypeNotAllowed(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.authorizeResult = map[string]any{
		"uploadId":           "up_db_mime_reject",
		"maxUploadBytes":     1024,
		"payloadWriteTicket": "ticket-mime",
	}
	docFixture.commitErr = newGatewayDocumentStatusError(
		codes.InvalidArgument,
		"MIME_TYPE_NOT_ALLOWED",
		"mime type is not allowed",
		map[string]string{"uploadId": "up_db_mime_reject", "contentType": "application/x-msdownload"},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	prevPayloadPut := payloadPutFunc
	payloadPutFunc = func(ctx context.Context, runtimeScope scope.Scope, req payloadPutRequest) (payloadPutReceipt, error) {
		return payloadPutReceipt{
			payloadID:      "sc:stored-mime",
			sizeBytes:      int64(len(req.body)),
			checksumSHA256: req.checksumSHA256,
			contentType:    req.contentType,
		}, nil
	}
	t.Cleanup(func() { payloadPutFunc = prevPayloadPut })

	const uploadID = "up_db_mime_reject"
	body := []byte("payload")

	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/"+uploadID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-msdownload")
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "upload-mime-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusUnsupportedMediaType, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.MIME_TYPE_NOT_ALLOWED" {
		t.Fatalf("code = %q, want document.MIME_TYPE_NOT_ALLOWED", code)
	}
	metadata := decodeGatewayErrorMetadata(t, rr)
	if metadata["uploadId"] != uploadID {
		t.Fatalf("metadata.uploadId = %q, want %q", metadata["uploadId"], uploadID)
	}
	if metadata["contentType"] != "application/x-msdownload" {
		t.Fatalf("metadata.contentType = %q, want application/x-msdownload", metadata["contentType"])
	}
}

func TestDocumentGatewayUploadHandlerRequiresAuthentication(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const uploadID = "up_db_auth_required"
	body := []byte("hello")

	req := httptest.NewRequest(http.MethodPut, "/_document/uploads/"+uploadID, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.UNAUTHENTICATED" {
		t.Fatalf("code = %q, want document.UNAUTHENTICATED", code)
	}
}

func TestDocumentGatewayDownloadRouteRejectsObjectAnchorPath(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux)

	req := httptest.NewRequest(http.MethodGet, "/_document/objects/obj_001/content", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestDocumentGatewayDownloadHandlerRejectsCompanyMismatch(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_db_company_scope"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = newGatewayDocumentStatusError(
		codes.PermissionDenied,
		"PERMISSION_DENIED",
		"binding company mismatch",
		map[string]string{
			"stage":      "resolve_download_content",
			"reason":     "binding_company_mismatch",
			"ownerModel": "auth.User",
			"fieldName":  "AttachmentField",
		},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c2",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-company-mismatch-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.PERMISSION_DENIED" {
		t.Fatalf("code = %q, want document.PERMISSION_DENIED", code)
	}
	if stage := decodeGatewayErrorStage(t, rr); stage != "resolve_download_content" {
		t.Fatalf("stage = %q, want resolve_download_content", stage)
	}
	metadata := decodeGatewayErrorMetadata(t, rr)
	if metadata["reason"] != "binding_company_mismatch" {
		t.Fatalf("metadata.reason = %q, want binding_company_mismatch", metadata["reason"])
	}
}

func TestDocumentGatewayDownloadHandlerRejectsOwnerRecordRuleFalse(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_owner_rr_false"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = newGatewayDocumentStatusError(
		codes.PermissionDenied,
		"PERMISSION_DENIED",
		"owner read authorization denied",
		map[string]string{
			"stage":      "resolve_download_content",
			"reason":     "owner_record_rule_false",
			"ownerModel": "auth.User",
			"fieldName":  "AttachmentField",
		},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-rr-false-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.PERMISSION_DENIED" {
		t.Fatalf("code = %q, want document.PERMISSION_DENIED", code)
	}
	if stage := decodeGatewayErrorStage(t, rr); stage != "resolve_download_content" {
		t.Fatalf("stage = %q, want resolve_download_content", stage)
	}
	metadata := decodeGatewayErrorMetadata(t, rr)
	if metadata["reason"] != "owner_record_rule_false" {
		t.Fatalf("metadata.reason = %q, want owner_record_rule_false", metadata["reason"])
	}
	if metadata["ownerModel"] != "auth.User" {
		t.Fatalf("metadata.ownerModel = %q, want auth.User", metadata["ownerModel"])
	}
	if metadata["fieldName"] != "AttachmentField" {
		t.Fatalf("metadata.fieldName = %q, want AttachmentField", metadata["fieldName"])
	}
}

func TestDocumentGatewayDownloadHandlerRejectsOwnerFieldReadDeny(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_owner_field_deny"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = newGatewayDocumentStatusError(
		codes.PermissionDenied,
		"PERMISSION_DENIED",
		"owner field read is denied by field rule",
		map[string]string{
			"stage":      "resolve_download_content",
			"reason":     "owner_field_read_deny",
			"ownerModel": "auth.User",
			"fieldName":  "AttachmentField",
		},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-field-deny-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.PERMISSION_DENIED" {
		t.Fatalf("code = %q, want document.PERMISSION_DENIED", code)
	}
	if stage := decodeGatewayErrorStage(t, rr); stage != "resolve_download_content" {
		t.Fatalf("stage = %q, want resolve_download_content", stage)
	}
	metadata := decodeGatewayErrorMetadata(t, rr)
	if metadata["reason"] != "owner_field_read_deny" {
		t.Fatalf("metadata.reason = %q, want owner_field_read_deny", metadata["reason"])
	}
	if metadata["ownerModel"] != "auth.User" {
		t.Fatalf("metadata.ownerModel = %q, want auth.User", metadata["ownerModel"])
	}
	if metadata["fieldName"] != "AttachmentField" {
		t.Fatalf("metadata.fieldName = %q, want AttachmentField", metadata["fieldName"])
	}
}

func TestDocumentGatewayDownloadHandlerPermissionDeniedDefaultsUnknownLabels(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_owner_missing_labels"
	docFixture := newGatewayDocumentAttachmentFixture()
	docFixture.resolveErr = newGatewayDocumentStatusError(
		codes.PermissionDenied,
		"PERMISSION_DENIED",
		"permission denied",
		map[string]string{},
	)
	dialer := newGatewayDocumentDialer(t, docFixture)

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	ctx := auth.ContextWithIdentity(req.Context(), gatewayFakeIdentity{
		userID:  "u1",
		tokenID: "tok-1",
		metadata: map[string]any{
			"activeCompanyId": "c1",
		},
	})
	ctx = auth.ContextWithAccessToken(ctx, "download-missing-labels-token")
	ctx = grpcclient.ContextWithServiceDialer(ctx, dialer)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.PERMISSION_DENIED" {
		t.Fatalf("code = %q, want document.PERMISSION_DENIED", code)
	}
	metadata := decodeGatewayErrorMetadata(t, rr)
	if metadata["stage"] != "unknown" {
		t.Fatalf("metadata.stage = %q, want unknown", metadata["stage"])
	}
	if metadata["reason"] != "unknown" {
		t.Fatalf("metadata.reason = %q, want unknown", metadata["reason"])
	}
}

func TestDocumentGatewayDownloadHandlerRequiresAuthentication(t *testing.T) {
	runtimeScope := newGatewayTestScope(t)
	mux := http.NewServeMux()
	RegisterSkeletonHandlers(mux, runtimeScope)

	const bindingID = "bnd_db_download_auth"

	req := httptest.NewRequest(http.MethodGet, "/_document/bindings/"+bindingID+"/content", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if code := decodeGatewayErrorCode(t, rr); code != "document.UNAUTHENTICATED" {
		t.Fatalf("code = %q, want document.UNAUTHENTICATED", code)
	}
}
