// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const messageSearch = vi.fn();
const fieldSearch = vi.fn();

vi.mock('./chatterStores', () => ({
  getMessageStore: () => ({ SearchByRecord: (...args: unknown[]) => messageSearch(...args) }),
  getFieldChangeStore: () => ({ SearchByRecord: (...args: unknown[]) => fieldSearch(...args) }),
}));

import { useChatterTimeline } from './useChatterTimeline';

function flush(): Promise<void> {
  return new Promise(resolve => {
    setTimeout(resolve, 0);
  });
}

describe('useChatterTimeline', () => {
  beforeEach(() => {
    messageSearch.mockReset();
    fieldSearch.mockReset();
    fieldSearch.mockResolvedValue([]);
  });

  it('ignores stale refresh results after the record changes', async () => {
    let resolveFirst: ((rows: unknown[]) => void) | undefined;
    messageSearch
      .mockImplementationOnce(
        () =>
          new Promise(resolve => {
            resolveFirst = resolve;
          })
      )
      .mockResolvedValueOnce([
        { Id: 'm2', Type: 'comment', Body: 'new', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
      ]);

    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');

    await flush();
    resId.value = 'r2';
    await flush();

    expect(timeline.entries.value.map(entry => entry.id)).toEqual(['m2']);
    resolveFirst?.([
      { Id: 'm1', Type: 'comment', Body: 'old', AuthorUid: 'u1', CreatedAt: '2024-01-01T00:00:00.000Z' },
    ]);
    await flush();
    expect(timeline.entries.value.map(entry => entry.id)).toEqual(['m2']);
    expect(timeline.loading.value).toBe(false);
    scope.stop();
  });

  it('clears loading when the thread identity is empty', async () => {
    messageSearch.mockResolvedValue([]);
    const model = ref('');
    const resId = ref<string | undefined>('');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value).toEqual([]);
    expect(timeline.loading.value).toBe(false);
    scope.stop();
  });

  it('loads merged timeline entries on success', async () => {
    messageSearch.mockResolvedValue([
      { Id: 'm1', Type: 'comment', Body: 'hello', AuthorUid: 'u1', CreatedAt: '2024-01-01T00:00:00.000Z' },
    ]);
    fieldSearch.mockResolvedValue([
      { Id: 'f1', Field: 'Name', Kind: 'field', OldValue: 'A', NewValue: 'B', ActorUid: 'u2', At: '2024-01-02T00:00:00.000Z' },
    ]);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value.map(entry => `${entry.kind}:${entry.id}`)).toEqual(['message:m1', 'fieldChange:f1']);
    expect(timeline.error.value).toBeNull();
    scope.stop();
  });

  it('stores refresh failures', async () => {
    messageSearch.mockRejectedValue(new Error('timeline failed'));
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value).toEqual([]);
    expect(timeline.error.value).toBe('timeline failed');
    expect(timeline.loading.value).toBe(false);
    scope.stop();
  });

  it('maps non-Error refresh failures to strings', async () => {
    messageSearch.mockRejectedValue('boom');
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.error.value).toBe('boom');
    scope.stop();
  });

  it('ignores stale refresh failures after the record changes', async () => {
    let rejectFirst: ((err: unknown) => void) | undefined;
    messageSearch
      .mockImplementationOnce(
        () =>
          new Promise((_resolve, reject) => {
            rejectFirst = reject;
          })
      )
      .mockResolvedValueOnce([
        { Id: 'm2', Type: 'comment', Body: 'new', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
      ]);

    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() => useChatterTimeline(model, resId));
    if (!timeline) throw new Error('timeline missing');

    await flush();
    resId.value = 'r2';
    await flush();
    expect(timeline.entries.value.map(entry => entry.id)).toEqual(['m2']);
    rejectFirst?.(new Error('stale'));
    await flush();
    expect(timeline.error.value).toBeNull();
    expect(timeline.entries.value.map(entry => entry.id)).toEqual(['m2']);
    scope.stop();
  });
});
