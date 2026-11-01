// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth AppSetting key: gate D20 silent browser-timezone persist
 * (table-local; no `auth.` prefix).
 *
 * Kept in a dependency-free module so web pages can import the key without
 * pulling ORM / dialect into the browser bundle.
 */
export const PERSIST_BROWSER_TIMEZONE_KEY = 'persist_browser_timezone';
