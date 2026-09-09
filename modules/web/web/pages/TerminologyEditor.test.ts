// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// TerminologyEditor needs pinia + vue-router + registry stores at setup;
// import smoke only until QJS mount backfill from main.

test('TerminologyEditor smoke: default export is a named Vue component', async () => {
  const mod = await import('./TerminologyEditor.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
