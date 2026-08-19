// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type recordingBus struct {
	mu        sync.Mutex
	published []bus.Event
	err       error
}

func (b *recordingBus) Publish(_ context.Context, event bus.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	b.published = append(b.published, event)
	return nil
}

func (b *recordingBus) Subscribe(string, bus.EventHandler) (bus.Subscription, error) {
	return noopSub{}, nil
}

func (b *recordingBus) snapshot() []bus.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]bus.Event, len(b.published))
	copy(out, b.published)
	return out
}

type noopSub struct{}

func (noopSub) Close() error { return nil }

type busBridgeTestScope struct {
	cfg    *config.Config
	logger *slog.Logger
}

func (e *busBridgeTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *busBridgeTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *busBridgeTestScope) Session() *scope.Session { return nil }
func (e *busBridgeTestScope) WithContext(ctx context.Context) scope.Scope {
	return &busBridgeTestScope{cfg: e.cfg, logger: e.logger}
}
func (e *busBridgeTestScope) Context() context.Context { return context.Background() }
func (e *busBridgeTestScope) Logger() *slog.Logger     { return e.logger }
func (e *busBridgeTestScope) Config() *config.Config   { return e.cfg }
func (e *busBridgeTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func newBusBridgeTestScope() *busBridgeTestScope {
	return &busBridgeTestScope{
		cfg:    &config.Config{},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestWithBusPublishUsesHostSingleton(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)

	events := &recordingBus{}
	bus.SetHost(events)

	engine := newTestQuickjsEngine(t, WithBus(newBusBridgeTestScope()))
	if got := evalString(t, engine, `$choysum.bus.topicMessageThreadChanged`); got != bus.TopicMessageThreadChanged {
		t.Fatalf("topicMessageThreadChanged = %q, want %q", got, bus.TopicMessageThreadChanged)
	}

	result := engine.Ctx.Eval(`$choysum.bus.publish({
		topic: $choysum.bus.topicMessageThreadChanged,
		source: 'message.Post',
		at: 1700000000000,
		payload: { model: 'partner.Partner', resId: 'r1', messageId: 'm1' },
	})`)
	defer result.Free()
	if result.IsException() {
		t.Fatalf("publish threw: %v", engine.Ctx.Exception())
	}

	got := events.snapshot()
	if len(got) != 1 {
		t.Fatalf("published = %#v, want 1 event", got)
	}
	if got[0].Topic != bus.TopicMessageThreadChanged || got[0].Source != "message.Post" {
		t.Fatalf("event = %#v", got[0])
	}
	if got[0].At.UTC().UnixMilli() != 1_700_000_000_000 {
		t.Fatalf("at = %v", got[0].At)
	}
	if got[0].Payload["model"] != "partner.Partner" || got[0].Payload["resId"] != "r1" || got[0].Payload["messageId"] != "m1" {
		t.Fatalf("payload = %#v", got[0].Payload)
	}
}

func TestWithBusPublishRequiresTopic(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)
	bus.SetHost(&recordingBus{})

	engine := newTestQuickjsEngine(t, WithBus(newBusBridgeTestScope()))
	result := engine.Ctx.Eval(`(() => { try { $choysum.bus.publish({ source: 'x' }); return ''; } catch (e) { return String(e); } })()`)
	defer result.Free()
	if result.IsException() {
		t.Fatalf("eval threw: %v", engine.Ctx.Exception())
	}
	if got := result.String(); !strings.Contains(got, "topic is required") {
		t.Fatalf("error = %q, want topic is required", got)
	}
}

func TestWithBusPublishPropagatesBusError(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)
	bus.SetHost(&recordingBus{err: errors.New("bus down")})

	engine := newTestQuickjsEngine(t, WithBus(newBusBridgeTestScope()))
	result := engine.Ctx.Eval(`(() => { try { $choysum.bus.publish({ topic: 't' }); return ''; } catch (e) { return String(e); } })()`)
	defer result.Free()
	if result.IsException() {
		t.Fatalf("eval threw: %v", engine.Ctx.Exception())
	}
	if got := result.String(); !strings.Contains(got, "bus down") {
		t.Fatalf("error = %q, want bus down", got)
	}
}

func TestWithBusPublishDefaultsAtWhenOmitted(t *testing.T) {
	bus.ClearHostForTest()
	t.Cleanup(bus.ClearHostForTest)
	events := &recordingBus{}
	bus.SetHost(events)

	engine := newTestQuickjsEngine(t, WithBus(newBusBridgeTestScope()))
	before := time.Now().UTC().Add(-time.Second)
	result := engine.Ctx.Eval(`$choysum.bus.publish({ topic: 't', source: 's' })`)
	defer result.Free()
	if result.IsException() {
		t.Fatalf("publish threw: %v", engine.Ctx.Exception())
	}
	after := time.Now().UTC().Add(time.Second)
	got := events.snapshot()
	if len(got) != 1 {
		t.Fatalf("published = %#v", got)
	}
	if got[0].At.Before(before) || got[0].At.After(after) {
		t.Fatalf("at = %v, want near now", got[0].At)
	}
}
