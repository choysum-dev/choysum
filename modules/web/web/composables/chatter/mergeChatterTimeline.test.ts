// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { mergeChatterTimeline } from './mergeChatterTimeline';

describe('mergeChatterTimeline', () => {
  it('merges messages and field changes by timestamp ascending', () => {
    const entries = mergeChatterTimeline(
      [
        { Id: 'm2', Type: 'comment', Body: 'second', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
        { Id: 'm1', Type: 'comment', Body: 'first', AuthorUid: 'u1', CreatedAt: '2024-01-01T00:00:00.000Z' },
      ],
      [{ Id: 'f1', Field: 'Name', Kind: 'field', OldValue: 'A', NewValue: 'B', ActorUid: 'u2', At: '2024-01-01T12:00:00.000Z' }]
    );

    expect(entries.map(entry => `${entry.kind}:${entry.id}`)).toEqual(['message:m1', 'fieldChange:f1', 'message:m2']);
  });

  it('skips rows without ids or timestamps', () => {
    const entries = mergeChatterTimeline(
      [{ Id: '', Body: 'x', CreatedAt: '2024-01-01T00:00:00.000Z' }, { Id: 'm1', Body: 'ok', CreatedAt: '' }],
      [{ Id: 'f1', Kind: 'create', At: null }]
    );
    expect(entries).toEqual([]);
  });
});
