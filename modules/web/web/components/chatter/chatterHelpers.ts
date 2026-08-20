// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChatterFieldChangeEntry } from '@/web/web/composables/chatter/chatterTypes';

export function formatFieldChangeSummary(
  entry: ChatterFieldChangeEntry,
  labels: {
    created: string;
    unlinked: string;
    changed: (field: string, oldValue: string, newValue: string) => string;
    action: (name: string) => string;
    fieldFallback: string;
  }
): string {
  const kind = String(entry.changeKind || '').trim();
  if (kind === 'create') return labels.created;
  if (kind === 'unlink') return labels.unlinked;
  if (kind.startsWith('action:')) {
    return labels.action(kind.slice('action:'.length) || kind);
  }
  const field = entry.field || labels.fieldFallback;
  const oldValue = entry.oldValue == null || entry.oldValue === '' ? '—' : entry.oldValue;
  const newValue = entry.newValue == null || entry.newValue === '' ? '—' : entry.newValue;
  return labels.changed(field, oldValue, newValue);
}
