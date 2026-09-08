// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge
// Full suite needs tip stream DI (onTips / subscribeThread) and fake timers.

test('useChatterThreadTips smoke: module exports the composable', async () => {
  const mod = await import('./useChatterThreadTips');
  expect(typeof mod.useChatterThreadTips).toBe('function');
});
