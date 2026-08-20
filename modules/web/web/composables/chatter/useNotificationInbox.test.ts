// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, effectScope, h, onMounted, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const SearchInbox = vi.fn();
const MarkRead = vi.fn();
const MarkAllRead = vi.fn();
const onTips = vi.fn();
const subscribeNotifications = vi.fn(() => ({}));

vi.mock('./chatterStores', () => ({
  getNotificationStore: () => ({ SearchInbox, MarkRead, MarkAllRead }),
}));

vi.mock('@/core/web/tip', () => ({
  onTips: (...args: unknown[]) => onTips(...args),
  subscribeNotifications: (...args: unknown[]) => subscribeNotifications(...args),
}));

import { useNotificationInbox } from './useNotificationInbox';

describe('useNotificationInbox', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    SearchInbox.mockReset();
    MarkRead.mockReset();
    MarkAllRead.mockReset();
    onTips.mockReset();
    subscribeNotifications.mockReset();
    SearchInbox.mockResolvedValue([]);
    MarkRead.mockResolvedValue(1);
    MarkAllRead.mockResolvedValue(1);
    onTips.mockResolvedValue(undefined);
    subscribeNotifications.mockReturnValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('starts poll fallback after the notification tip stream ends', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    await Promise.resolve();
    await Promise.resolve();
    SearchInbox.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(SearchInbox).toHaveBeenCalled();
    scope.stop();
  });

  it('stores mark-read failures without throwing', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    MarkRead.mockRejectedValue(new Error('mark failed'));
    await inbox.markRead('n1');
    expect(inbox.error.value).toBe('mark failed');
    MarkAllRead.mockRejectedValue(new Error('mark all failed'));
    await inbox.markAllRead();
    expect(inbox.error.value).toBe('mark all failed');
    scope.stop();
  });

  it('ignores stale SearchInbox results after a later refresh', async () => {
    let resolveFirst: ((rows: unknown[]) => void) | undefined;
    SearchInbox.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveFirst = resolve;
        })
    ).mockResolvedValueOnce([{ Id: 'n2', IsRead: true }]);

    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
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

  it('does not start tips after deactivate during activate', async () => {
    let resolveInbox: ((rows: unknown[]) => void) | undefined;
    SearchInbox.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveInbox = resolve;
        })
    );
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    const pending = inbox.activate();
    await Promise.resolve();
    inbox.deactivate();
    resolveInbox?.([]);
    await pending;
    expect(onTips).not.toHaveBeenCalled();
    scope.stop();
  });

  it('clears inbox state when refresh is disabled', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => false));
    if (!inbox) throw new Error('inbox missing');
    inbox.rows.value = [{ Id: 'n1', IsRead: false }];
    inbox.error.value = 'old';
    inbox.loading.value = true;
    await inbox.refresh();
    expect(inbox.rows.value).toEqual([]);
    expect(inbox.error.value).toBeNull();
    expect(inbox.loading.value).toBe(false);
    expect(SearchInbox).not.toHaveBeenCalled();
    scope.stop();
  });

  it('stores SearchInbox failures and ignores markRead without an id', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockRejectedValueOnce('inbox down');
    await inbox.refresh();
    expect(inbox.error.value).toBe('inbox down');

    await inbox.markRead('  ');
    expect(MarkRead).not.toHaveBeenCalled();
    scope.stop();
  });

  it('refreshes when notification tips fire', async () => {
    onTips.mockImplementation(async (_stream, callback: () => Promise<void>) => {
      await callback();
    });
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockClear();
    await inbox.activate();
    await Promise.resolve();
    expect(SearchInbox.mock.calls.length).toBeGreaterThanOrEqual(2);
    scope.stop();
  });

  it('refreshes after markRead and markAllRead succeed', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    SearchInbox.mockClear();
    await inbox.markRead('n1');
    expect(MarkRead).toHaveBeenCalledWith(['n1']);
    expect(SearchInbox).toHaveBeenCalled();

    SearchInbox.mockClear();
    await inbox.markAllRead();
    expect(MarkAllRead).toHaveBeenCalled();
    expect(SearchInbox).toHaveBeenCalled();
    scope.stop();
  });

  it('deactivates and stops polling', async () => {
    const scope = effectScope();
    const inbox = scope.run(() => useNotificationInbox(() => true));
    if (!inbox) throw new Error('inbox missing');
    await inbox.activate();
    SearchInbox.mockClear();
    inbox.deactivate();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(SearchInbox).not.toHaveBeenCalled();
    scope.stop();
  });

  it('deactivates when the host component unmounts', async () => {
    const scope = effectScope();
    let inbox: ReturnType<typeof useNotificationInbox> | undefined;
    const Host = defineComponent({
      setup() {
        inbox = useNotificationInbox(() => true);
        onMounted(() => {
          void inbox?.activate();
        });
        return () => h('div');
      },
    });
    const wrapper = mount(Host);
    await Promise.resolve();
    SearchInbox.mockClear();
    wrapper.unmount();
    scope.stop();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(SearchInbox).not.toHaveBeenCalled();
    expect(inbox?.rows.value).toEqual([]);
  });
});
