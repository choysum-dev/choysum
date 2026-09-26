// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { htmlToPlaintext, normalizeHtmlForStore, resolveChoyHtmlLinkHref, sanitizeHtmlForClient, type DomPurifyLike } from './htmlHelpers';

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
        '<p>Hello</p><p>world</p>': '<p>Hello</p><p>world</p>',
        '<hr>': '<hr>',
      };
      if (html === 'HOOK') return blankAttrs.rel || '';
      const known = fixtures[String(html)];
      if (known !== undefined) return known;
      // Build the anchor from the hook mutation so the assertion depends on
      // afterSanitizeAttributes rather than a hard-coded rel fixture.
      if (html === '<a href="https://example.com" target="_blank">x</a>') {
        return `<a href="https://example.com" target="_blank" rel="${blankAttrs.rel ?? ''}">x</a>`;
      }
      return String(html);
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

  test('sanitizeHtmlForClient handles null and empty', () => {
    const purify = createTestPurify();
    expect(sanitizeHtmlForClient(null, { purify })).toBe('');
    expect(sanitizeHtmlForClient('', { purify })).toBe('');
  });

  test('sanitizeHtmlForClient strips tags when purify.sanitize is missing', () => {
    const purify = {
      addHook: () => undefined,
      sanitize: undefined as unknown as DomPurifyLike['sanitize'],
    };
    expect(sanitizeHtmlForClient('<p>Hi</p><script>x</script>', { purify })).toBe('Hix');
    // Unterminated tag must not survive the no-DOM fallback.
    expect(sanitizeHtmlForClient('<img src=x onerror=alert(1)', { purify })).toBe('');
  });

  test('sanitizeHtmlForClient strips tags when purify.sanitize throws', () => {
    const purify = {
      addHook: () => undefined,
      sanitize: () => {
        throw new Error('sanitize failed');
      },
    };
    expect(sanitizeHtmlForClient('<p>Hi</p><script>x</script>', { purify })).toBe('Hix');
    expect(sanitizeHtmlForClient('<img src=x onerror=alert(1)', { purify })).toBe('');
  });

  test('htmlToPlaintext strips tags (non-DOM path)', () => {
    const purify = createTestPurify();
    // Prefer non-DOM fallback: avoid mutating global document in QuickJS.
    const previousDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'document');
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: undefined,
    });
    try {
      expect(htmlToPlaintext('<p>Hello <strong>world</strong></p>', { purify })).toBe('Hello world');
      expect(htmlToPlaintext('<script>alert(1)</script><p>ok</p>', { purify })).toBe('ok');
      expect(htmlToPlaintext('<style>.x{}</style>hi', { purify })).toBe('hi');
      expect(htmlToPlaintext(null, { purify })).toBe('');
      expect(htmlToPlaintext('', { purify })).toBe('');
    } finally {
      if (previousDescriptor) {
        Object.defineProperty(globalThis, 'document', previousDescriptor);
      } else {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete (globalThis as { document?: unknown }).document;
      }
    }
  });

  test('htmlToPlaintext DOM path preserves separators between blocks', () => {
    const purify = createTestPurify();
    const previousDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'document');
    let stored = '';
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: {
        createElement: () => ({
          set innerHTML(html: string) {
            stored = String(html).replace(/<[^>]+>/g, '');
          },
          get textContent() {
            return stored;
          },
        }),
      },
    });
    try {
      expect(htmlToPlaintext('<p>Hello</p><p>world</p>', { purify })).toBe('Hello world');
      expect(htmlToPlaintext('a<hr>b', { purify })).toBe('a b');
    } finally {
      if (previousDescriptor) {
        Object.defineProperty(globalThis, 'document', previousDescriptor);
      } else {
        // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
        delete (globalThis as { document?: unknown }).document;
      }
    }
  });

  test('normalizeHtmlForStore maps empty markup to null and keeps hr', () => {
    const purify = createTestPurify();
    expect(normalizeHtmlForStore('<p></p>', { purify })).toBeNull();
    expect(normalizeHtmlForStore('<p>ok</p>', { purify })).toBe('<p>ok</p>');
    expect(normalizeHtmlForStore('<hr>', { purify })).toBe('<hr>');
    expect(normalizeHtmlForStore(null, { purify })).toBeNull();
  });

  test('resolveChoyHtmlLinkHref validates schemes', () => {
    expect(resolveChoyHtmlLinkHref('')).toBeNull();
    expect(resolveChoyHtmlLinkHref('  ')).toBeNull();
    expect(resolveChoyHtmlLinkHref('javascript:alert(1)')).toBe(false);
    expect(resolveChoyHtmlLinkHref('data:text/html,x')).toBe(false);
    expect(resolveChoyHtmlLinkHref('https://example.com')).toBe('https://example.com');
    expect(resolveChoyHtmlLinkHref('http://example.com')).toBe('http://example.com');
    expect(resolveChoyHtmlLinkHref('mailto:a@b.com')).toBe('mailto:a@b.com');
    expect(resolveChoyHtmlLinkHref('example.com/path')).toBe('https://example.com/path');
    expect(resolveChoyHtmlLinkHref('localhost:3000')).toBe('https://localhost:3000');
    expect(resolveChoyHtmlLinkHref('example.com:8080/path')).toBe('https://example.com:8080/path');
    expect(resolveChoyHtmlLinkHref('//example.com/path')).toBe('https://example.com/path');
  });
});
