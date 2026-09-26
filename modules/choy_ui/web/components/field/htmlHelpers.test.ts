// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient, type DomPurifyLike } from './htmlHelpers';

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
      };
      if (html === 'HOOK') return blankAttrs.rel || '';
      return fixtures[String(html)] ?? String(html);
    },
  };
}

describe('htmlHelpers', () => {
  test('sanitizeHtmlForClient strips script and dangerous protocols', () => {
    const purify = createTestPurify();
    const cleaned = sanitizeHtmlForClient(
      `<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>`,
      { purify },
    );
    expect(cleaned).not.toContain('<script');
    expect(cleaned).not.toContain('onclick');
    expect(cleaned).toContain('<p>Hi</p>');
  });

  test('sanitizeHtmlForClient adds rel on target=_blank', () => {
    const purify = createTestPurify();
    const cleaned = sanitizeHtmlForClient('<a href="https://example.com" target="_blank">x</a>', {
      purify,
    });
    expect(cleaned).toContain('rel="noopener noreferrer"');
  });

  test('htmlToPlaintext strips tags (non-DOM path)', () => {
    const purify = createTestPurify();
    expect(htmlToPlaintext('<p>Hello <strong>world</strong></p>', { purify })).toBe('Hello world');
  });

  test('normalizeHtmlForStore maps empty markup to null', () => {
    const purify = createTestPurify();
    expect(normalizeHtmlForStore('<p></p>', { purify })).toBeNull();
    expect(normalizeHtmlForStore('<p>ok</p>', { purify })).toBe('<p>ok</p>');
  });
});
