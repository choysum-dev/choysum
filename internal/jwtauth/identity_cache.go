// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jwtauth

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/cache"
)

const (
	jwtAccessIdentityNamespace = "auth.jwt.access_identity.v1"
	jwtUserIndexNamespace      = "auth.jwt.user_index.v1"
)

type cachedIdentityEnvelope struct {
	UserID      string                 `json:"userId"`
	TokenID     string                 `json:"tokenId"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt   time.Time              `json:"expiresAt"`
	IssuedAt    time.Time              `json:"issuedAt"`
	TokenType   auth.TokenType         `json:"tokenType"`
	Issuer      string                 `json:"issuer"`
	CacheExpiry time.Time              `json:"cacheExpiry"`
	CacheTime   time.Time              `json:"cacheTime"`
}

type userIndexEntry struct {
	Tokens []userIndexToken `json:"tokens"`
}

type userIndexToken struct {
	CacheKey string    `json:"cacheKey"`
	TokenID  string    `json:"tokenId"`
	Expiry   time.Time `json:"expiry"`
}

type identityCache struct {
	mu    sync.Mutex
	store cache.Cache
}

func (c *identityCache) get(token string) (*cachedIdentity, error) {
	cached, _, err := c.peek(token)
	return cached, err
}

func (c *identityCache) peek(token string) (*cachedIdentity, bool, error) {
	data, found, err := c.store.Get(context.Background(), jwtAccessIdentityNamespace, token)
	if err != nil || !found {
		return nil, found, err
	}

	cached, err := decodeCachedIdentity(data)
	if err != nil {
		_ = c.store.Delete(context.Background(), jwtAccessIdentityNamespace, token)
		return nil, false, err
	}
	if time.Now().After(cached.expiry) {
		_ = c.store.Delete(context.Background(), jwtAccessIdentityNamespace, token)
		return nil, false, nil
	}

	return cached, true, nil
}

func (c *identityCache) set(token string, cached *cachedIdentity) error {
	if cached == nil || cached.identity == nil {
		return nil
	}

	payload, err := encodeCachedIdentity(cached)
	if err != nil {
		return err
	}
	entryTTL := time.Until(cached.expiry)
	if entryTTL <= 0 {
		return cache.ErrInvalidTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.store.Set(context.Background(), jwtAccessIdentityNamespace, token, payload, entryTTL); err != nil {
		return err
	}

	refs, err := c.loadUserIndexLocked(cached.identity.userID)
	if err != nil {
		_ = c.store.Delete(context.Background(), jwtAccessIdentityNamespace, token)
		return err
	}
	refs = upsertUserIndexToken(refs, userIndexToken{
		CacheKey: token,
		TokenID:  cached.identity.tokenID,
		Expiry:   cached.expiry,
	})
	if err := c.writeUserIndexLocked(cached.identity.userID, refs); err != nil {
		_ = c.store.Delete(context.Background(), jwtAccessIdentityNamespace, token)
		return err
	}
	return nil
}

func (c *identityCache) delete(token string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, found, err := c.peek(token)
	if err != nil {
		return err
	}
	if err := c.store.Delete(context.Background(), jwtAccessIdentityNamespace, token); err != nil {
		return err
	}
	if !found || cached == nil || cached.identity == nil {
		return nil
	}

	refs, err := c.loadUserIndexLocked(cached.identity.userID)
	if err != nil {
		return err
	}
	refs = removeUserIndexToken(refs, token, cached.identity.tokenID)
	return c.writeUserIndexLocked(cached.identity.userID, refs)
}

func (c *identityCache) deleteUserTokens(userID string, exceptTokenID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	refs, err := c.loadUserIndexLocked(userID)
	if err != nil {
		return err
	}
	kept := make([]userIndexToken, 0, len(refs))
	for _, ref := range refs {
		if ref.TokenID == exceptTokenID {
			kept = append(kept, ref)
			continue
		}
		if err := c.store.Delete(context.Background(), jwtAccessIdentityNamespace, ref.CacheKey); err != nil {
			return err
		}
	}
	return c.writeUserIndexLocked(userID, kept)
}

func (c *identityCache) Close() error {
	return c.store.Close()
}

func (c *identityCache) loadUserIndexLocked(userID string) ([]userIndexToken, error) {
	data, found, err := c.store.Get(context.Background(), jwtUserIndexNamespace, userID)
	if err != nil || !found {
		return nil, err
	}

	var entry userIndexEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		_ = c.store.Delete(context.Background(), jwtUserIndexNamespace, userID)
		return nil, err
	}

	now := time.Now()
	tokens := make([]userIndexToken, 0, len(entry.Tokens))
	for _, ref := range entry.Tokens {
		if ref.CacheKey == "" || ref.TokenID == "" || !ref.Expiry.After(now) {
			continue
		}
		tokens = append(tokens, ref)
	}
	return tokens, nil
}

func (c *identityCache) writeUserIndexLocked(userID string, refs []userIndexToken) error {
	now := time.Now()
	filtered := make([]userIndexToken, 0, len(refs))
	var maxExpiry time.Time
	for _, ref := range refs {
		if ref.CacheKey == "" || ref.TokenID == "" || !ref.Expiry.After(now) {
			continue
		}
		filtered = append(filtered, ref)
		if maxExpiry.IsZero() || ref.Expiry.After(maxExpiry) {
			maxExpiry = ref.Expiry
		}
	}
	if len(filtered) == 0 {
		return c.store.Delete(context.Background(), jwtUserIndexNamespace, userID)
	}

	payload, err := json.Marshal(userIndexEntry{Tokens: filtered})
	if err != nil {
		return err
	}
	return c.store.Set(context.Background(), jwtUserIndexNamespace, userID, payload, time.Until(maxExpiry))
}

func encodeCachedIdentity(cached *cachedIdentity) ([]byte, error) {
	envelope := cachedIdentityEnvelope{
		UserID:      cached.identity.userID,
		TokenID:     cached.identity.tokenID,
		Metadata:    cached.identity.metadata,
		ExpiresAt:   cached.identity.expiresAt,
		IssuedAt:    cached.identity.issuedAt,
		TokenType:   cached.identity.tokenType,
		Issuer:      cached.identity.issuer,
		CacheExpiry: cached.expiry,
		CacheTime:   cached.cacheTime,
	}
	return json.Marshal(envelope)
}

func decodeCachedIdentity(data []byte) (*cachedIdentity, error) {
	var envelope cachedIdentityEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &cachedIdentity{
		identity: NewIdentity(
			envelope.UserID,
			envelope.TokenID,
			envelope.Metadata,
			envelope.ExpiresAt,
			envelope.IssuedAt,
			envelope.TokenType,
			envelope.Issuer,
		),
		expiry:    envelope.CacheExpiry,
		cacheTime: envelope.CacheTime,
	}, nil
}

func upsertUserIndexToken(refs []userIndexToken, next userIndexToken) []userIndexToken {
	updated := false
	out := make([]userIndexToken, 0, len(refs)+1)
	for _, ref := range refs {
		if ref.CacheKey == next.CacheKey || ref.TokenID == next.TokenID {
			out = append(out, next)
			updated = true
			continue
		}
		out = append(out, ref)
	}
	if !updated {
		out = append(out, next)
	}
	return out
}

func removeUserIndexToken(refs []userIndexToken, cacheKey string, tokenID string) []userIndexToken {
	out := make([]userIndexToken, 0, len(refs))
	for _, ref := range refs {
		if ref.CacheKey == cacheKey {
			continue
		}
		if tokenID != "" && ref.TokenID == tokenID {
			continue
		}
		out = append(out, ref)
	}
	return out
}
