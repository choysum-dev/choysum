// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isClient } from '@vueuse/core';
import { AUTH_STORAGE_KEY } from './options';

/**
 * Simplified auth storage implementation.
 *
 * Chooses localStorage or sessionStorage based on rememberMe.
 */
export class AuthStorage {
  /**
   * Get data from storage.
   *
   * Reads localStorage first, then sessionStorage.
   *
   * @param key - Storage key.
   * @returns The stored value or null when missing.
   */
  getItem(key: string): string | null {
    if (!isClient) return null;

    // Check localStorage first, then fall back to sessionStorage.
    return localStorage.getItem(key) || sessionStorage.getItem(key);
  }

  /**
   * Save data to the appropriate storage backend.
   *
   * The rememberMe flag decides between localStorage and sessionStorage.
   *
   * @param key - Storage key.
   * @param value - Serialized JSON payload.
   */
  setItem(key: string, value: string): void {
    if (!isClient) return;

    try {
      // Parse the payload to decide which storage backend to use.
      const data = JSON.parse(value);
      const shouldRemember = !!data.rememberMe;

      if (shouldRemember) {
        // Persist long-lived sessions in localStorage.
        localStorage.setItem(key, value);
        sessionStorage.removeItem(key);
      } else {
        // Persist temporary sessions in sessionStorage.
        sessionStorage.setItem(key, value);
        localStorage.removeItem(key);
      }
    } catch (error) {
      // Default to sessionStorage when parsing fails.
      sessionStorage.setItem(key, value);
    }
  }

  /**
   * Remove data from every storage backend.
   *
   * @param key - Storage key.
   */
  removeItem(key: string): void {
    if (!isClient) return;

    // Remove the key from both storage locations.
    localStorage.removeItem(key);
    sessionStorage.removeItem(key);
  }

  /**
   * Clear auth-related persisted state.
   */
  clearAuthStorage(): void {
    if (!isClient) return;
    this.removeItem(AUTH_STORAGE_KEY);
  }
}

/**
 * Shared storage instance used by the Pinia persistence plugin.
 */
export const authStorage = new AuthStorage();
