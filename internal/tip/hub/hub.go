// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/tip/proto/tippb"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/bus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultMaxStreamsPerUser = 8
	tipBufferSize            = 16

	payloadKeyModel     = "model"
	payloadKeyResID     = "resId"
	payloadKeyMessageID = "messageId"
	payloadKeyUserID    = "userId"
)

// Hub serves TipHub server-streaming RPCs by subscribing to the host EventBus.
type Hub struct {
	tippb.UnimplementedTipHubServer

	events     bus.EventBus
	maxPerUser int

	mu            sync.Mutex
	streamsByUser map[string]int
}

// Option configures Hub construction.
type Option func(*Hub)

// WithMaxStreamsPerUser caps concurrent tip streams per authenticated user.
func WithMaxStreamsPerUser(n int) Option {
	return func(h *Hub) {
		h.maxPerUser = n
	}
}

// New builds a TipHub backed by the host event bus singleton.
func New(events bus.EventBus, opts ...Option) *Hub {
	h := &Hub{
		events:        events,
		maxPerUser:    defaultMaxStreamsPerUser,
		streamsByUser: map[string]int{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	if h.maxPerUser <= 0 {
		h.maxPerUser = defaultMaxStreamsPerUser
	}
	return h
}

// SubscribeThread streams tips for one (model, res_id) thread.
func (h *Hub) SubscribeThread(req *tippb.SubscribeThreadReq, stream grpc.ServerStreamingServer[tippb.Tip]) error {
	identity, err := requireIdentity(stream.Context())
	if err != nil {
		return err
	}
	model := strings.TrimSpace(req.GetModel())
	resID := strings.TrimSpace(req.GetResId())
	if model == "" || resID == "" {
		return status.Error(codes.InvalidArgument, "model and res_id are required")
	}
	return h.serve(stream, identity, bus.TopicMessageThreadChanged, func(event bus.Event) bool {
		return payloadString(event, payloadKeyModel) == model && payloadString(event, payloadKeyResID) == resID
	})
}

// SubscribeNotifications streams inbox tips for the authenticated user.
func (h *Hub) SubscribeNotifications(_ *tippb.SubscribeNotificationsReq, stream grpc.ServerStreamingServer[tippb.Tip]) error {
	identity, err := requireIdentity(stream.Context())
	if err != nil {
		return err
	}
	userID := strings.TrimSpace(identity.GetUserID())
	return h.serve(stream, identity, bus.TopicMessageNotificationUser, func(event bus.Event) bool {
		payloadUser := payloadString(event, payloadKeyUserID)
		return payloadUser != "" && payloadUser == userID
	})
}

func (h *Hub) serve(stream grpc.ServerStreamingServer[tippb.Tip], identity auth.Identity, topic string, match func(bus.Event) bool) error {
	if h == nil || h.events == nil {
		return status.Error(codes.Unavailable, "event bus unavailable")
	}
	userID := strings.TrimSpace(identity.GetUserID())
	if !h.acquireStream(userID) {
		return status.Error(codes.ResourceExhausted, "too many tip streams")
	}
	defer h.releaseStream(userID)

	tips := make(chan *tippb.Tip, tipBufferSize)
	sub, err := h.events.Subscribe(topic, func(_ context.Context, event bus.Event) {
		if match != nil && !match(event) {
			return
		}
		select {
		case tips <- eventToTip(event):
		default:
		}
	})
	if err != nil {
		return status.Errorf(codes.Unavailable, "subscribe failed: %v", err)
	}
	defer func() { _ = sub.Close() }()

	for {
		select {
		case <-stream.Context().Done():
			return status.FromContextError(stream.Context().Err()).Err()
		case tip := <-tips:
			if err := stream.Send(tip); err != nil {
				return err
			}
		}
	}
}

func requireIdentity(ctx context.Context) (auth.Identity, error) {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil || !identity.IsValid() || strings.TrimSpace(identity.GetUserID()) == "" {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}
	return identity, nil
}

func (h *Hub) acquireStream(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.streamsByUser[userID] >= h.maxPerUser {
		return false
	}
	h.streamsByUser[userID]++
	return true
}

func (h *Hub) releaseStream(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := h.streamsByUser[userID] - 1
	if n <= 0 {
		delete(h.streamsByUser, userID)
		return
	}
	h.streamsByUser[userID] = n
}

func eventToTip(event bus.Event) *tippb.Tip {
	tip := &tippb.Tip{
		Topic:     event.Topic,
		Source:    event.Source,
		Model:     payloadString(event, payloadKeyModel),
		ResId:     payloadString(event, payloadKeyResID),
		MessageId: payloadString(event, payloadKeyMessageID),
		UserId:    payloadString(event, payloadKeyUserID),
	}
	if !event.At.IsZero() {
		tip.AtUnixMs = event.At.UTC().UnixMilli()
	}
	return tip
}

func payloadString(event bus.Event, key string) string {
	if event.Payload == nil {
		return ""
	}
	value, ok := event.Payload[key]
	if !ok || value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return s
}
