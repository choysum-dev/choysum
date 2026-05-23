// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultcache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/cache"
)

func TestNewInMemoryRejectsInvalidMaxEntries(t *testing.T) {
	if _, err := NewInMemory(Options{}); !errors.Is(err, ErrInvalidMaxEntries) {
		t.Fatalf("expected invalid max entries error, got %v", err)
	}
}

func TestInMemoryCacheRejectsInvalidInputs(t *testing.T) {
	c, err := NewInMemory(Options{MaxEntries: 2})
	if err != nil {
		t.Fatalf("NewInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Set(context.Background(), " ", "token", []byte("v"), time.Second); !errors.Is(err, cache.ErrInvalidNamespace) {
		t.Fatalf("expected invalid namespace error, got %v", err)
	}
	if err := c.Set(context.Background(), "auth.jwt", " ", []byte("v"), time.Second); !errors.Is(err, cache.ErrInvalidKey) {
		t.Fatalf("expected invalid key error, got %v", err)
	}
	if err := c.Set(context.Background(), "auth.jwt", "token", []byte("v"), 0); !errors.Is(err, cache.ErrInvalidTTL) {
		t.Fatalf("expected invalid ttl error, got %v", err)
	}
	if _, _, err := c.Get(context.Background(), "", "token"); !errors.Is(err, cache.ErrInvalidNamespace) {
		t.Fatalf("expected invalid namespace error from Get, got %v", err)
	}
	if err := c.Delete(context.Background(), "auth.jwt", ""); !errors.Is(err, cache.ErrInvalidKey) {
		t.Fatalf("expected invalid key error from Delete, got %v", err)
	}
}

func TestInMemoryCacheSupportsNamespaceTTLAndDelete(t *testing.T) {
	c, err := NewInMemory(Options{MaxEntries: 4})
	if err != nil {
		t.Fatalf("NewInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Set(context.Background(), "auth.jwt.access_identity", "token-1", []byte("value-1"), 200*time.Millisecond); err != nil {
		t.Fatalf("Set(access_identity) error = %v", err)
	}
	if err := c.Set(context.Background(), "auth.jwt.user_index", "token-1", []byte("value-2"), 200*time.Millisecond); err != nil {
		t.Fatalf("Set(user_index) error = %v", err)
	}

	got, found, err := c.Get(context.Background(), "auth.jwt.access_identity", "token-1")
	if err != nil {
		t.Fatalf("Get(access_identity) error = %v", err)
	}
	if !found || string(got) != "value-1" {
		t.Fatalf("unexpected access_identity lookup: found=%v value=%q", found, string(got))
	}

	got, found, err = c.Get(context.Background(), "auth.jwt.user_index", "token-1")
	if err != nil {
		t.Fatalf("Get(user_index) error = %v", err)
	}
	if !found || string(got) != "value-2" {
		t.Fatalf("unexpected user_index lookup: found=%v value=%q", found, string(got))
	}

	if err := c.Delete(context.Background(), "auth.jwt.user_index", "token-1"); err != nil {
		t.Fatalf("Delete(user_index) error = %v", err)
	}
	if _, found, err := c.Get(context.Background(), "auth.jwt.user_index", "token-1"); err != nil {
		t.Fatalf("Get(user_index after delete) error = %v", err)
	} else if found {
		t.Fatal("expected deleted user_index key to be missing")
	}

	if err := c.Set(context.Background(), "auth.jwt.access_identity", "token-expiring", []byte("value-expiring"), 20*time.Millisecond); err != nil {
		t.Fatalf("Set(expiring) error = %v", err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, found, err := c.Get(context.Background(), "auth.jwt.access_identity", "token-expiring"); err != nil {
		t.Fatalf("Get(expiring) error = %v", err)
	} else if found {
		t.Fatal("expected expired key to be evicted")
	}
}

func TestInMemoryCacheEvictsLeastRecentlyUsedEntries(t *testing.T) {
	c, err := NewInMemory(Options{MaxEntries: 1})
	if err != nil {
		t.Fatalf("NewInMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if err := c.Set(context.Background(), "auth.jwt.access_identity", "token-a", []byte("value-a"), time.Second); err != nil {
		t.Fatalf("Set(token-a) error = %v", err)
	}
	if err := c.Set(context.Background(), "auth.jwt.access_identity", "token-b", []byte("value-b"), time.Second); err != nil {
		t.Fatalf("Set(token-b) error = %v", err)
	}

	if _, found, err := c.Get(context.Background(), "auth.jwt.access_identity", "token-a"); err != nil {
		t.Fatalf("Get(token-a) error = %v", err)
	} else if found {
		t.Fatal("expected token-a to be evicted when cache is full")
	}
	if got, found, err := c.Get(context.Background(), "auth.jwt.access_identity", "token-b"); err != nil {
		t.Fatalf("Get(token-b) error = %v", err)
	} else if !found || string(got) != "value-b" {
		t.Fatalf("unexpected token-b lookup: found=%v value=%q", found, string(got))
	}
}
