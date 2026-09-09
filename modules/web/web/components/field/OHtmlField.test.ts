// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// OHtmlField: accept-thin by policy — TipTap useEditor has no QJS package stub.
// Unit covers ohtml_helpers + this smoke; deep editor interaction stays in E2E.

import {
  HTML_ALLOWED_ATTR,
  HTML_ALLOWED_TAGS,
  htmlToPlaintext,
  normalizeHtmlForStore,
  sanitizeHtmlForClient,
} from './ohtml_helpers';

test('OHtmlField smoke: default export is a named Vue component', async () => {
  const mod = await import('./OHtmlField.vue');
  expect(mod.default).toBeTruthy();
  const name =
    (mod.default as { name?: string; __name?: string }).name ||
    (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBe('OHtmlField');
});

test('OHtmlField re-exports helpers via ohtml_helpers surface', () => {
  expect(typeof sanitizeHtmlForClient).toBe('function');
  expect(typeof htmlToPlaintext).toBe('function');
  expect(typeof normalizeHtmlForStore).toBe('function');
  expect(HTML_ALLOWED_TAGS).toContain('p');
  expect(HTML_ALLOWED_ATTR).toContain('href');
});
