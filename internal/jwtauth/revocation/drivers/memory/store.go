// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/internal/jwtauth/revocation"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/auth/autherrors"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// MemoryStore keeps revoked tokens in memory.
type MemoryStore struct {
	tokens    map[string]revocation.Token // Tokens indexed by token ID.
	userIndex map[string][]string         // User ID -> token IDs for fast per-user lookups.
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewMemoryStore creates an in-memory revocation store.
func NewMemoryStore(runtimeScope scope.Scope) (revocation.Store, error) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &MemoryStore{
		tokens:    make(map[string]revocation.Token),
		userIndex: make(map[string][]string),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start the background cleanup task.
	go store.backgroundCleanup()

	return store, nil
}

// IsRevoked reports whether the token has been revoked.
func (s *MemoryStore) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	if !revocation.IsValidTokenID(tokenID) {
		return false, autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token ID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.tokens[tokenID]
	return exists, nil
}

// RevokeToken revokes a token.
func (s *MemoryStore) RevokeToken(ctx context.Context, tokenID string, userID string, tokenType auth.TokenType, expiresAt time.Time, reason string) error {
	if !revocation.IsValidTokenID(tokenID) {
		return autherrors.NewAuthError(autherrors.ErrInvalidTokenID, "token ID cannot be empty")
	}
	if !revocation.IsValidUserID(userID) {
		return autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether the token has already been revoked.
	if _, exists := s.tokens[tokenID]; exists {
		return autherrors.NewAuthError(autherrors.ErrTokenAlreadyRevoked,
			fmt.Sprintf("token %s has already been revoked", tokenID))
	}

	// Create the revocation record.
	token := revocation.NewToken(tokenID, userID, tokenType, expiresAt, reason)

	// Save the record.
	s.tokens[tokenID] = token

	// Update the user index.
	s.userIndex[userID] = append(s.userIndex[userID], tokenID)

	return nil
}

// RevokeAllUserTokens revokes all user tokens except the specified token.
//
// userID: user ID
// exceptTokenID: token ID to exclude, if any
// reason: revocation reason recorded in storage
func (s *MemoryStore) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	if !revocation.IsValidUserID(userID) {
		return 0, autherrors.NewAuthError(autherrors.ErrInvalidUserID, "user ID cannot be empty")
	}

	// Use the default revocation reason when none is provided.
	if reason == "" {
		reason = "bulk revocation"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Read all token IDs for the user.
	tokenIDs, exists := s.userIndex[userID]
	if !exists || len(tokenIDs) == 0 {
		return 0, nil // The user has no revoked tokens.
	}

	// Split the token IDs into revoke and keep sets.
	var tokensToRevoke []string
	var tokenIDsToKeep []string

	for _, id := range tokenIDs {
		// Revoke tokens that are not excluded and are not already revoked.
		if id != exceptTokenID && !s.isRevokedNoLock(id) {
			tokensToRevoke = append(tokensToRevoke, id)
		} else {
			// Keep tokens that are already revoked or explicitly excluded.
			tokenIDsToKeep = append(tokenIDsToKeep, id)
		}
	}

	// Revoke all selected tokens.
	now := time.Now()
	expiry := now.Add(24 * 30 * time.Hour) // Default to 30 days.

	for _, tokenID := range tokensToRevoke {
		// Create the revocation record.
		token := revocation.NewToken(tokenID, userID, auth.AccessToken, expiry, reason)
		s.tokens[tokenID] = token
	}

	// Update the user index.
	if len(tokenIDsToKeep) > 0 {
		s.userIndex[userID] = tokenIDsToKeep
	} else if len(tokensToRevoke) > 0 {
		// Delete the user index entry when every token was revoked.
		delete(s.userIndex, userID)
	}

	return len(tokensToRevoke), nil
}

// isRevokedNoLock checks revocation state without taking a lock.
func (s *MemoryStore) isRevokedNoLock(tokenID string) bool {
	_, exists := s.tokens[tokenID]
	return exists
}

// CleanExpired removes expired revocation records.
func (s *MemoryStore) CleanExpired(ctx context.Context) (int, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	var expired []string

	// Find expired records.
	for tokenID, token := range s.tokens {
		if token.GetExpiresAt().Before(now) {
			expired = append(expired, tokenID)
		}
	}

	// Delete expired records.
	for _, tokenID := range expired {
		token := s.tokens[tokenID]
		delete(s.tokens, tokenID)

		// Update the user index.
		if tokenIDs, exists := s.userIndex[token.GetUserID()]; exists {
			// Find and remove the token ID from the index.
			for i, id := range tokenIDs {
				if id == tokenID {
					// Remove the element while preserving order.
					s.userIndex[token.GetUserID()] = append(tokenIDs[:i], tokenIDs[i+1:]...)
					break
				}
			}
			// Delete the full user entry when the list becomes empty.
			if len(s.userIndex[token.GetUserID()]) == 0 {
				delete(s.userIndex, token.GetUserID())
			}
		}
	}

	return len(expired), nil
}

// Close closes the store and releases its resources.
func (s *MemoryStore) Close() error {
	s.cancel() // Stop the background cleanup task.
	return nil
}

// backgroundCleanup periodically cleans expired records.
func (s *MemoryStore) backgroundCleanup() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := s.CleanExpired(s.ctx); err != nil {
				// Logging could be added here, but cleanup should keep running.
				// log.Printf("failed to clean expired tokens from memory store: %v", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}
