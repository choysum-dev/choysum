// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope, ref } from 'vue';
import { useChatterTimeline } from './useChatterTimeline';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function flush(): Promise<void> {
  return new Promise(resolve => {
    setTimeout(resolve, 0);
  });
}

function makeSearchQueue() {
  const queue: Array<() => Promise<unknown[]>> = [];
  const search = fnRecorder((_model: string, _resId: string, _fields: readonly string[]) => {
    const next = queue.shift();
    if (!next) return Promise.resolve([]);
    return next();
  });
  return {
    search,
    enqueue(factory: () => Promise<unknown[]>) {
      queue.push(factory);
    },
    enqueueResolved(rows: unknown[]) {
      queue.push(() => Promise.resolve(rows));
    },
    enqueueRejected(err: unknown) {
      queue.push(async () => {
        throw err instanceof Error ? err : new Error(String(err));
      });
    },
  };
}

describe('useChatterTimeline', () => {
  test('uses default store getters when deps are omitted', () => {
    // FE unit host stubs the store registry factory, so the default path resolves instead of throwing.
    const timeline = useChatterTimeline(ref('partner.Partner'), ref('r1'));
    expect(timeline).toBeTruthy();
    expect(typeof timeline.refresh).toBe('function');
  });

  test('ignores stale refresh results after the record changes', async () => {
    let resolveFirst: ((rows: unknown[]) => void) | undefined;
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    messages.enqueue(
      () =>
        new Promise(resolve => {
          resolveFirst = resolve;
        })
    );
    messages.enqueueResolved([
      { Id: 'm2', Type: 'comment', Body: 'new', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
    ]);
    fields.enqueueResolved([]);
    fields.enqueueResolved([]);

    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
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

  test('clears loading when the thread identity is empty', async () => {
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    const model = ref('');
    const resId = ref<string | undefined>('');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value).toEqual([]);
    expect(timeline.loading.value).toBe(false);
    scope.stop();
  });

  test('loads merged timeline entries on success', async () => {
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    messages.enqueueResolved([
      { Id: 'm1', Type: 'comment', Body: 'hello', AuthorUid: 'u1', CreatedAt: '2024-01-01T00:00:00.000Z' },
    ]);
    fields.enqueueResolved([
      { Id: 'f1', Field: 'Name', Kind: 'field', OldValue: 'A', NewValue: 'B', ActorUid: 'u2', At: '2024-01-02T00:00:00.000Z' },
    ]);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value.map(entry => `${entry.kind}:${entry.id}`)).toEqual(['message:m1', 'fieldChange:f1']);
    expect(timeline.error.value).toBeNull();
    scope.stop();
  });

  test('stores refresh failures', async () => {
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    messages.enqueueRejected(new Error('timeline failed'));
    fields.enqueueResolved([]);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.entries.value).toEqual([]);
    expect(timeline.error.value).toBe('timeline failed');
    expect(timeline.loading.value).toBe(false);
    scope.stop();
  });

  test('maps non-Error refresh failures to strings', async () => {
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    messages.enqueueRejected('boom');
    fields.enqueueResolved([]);
    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
    if (!timeline) throw new Error('timeline missing');
    await flush();
    expect(timeline.error.value).toBe('boom');
    scope.stop();
  });

  test('ignores stale refresh failures after the record changes', async () => {
    let rejectFirst: ((err: unknown) => void) | undefined;
    const messages = makeSearchQueue();
    const fields = makeSearchQueue();
    messages.enqueue(
      () =>
        new Promise((_resolve, reject) => {
          rejectFirst = reject;
        })
    );
    messages.enqueueResolved([
      { Id: 'm2', Type: 'comment', Body: 'new', AuthorUid: 'u1', CreatedAt: '2024-01-02T00:00:00.000Z' },
    ]);
    fields.enqueueResolved([]);
    fields.enqueueResolved([]);

    const model = ref('partner.Partner');
    const resId = ref<string | undefined>('r1');
    const scope = effectScope();
    const timeline = scope.run(() =>
      useChatterTimeline(model, resId, {
        getMessageStore: () => ({ SearchByRecord: messages.search } as any),
        getFieldChangeStore: () => ({ SearchByRecord: fields.search } as any),
      })
    );
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
