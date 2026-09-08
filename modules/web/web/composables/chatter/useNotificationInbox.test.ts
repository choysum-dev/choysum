// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge
// Full suite needs notification store + tip stream DI and fake timers.

test('useNotificationInbox smoke: module exports the composable', async () => {
  const mod = await import('./useNotificationInbox');
  expect(typeof mod.useNotificationInbox).toBe('function');
});
