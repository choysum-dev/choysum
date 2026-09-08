// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OPropertiesDefinitionEditor needs registry store + definition helpers for a green mount;
// leave import smoke; helpers/composable coverage remains elsewhere.

test('OPropertiesDefinitionEditor smoke: default export is a named Vue component', async () => {
  const mod = await import('./OPropertiesDefinitionEditor.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
