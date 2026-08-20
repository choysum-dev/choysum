// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { effectScope } from 'vue';
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
});
