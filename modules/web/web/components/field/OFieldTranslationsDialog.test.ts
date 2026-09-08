// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OFieldTranslationsDialog needs registry stores and i18n for a green mount;
// leave import smoke; OFieldBase stubs this dialog for action wiring coverage.

test('OFieldTranslationsDialog smoke: default export is a named Vue component', async () => {
  const mod = await import('./OFieldTranslationsDialog.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
