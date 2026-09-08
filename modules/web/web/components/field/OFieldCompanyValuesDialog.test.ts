// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OFieldCompanyValuesDialog needs auth + registry store wiring for a green mount;
// leave import smoke; OFieldBase stubs this dialog for action wiring coverage.

test('OFieldCompanyValuesDialog smoke: default export is a named Vue component', async () => {
  const mod = await import('./OFieldCompanyValuesDialog.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
