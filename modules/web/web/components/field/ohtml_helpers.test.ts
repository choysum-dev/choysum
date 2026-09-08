// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient, type DomPurifyLike } from './ohtml_helpers';

function createTestPurify(): DomPurifyLike & { hooks: Array<(node: any) => void> } {
  const hooks: Array<(node: any) => void> = [];
  return {
    hooks,
    addHook: (_name: string, fn: (node: any) => void) => {
      hooks.push(fn);
    },
    sanitize: (html: string) => {
      const run = (node: any) => {
        for (const fn of hooks) fn(node);
      };
      run({
        nodeName: 'P',
        getAttribute: () => null,
        setAttribute: () => undefined,
      });
      run({
        nodeName: 'A',
        getAttribute: (key: string) => (key === 'target' ? '_self' : null),
        setAttribute: () => undefined,
      });
      const blankAttrs: Record<string, string> = { target: '_blank' };
      run({
        nodeName: 'A',
        getAttribute: (key: string) => blankAttrs[key] ?? null,
        setAttribute: (key: string, value: string) => {
          blankAttrs[key] = value;
        },
      });

      const fixtures: Record<string, string> = {
        '<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>': '<p>Hi</p><a>x</a>',
        '<p></p>': '<p></p>',
        '<p>ok</p>': '<p>ok</p>',
        '<p>Hello <strong>world</strong></p>': '<p>Hello <strong>world</strong></p>',
        '<a href="https://example.com" target="_blank">x</a>':
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer">x</a>',
        '<a href="https://example.com" target="_blank">y</a>':
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer">y</a>',
      };
      if (html === 'HOOK') return blankAttrs.rel || '';
      return fixtures[String(html)] ?? String(html);
    },
  };
}

describe('ohtml_helpers', () => {
  test('sanitizeHtmlForClient strips script and dangerous protocols', () => {
    const purify = createTestPurify();
    const cleaned = sanitizeHtmlForClient(
      `<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>`,
      { purify }
    );
    expect(cleaned).not.toContain('<script');
    expect(cleaned).not.toContain('onclick');
    expect(cleaned).not.toContain('javascript:');
    expect(cleaned).toContain('Hi');
  });

  test('sanitizeHtmlForClient handles null/empty and installs hooks once', () => {
    const purify = createTestPurify();
    expect(sanitizeHtmlForClient(null, { purify })).toBe('');
    expect(sanitizeHtmlForClient(undefined, { purify })).toBe('');
    expect(sanitizeHtmlForClient('', { purify })).toBe('');
    expect(sanitizeHtmlForClient('HOOK', { purify })).toBe('noopener noreferrer');
    const afterFirst = purify.hooks.length;
    expect(afterFirst).toBe(1);
    sanitizeHtmlForClient('HOOK', { purify });
    expect(purify.hooks.length).toBe(afterFirst);
  });

  test('htmlToPlaintext strips tags and supports non-DOM fallback', () => {
    const purify = createTestPurify();
    expect(htmlToPlaintext('<p>Hello <strong>world</strong></p>', { purify })).toBe('Hello world');
    expect(htmlToPlaintext(null, { purify })).toBe('');
    expect(htmlToPlaintext('', { purify })).toBe('');

    const originalDocument = globalThis.document;
    Object.defineProperty(globalThis, 'document', { value: undefined, configurable: true });
    try {
      expect(htmlToPlaintext('<b>hi</b>', { purify })).toBe('hi');
    } finally {
      Object.defineProperty(globalThis, 'document', { value: originalDocument, configurable: true });
    }
  });

  test('normalizeHtmlForStore nulls blank markup', () => {
    const purify = createTestPurify();
    expect(normalizeHtmlForStore(null, { purify })).toBeNull();
    expect(normalizeHtmlForStore('', { purify })).toBeNull();
    expect(normalizeHtmlForStore('<p></p>', { purify })).toBeNull();
    expect(normalizeHtmlForStore('<p>ok</p>', { purify })).toBe('<p>ok</p>');
    expect(normalizeHtmlForStore('<hr>', { purify })).toBe('<hr>');
  });
});
