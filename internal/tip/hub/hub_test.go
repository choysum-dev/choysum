// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/bus/inprocess"
	"github.com/choysum-dev/choysum/internal/tip/proto/tippb"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/bus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type hubTestIdentity struct {
	userID string
	valid  bool
}

func (i hubTestIdentity) GetUserID() string                   { return i.userID }
func (i hubTestIdentity) GetTokenID() string                  { return "" }
func (i hubTestIdentity) GetMetadata() map[string]interface{} { return nil }
func (i hubTestIdentity) IsValid() bool                       { return i.valid }

type testStream struct {
	grpc.ServerStreamingServer[tippb.Tip]
	ctx       context.Context
	mu        sync.Mutex
	tips      []*tippb.Tip
	sendErr   error
	sendDelay time.Duration
	sent      chan struct{}
}

func (s *testStream) Context() context.Context { return s.ctx }

func (s *testStream) Send(tip *tippb.Tip) error {
	if s.sendDelay > 0 {
		time.Sleep(s.sendDelay)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sendErr != nil {
		return s.sendErr
	}
	s.tips = append(s.tips, proto.Clone(tip).(*tippb.Tip))
	if s.sent != nil {
		select {
		case s.sent <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *testStream) snapshot() []*tippb.Tip {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*tippb.Tip, len(s.tips))
	copy(out, s.tips)
	return out
}

type subscribedBus struct {
	inner      bus.EventBus
	subscribed chan struct{}
}

func (b *subscribedBus) Publish(ctx context.Context, event bus.Event) error {
	return b.inner.Publish(ctx, event)
}

func (b *subscribedBus) Subscribe(topic string, handler bus.EventHandler) (bus.Subscription, error) {
	sub, err := b.inner.Subscribe(topic, handler)
	if err == nil && b.subscribed != nil {
		select {
		case b.subscribed <- struct{}{}:
		default:
		}
	}
	return sub, err
}

type failSubscribeBus struct{}

func (failSubscribeBus) Publish(context.Context, bus.Event) error { return nil }
func (failSubscribeBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return nil, errors.New("subscribe down")
}

func identityCtx(userID string) context.Context {
	return auth.ContextWithIdentity(context.Background(), hubTestIdentity{userID: userID, valid: true})
}

func waitSubscribed(t *testing.T, subscribed <-chan struct{}) {
	t.Helper()
	select {
	case <-subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bus subscribe")
	}
}

func TestSubscribeThreadFiltersAndCancels(t *testing.T) {
	inner := inprocess.NewInProcessBus()
	subscribed := make(chan struct{}, 1)
	events := &subscribedBus{inner: inner, subscribed: subscribed}
	h := New(events)

	ctx, cancel := context.WithCancel(identityCtx("user-1"))
	defer cancel()
	stream := &testStream{ctx: ctx, sent: make(chan struct{}, 8)}
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "42"}, stream)
	}()
	waitSubscribed(t, subscribed)

	if err := inner.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicMessageThreadChanged,
		Source: "mismatch",
		Payload: map[string]any{
			"model": "message.thread",
			"resId": "other",
		},
	}); err != nil {
		t.Fatalf("Publish mismatch: %v", err)
	}
	if err := inner.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicMessageThreadChanged,
		Source: "match",
		At:     time.UnixMilli(1_700_000_000_000).UTC(),
		Payload: map[string]any{
			"model":     "message.thread",
			"resId":     "42",
			"messageId": "m-1",
		},
	}); err != nil {
		t.Fatalf("Publish match: %v", err)
	}

	select {
	case <-stream.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for matching tip")
	}

	got := stream.snapshot()
	if len(got) != 1 {
		t.Fatalf("tips = %#v, want 1 matching tip", got)
	}
	if got[0].GetTopic() != bus.TopicMessageThreadChanged || got[0].GetSource() != "match" {
		t.Fatalf("unexpected tip metadata: %#v", got[0])
	}
	if got[0].GetModel() != "message.thread" || got[0].GetResId() != "42" || got[0].GetMessageId() != "m-1" {
		t.Fatalf("unexpected tip locators: %#v", got[0])
	}
	if got[0].GetAtUnixMs() != 1_700_000_000_000 {
		t.Fatalf("at_unix_ms = %d", got[0].GetAtUnixMs())
	}

	cancel()
	select {
	case err := <-errCh:
		if status.Code(err) != codes.Canceled {
			t.Fatalf("SubscribeThread after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for canceled subscribe")
	}

	if err := inner.Publish(context.Background(), bus.Event{
		Topic: bus.TopicMessageThreadChanged,
		Payload: map[string]any{
			"model": "message.thread",
			"resId": "42",
		},
	}); err != nil {
		t.Fatalf("Publish after cancel: %v", err)
	}
	if got := stream.snapshot(); len(got) != 1 {
		t.Fatalf("tips after cancel = %#v, want no further Send", got)
	}
}

func TestSubscribeNotificationsFiltersByIdentityUser(t *testing.T) {
	inner := inprocess.NewInProcessBus()
	subscribed := make(chan struct{}, 1)
	events := &subscribedBus{inner: inner, subscribed: subscribed}
	h := New(events)

	ctx, cancel := context.WithCancel(identityCtx("user-1"))
	defer cancel()
	stream := &testStream{ctx: ctx, sent: make(chan struct{}, 8)}
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.SubscribeNotifications(&tippb.SubscribeNotificationsReq{}, stream)
	}()
	waitSubscribed(t, subscribed)

	if err := inner.Publish(context.Background(), bus.Event{
		Topic:   bus.TopicMessageNotificationUser,
		Payload: map[string]any{"userId": "other"},
	}); err != nil {
		t.Fatalf("Publish other user: %v", err)
	}
	if err := inner.Publish(context.Background(), bus.Event{
		Topic:   bus.TopicMessageNotificationUser,
		Payload: map[string]any{},
	}); err != nil {
		t.Fatalf("Publish missing userId: %v", err)
	}
	if err := inner.Publish(context.Background(), bus.Event{
		Topic:   bus.TopicMessageNotificationUser,
		Source:  "inbox",
		Payload: map[string]any{"userId": "user-1"},
	}); err != nil {
		t.Fatalf("Publish matching user: %v", err)
	}

	select {
	case <-stream.sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for matching notification tip")
	}
	got := stream.snapshot()
	if len(got) != 1 || got[0].GetUserId() != "user-1" || got[0].GetSource() != "inbox" {
		t.Fatalf("tips = %#v, want one tip for user-1", got)
	}
	cancel()
	<-errCh
}

func TestSubscribeThreadUnauthenticated(t *testing.T) {
	h := New(inprocess.NewInProcessBus())
	err := h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{ctx: context.Background()})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated code = %v err=%v", status.Code(err), err)
	}

	err = h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{
		ctx: auth.ContextWithIdentity(context.Background(), hubTestIdentity{userID: "u", valid: false}),
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("invalid identity code = %v err=%v", status.Code(err), err)
	}
}

func TestSubscribeThreadInvalidArgument(t *testing.T) {
	h := New(inprocess.NewInProcessBus())
	err := h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread"}, &testStream{ctx: identityCtx("u1")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing res_id code = %v err=%v", status.Code(err), err)
	}
}

func TestSubscribeThreadNilBusUnavailable(t *testing.T) {
	h := New(nil)
	err := h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{ctx: identityCtx("u1")})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("nil bus code = %v err=%v", status.Code(err), err)
	}
}

func TestSubscribeThreadSubscribeFailureUnavailable(t *testing.T) {
	h := New(failSubscribeBus{})
	err := h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{ctx: identityCtx("u1")})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("subscribe failure code = %v err=%v", status.Code(err), err)
	}
}

func TestSubscribeThreadPerUserCap(t *testing.T) {
	inner := inprocess.NewInProcessBus()
	subscribed := make(chan struct{}, 2)
	events := &subscribedBus{inner: inner, subscribed: subscribed}
	h := New(events, WithMaxStreamsPerUser(1))

	ctx, cancel := context.WithCancel(identityCtx("user-1"))
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{ctx: ctx})
	}()
	waitSubscribed(t, subscribed)

	err := h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "2"}, &testStream{ctx: identityCtx("user-1")})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("cap code = %v err=%v", status.Code(err), err)
	}

	otherErrCh := make(chan error, 1)
	otherCtx, otherCancel := context.WithCancel(identityCtx("user-2"))
	defer otherCancel()
	go func() {
		otherErrCh <- h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, &testStream{ctx: otherCtx})
	}()
	waitSubscribed(t, subscribed)
	otherCancel()
	if err := <-otherErrCh; status.Code(err) != codes.Canceled {
		t.Fatalf("other user subscribe: %v", err)
	}

	cancel()
	if err := <-errCh; status.Code(err) != codes.Canceled {
		t.Fatalf("first subscribe: %v", err)
	}
}

func TestPublishDoesNotBlockWhenSendIsSlow(t *testing.T) {
	inner := inprocess.NewInProcessBus()
	subscribed := make(chan struct{}, 1)
	events := &subscribedBus{inner: inner, subscribed: subscribed}
	h := New(events)

	ctx, cancel := context.WithCancel(identityCtx("user-1"))
	defer cancel()
	stream := &testStream{ctx: ctx, sendDelay: 50 * time.Millisecond}
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.SubscribeThread(&tippb.SubscribeThreadReq{Model: "message.thread", ResId: "1"}, stream)
	}()
	waitSubscribed(t, subscribed)

	start := time.Now()
	for i := 0; i < 32; i++ {
		if err := inner.Publish(context.Background(), bus.Event{
			Topic: bus.TopicMessageThreadChanged,
			Payload: map[string]any{
				"model": "message.thread",
				"resId": "1",
			},
		}); err != nil {
			t.Fatalf("Publish %d: %v", i, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Publish blocked for %s; bus handlers must not Send", elapsed)
	}
	cancel()
	<-errCh
}

func TestSubscribeThreadBufconn(t *testing.T) {
	inner := inprocess.NewInProcessBus()
	subscribed := make(chan struct{}, 1)
	events := &subscribedBus{inner: inner, subscribed: subscribed}
	h := New(events)
	lis := bufconn.Listen(1024 * 1024)
	identity := hubTestIdentity{userID: "user-1", valid: true}
	server := grpc.NewServer(grpc.StreamInterceptor(func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &ctxServerStream{ServerStream: ss, ctx: auth.ContextWithIdentity(ss.Context(), identity)})
	}))
	tippb.RegisterTipHubServer(server, h)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := tippb.NewTipHubClient(conn).SubscribeThread(ctx, &tippb.SubscribeThreadReq{
		Model: "message.thread",
		ResId: "42",
	})
	if err != nil {
		t.Fatalf("SubscribeThread: %v", err)
	}
	waitSubscribed(t, subscribed)

	if err := inner.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicMessageThreadChanged,
		Source: "bufconn",
		Payload: map[string]any{
			"model": "message.thread",
			"resId": "42",
		},
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	tip, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if tip.GetSource() != "bufconn" || tip.GetResId() != "42" {
		t.Fatalf("unexpected tip: %#v", tip)
	}
}

type ctxServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *ctxServerStream) Context() context.Context { return s.ctx }
