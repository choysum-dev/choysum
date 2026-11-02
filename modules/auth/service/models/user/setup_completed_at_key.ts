// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth AppSetting key: first-run setup completion timestamp (RFC3339).
 * Absence / empty ⇒ SetupPending; set ⇒ SetupComplete (Ready when provisioned).
 *
 * Kept in a dependency-free module so host Go can mirror the same string and
 * web code can import without pulling ORM into the browser bundle.
 */
export const SETUP_COMPLETED_AT_KEY = 'setup.completed_at';
