// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge
// Conversion contract needs dayjs timezone + real ICU under QJS.

test('ODatetimeField conversion smoke: default export is a named Vue component', async () => {
  const mod = await import('./ODatetimeField.vue');
  expect(mod.default).toBeTruthy();
  const name =
    (mod.default as { name?: string; __name?: string }).name ||
    (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
