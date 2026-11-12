// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Sidebar identity label: Language.PersonNameFormat over First/Last, then FullName, then login ids.
 */

import { formatPersonName } from '@/core/service/utils/person_name';

function trimText(value: unknown): string {
  return String(value ?? '').trim();
}

/** Build the user-menu primary label from profile fields + optional person-name template. */
export function resolveUserMenuDisplayName(input: {
  firstName?: unknown;
  lastName?: unknown;
  fullName?: unknown;
  displayName?: unknown;
  username?: unknown;
  email?: unknown;
  personNameFormat?: string | null;
}): string {
  const first = trimText(input.firstName);
  const last = trimText(input.lastName);
  if (first || last) {
    const formatted = formatPersonName(
      { FirstName: first, LastName: last },
      input.personNameFormat,
    );
    if (formatted) return formatted;
  }
  const full = trimText(input.fullName);
  if (full) return full;
  const display = trimText(input.displayName);
  if (display) return display;
  const username = trimText(input.username);
  if (username) return username;
  return trimText(input.email);
}

/**
 * Secondary identity line for sidebar / preferences header: username and email
 * on one line (`username · email`). Omits empty parts, identical duplicates, and
 * segments that equal the primary display label.
 */
export function resolveUserMenuMeta(input: {
  email?: unknown;
  username?: unknown;
  /** When set, drop segments that equal this primary label (case-sensitive trim). */
  primaryLabel?: unknown;
}): string {
  const primary = trimText(input.primaryLabel);
  const parts: string[] = [];
  for (const raw of [input.username, input.email]) {
    const part = trimText(raw);
    if (!part) continue;
    if (primary && part === primary) continue;
    if (parts.includes(part)) continue;
    parts.push(part);
  }
  return parts.join(' · ');
}
