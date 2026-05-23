// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Extend vue-router route metadata for auth-specific fields.
 */
declare module 'vue-router' {
  interface RouteMeta {
    /** Whether the route requires an authenticated user. */
    requiresAuth?: boolean;

    /** Resource id injected by defineRoute for permission checks. */
    resourceId?: string;

    /** Whether the route belongs to the auth entry flow. */
    isAuthPage?: boolean;
  }
}

export {};
