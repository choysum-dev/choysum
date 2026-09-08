// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// ONotificationBell needs useNotificationInbox and auth store mocks at setup;
// full mount needs store/composable stubs at setup; import smoke only for this knife.

test('ONotificationBell smoke: default export is a named Vue component', async () => {
  const mod = await import('./ONotificationBell.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
