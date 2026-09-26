// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChatterFieldChangeEntry } from './chatterTypes';

/**
 * Formats a field-change timeline entry into a single summary line.
 */
export function formatFieldChangeSummary(
  entry: ChatterFieldChangeEntry,
  labels: {
    created: string;
    unlinked: string;
    changed: (field: string, oldValue: string, newValue: string) => string;
    action: (name: string) => string;
    fieldFallback: string;
  },
): string {
  const kind = String(entry.changeKind || '').trim();
  const normalizedKind = kind.toLowerCase();
  if (normalizedKind === 'create') return labels.created;
  if (normalizedKind === 'unlink') return labels.unlinked;
  if (normalizedKind.startsWith('action:')) {
    // Keep empty-name fallback as the raw kind (product parity: Action:action:).
    const colon = kind.indexOf(':');
    const name = colon >= 0 ? kind.slice(colon + 1).trim() : '';
    return labels.action(name || kind);
  }
  const field = entry.field || labels.fieldFallback;
  const oldValue = entry.oldValue == null || entry.oldValue === '' ? '—' : entry.oldValue;
  const newValue = entry.newValue == null || entry.newValue === '' ? '—' : entry.newValue;
  return labels.changed(field, oldValue, newValue);
}

/**
 * Formats a UTC epoch ms as `YYYY-MM-DD HH:mm` (UTC wall clock).
 */
export function formatChoyUtcIso(ms: number | null | undefined): string {
  if (ms == null || !Number.isFinite(ms)) return '';
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return '';
  const year = d.getUTCFullYear();
  const y = `${year < 0 ? '-' : ''}${String(Math.abs(year)).padStart(4, '0')}`;
  const mo = String(d.getUTCMonth() + 1).padStart(2, '0');
  const day = String(d.getUTCDate()).padStart(2, '0');
  const h = String(d.getUTCHours()).padStart(2, '0');
  const mi = String(d.getUTCMinutes()).padStart(2, '0');
  return `${y}-${mo}-${day} ${h}:${mi}`;
}

/**
 * Resolves a display label for a chatter author / actor uid.
 */
export function resolveChoyChatterAuthorLabel(
  userId: string | null | undefined,
  opts: {
    currentUserId?: string | null;
    currentUserName?: string | null;
    systemLabel?: string;
    youLabel?: string;
  } = {},
): string {
  const normalized = String(userId || '').trim();
  if (!normalized) return opts.systemLabel || 'System';
  const currentId = String(opts.currentUserId || '').trim();
  if (currentId && normalized === currentId) {
    const name = String(opts.currentUserName || '').trim();
    return name || opts.youLabel || 'You';
  }
  return normalized;
}
