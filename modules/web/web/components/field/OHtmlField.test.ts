// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OHtmlField calls TipTap useEditor at setup; QJS FE unit has no package stub for @tiptap/*.
// Sanitize/plaintext helpers remain covered in ohtml_helpers.test.ts.

test('OHtmlField smoke: default export is a named Vue component', async () => {
  const mod = await import('./OHtmlField.vue');
  expect(mod.default).toBeTruthy();
  const name = (mod.default as { name?: string; __name?: string }).name
    || (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
