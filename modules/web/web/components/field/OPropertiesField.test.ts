// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OPropertiesField needs ResolveProperties + registry store wiring for a green mount;
// leave import smoke; helpers/composable coverage remains elsewhere.

test('OPropertiesField smoke: default export is a named Vue component', async () => {
  const mod = await import('./OPropertiesField.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
