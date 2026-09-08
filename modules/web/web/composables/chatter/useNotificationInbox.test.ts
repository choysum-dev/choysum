// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, effectScope, h, onMounted } from 'vue';

import { fnRecorder, flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useNotificationInbox } from './useNotificationInbox';

const POLL_MS = 25;

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

describe('useNotificationInbox', () => {
  const SearchInbox = fnRecorder(async () => [] as any[]);
  const MarkRead = fnRecorder(async () => 1);
  const MarkAllRead = fnRecorder(async () => 1);
  const onTips = fnRecorder(async () => undefined);
  const subscribeNotifications = fnRecorder(() => ({} as any));

  function tipDeps() {
    return {
      getNotificationStore: () => ({ SearchInbox, MarkRead, MarkAllRead }) as any,
      onTips: onTips as any,
      subscribeNotifications: subscribeNotifications as any,
      pollFallbackMs: POLL_MS,
    };
  }

  beforeEach(() => {
    SearchInbox.mockReset();
    SearchInbox.mockImplementation(async () => []);
    MarkRead.mockReset();
    MarkRead.mockImplementation(async () => 1);
    MarkAllRead.mockReset();
    MarkAllRead.mockImplementation(async () => 1);
    onTips.mockReset();
    onTips.mockImplementation(async () => undefined);
    subscribeNotifications.mockReset();
    subscribeNotifications.mockImplementation(() => ({} as any));
  });

  test('starts poll fallback after the notification tip stream ends', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    await flushPromises();
    SearchInbox.mockClear();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('stores mark-read failures without throwing', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    MarkRead.mockImplementation(async () => {
      throw new Error('mark failed');
    });
    await inbox.markRead('n1');
    expect(inbox.error.value).toBe('mark failed');
    MarkAllRead.mockImplementation(async () => {
      throw new Error('mark all failed');
    });
    await inbox.markAllRead();
    expect(inbox.error.value).toBe('mark all failed');
    scope.stop();
  });

  test('ignores stale SearchInbox results after a later refresh', async () => {
    let resolveFirst: ((rows: unknown[]) => void) | undefined;
    let call = 0;
    SearchInbox.mockImplementation(() => {
      call += 1;
      if (call === 1) {
        return new Promise(resolve => {
          resolveFirst = resolve;
        });
      }
      return Promise.resolve([{ Id: 'n2', IsRead: true }]);
    });

    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    const first = inbox.refresh();
    await Promise.resolve();
    await inbox.refresh();
    expect(inbox.rows.value.map(row => row.Id)).toEqual(['n2']);
    resolveFirst?.([{ Id: 'n1', IsRead: false }]);
    await first;
    expect(inbox.rows.value.map(row => row.Id)).toEqual(['n2']);
    expect(inbox.loading.value).toBe(false);
    scope.stop();
  });

  test('clears inbox state when refresh is disabled', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => false, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    inbox.rows.value = [{ Id: 'n1', IsRead: false }];
    inbox.error.value = 'old';
    inbox.loading.value = true;
    await inbox.refresh();
    expect(inbox.rows.value).toEqual([]);
    expect(inbox.error.value).toBeNull();
    expect(inbox.loading.value).toBe(false);
    expect(SearchInbox.calls.length).toBe(0);
    scope.stop();
  });

  test('stores SearchInbox failures and ignores markRead without an id', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    let failCount = 0;
    SearchInbox.mockImplementation(async () => {
      failCount += 1;
      if (failCount === 1) throw 'inbox down';
      throw new Error('inbox error');
    });
    await inbox.refresh();
    expect(inbox.error.value).toBe('inbox down');

    await inbox.refresh();
    expect(inbox.error.value).toBe('inbox error');

    await inbox.markRead('  ');
    expect(MarkRead.calls.length).toBe(0);
    await inbox.markRead('');
    expect(MarkRead.calls.length).toBe(0);
    scope.stop();
  });

  test('refreshes when notification tips fire', async () => {
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
    });
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockClear();
    await inbox.activate();
    await flushPromises();
    expect(SearchInbox.calls.length).toBeGreaterThanOrEqual(2);
    scope.stop();
  });

  test('refreshes after markRead and markAllRead succeed', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockClear();
    await inbox.markRead('n1');
    expect(MarkRead.calls[0]).toEqual([['n1']]);
    expect(SearchInbox.calls.length).toBeGreaterThan(0);

    SearchInbox.mockClear();
    await inbox.markAllRead();
    expect(MarkAllRead.calls.length).toBeGreaterThan(0);
    expect(SearchInbox.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('deactivates and stops polling', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    SearchInbox.mockClear();
    inbox.deactivate();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBe(0);
    scope.stop();
  });

  test('deactivates when the host component unmounts', async () => {
    let inbox: ReturnType<typeof useNotificationInbox> | undefined;
    const Host = defineComponent({
      setup() {
        inbox = useNotificationInbox(() => true, tipDeps());
        onMounted(() => {
          void inbox?.activate();
        });
        return () => h('div');
      },
    });
    const mounted = mountApp(Host);
    await flushPromises();
    SearchInbox.mockClear();
    mounted.unmount();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBe(0);
    expect(inbox?.rows.value).toEqual([]);
  });

  test('ignores stale refresh results after inbox becomes disabled', async () => {
    let enabled = true;
    let resolveFirst: ((rows: unknown[]) => void) | undefined;
    SearchInbox.mockImplementation(() =>
      new Promise(resolve => {
        resolveFirst = resolve;
      })
    );
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => enabled, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    const first = inbox.refresh();
    enabled = false;
    SearchInbox.mockImplementation(async () => []);
    await inbox.refresh();
    expect(inbox.rows.value).toEqual([]);
    resolveFirst?.([{ Id: 'n1', IsRead: false }]);
    await first;
    expect(inbox.rows.value).toEqual([]);
    scope.stop();
  });

  test('skips tip subscription when refresh completes after deactivate', async () => {
    let resolveInbox: ((rows: unknown[]) => void) | undefined;
    SearchInbox.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveInbox = resolve;
        })
    );
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    const pending = inbox.activate();
    inbox.deactivate();
    resolveInbox?.([]);
    await pending;
    expect(onTips.calls.length).toBe(0);
    scope.stop();
  });

  test('maps mark-all-read failures to strings', async () => {
    MarkAllRead.mockImplementation(async () => {
      throw 'mark all down';
    });
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.markAllRead();
    expect(inbox.error.value).toBe('mark all down');
    scope.stop();
  });

  test('maps mark-read non-Error failures to strings', async () => {
    MarkRead.mockImplementation(async () => {
      throw 'mark down';
    });
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.markRead('n1');
    expect(inbox.error.value).toBe('mark down');
    scope.stop();
  });

  test('ignores stale SearchInbox failures after a later refresh', async () => {
    let rejectFirst: ((err: unknown) => void) | undefined;
    let call = 0;
    SearchInbox.mockImplementation(() => {
      call += 1;
      if (call === 1) {
        return new Promise((_resolve, reject) => {
          rejectFirst = reject;
        });
      }
      return Promise.resolve([{ Id: 'n2', IsRead: true }]);
    });

    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    const first = inbox.refresh();
    await Promise.resolve();
    await inbox.refresh();
    expect(inbox.rows.value.map(row => row.Id)).toEqual(['n2']);
    rejectFirst?.(new Error('stale'));
    await first;
    expect(inbox.error.value).toBeNull();
    expect(inbox.rows.value.map(row => row.Id)).toEqual(['n2']);
    scope.stop();
  });

  test('ignores stale disabled refresh generations', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => false, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    const first = inbox.refresh();
    await inbox.refresh();
    await first;
    expect(inbox.rows.value).toEqual([]);
    expect(inbox.loading.value).toBe(false);
    scope.stop();
  });

  test('counts unread rows and ignores already-read entries', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockImplementation(async () => [
      { Id: 'n1', IsRead: false },
      { Id: 'n2', IsRead: true },
      { Id: 'n3' },
      null,
    ] as any);
    await inbox.refresh();
    expect(inbox.unreadCount.value).toBe(3);
    scope.stop();
  });

  test('does not start poll fallback when tip subscription is aborted', async () => {
    let resolveTips: (() => void) | undefined;
    onTips.mockImplementation(
      () =>
        new Promise<void>(resolve => {
          resolveTips = resolve;
        })
    );
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    await flushPromises();
    inbox.deactivate();
    resolveTips?.();
    await flushPromises();
    SearchInbox.mockClear();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBe(0);
    scope.stop();
  });

  test('starts poll fallback when the tip stream rejects while still subscribed', async () => {
    onTips.mockImplementation(async () => {
      throw new Error('stream down');
    });
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    await flushPromises();
    SearchInbox.mockClear();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBeGreaterThan(0);
    scope.stop();
  });

  test('does not start tips or poll fallback when the inbox is disabled', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => false, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    expect(onTips.calls.length).toBe(0);
    SearchInbox.mockClear();
    await sleep(POLL_MS + 20);
    expect(SearchInbox.calls.length).toBe(0);
    scope.stop();
  });

  test('returns early from a disabled refresh when deactivate races inside enabled()', async () => {
    const scope = effectScope();
    let inbox: ReturnType<typeof useNotificationInbox> | undefined;
    let armed = false;
    inbox = scope.run(() =>
      useNotificationInbox(() => {
        if (armed && inbox) {
          inbox.deactivate();
          return false;
        }
        return false;
      }, tipDeps())
    );
    if (!inbox) throw new Error('inbox missing');
    armed = true;
    inbox.rows.value = [{ Id: 'n1', IsRead: false }];
    await inbox.refresh();
    expect(inbox.rows.value).toEqual([]);
    scope.stop();
  });

  test('skips startTips when enabled flips false after activate refresh', async () => {
    const scope = effectScope();
    let calls = 0;
    const inbox = scope.run(() => useNotificationInbox(() => ++calls < 3, tipDeps()));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    expect(onTips.calls.length).toBe(0);
    scope.stop();
  });
});
