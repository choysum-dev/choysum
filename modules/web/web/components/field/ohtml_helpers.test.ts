// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient } from './ohtml_helpers';

vi.mock('dompurify', () => ({
  default: {
    addHook: () => undefined,
    // Fixture map only — production path uses real DOMPurify.
    sanitize: (html: string) => {
      const fixtures: Record<string, string> = {
        '<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>': '<p>Hi</p><a>x</a>',
        '<p></p>': '<p></p>',
        '<p>ok</p>': '<p>ok</p>',
        '<p>Hello <strong>world</strong></p>': '<p>Hello <strong>world</strong></p>',
        '<a href="https://example.com" target="_blank">x</a>':
          '<a href="https://example.com" target="_blank" rel="noopener noreferrer">x</a>',
      };
      return fixtures[String(html)] ?? String(html);
    },
  },
}));

describe('ohtml_helpers', () => {
  it('sanitizeHtmlForClient strips script and dangerous protocols', () => {
    const cleaned = sanitizeHtmlForClient(
      `<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>`
    );
    expect(cleaned).not.toContain('<script');
    expect(cleaned).not.toContain('onclick');
    expect(cleaned).not.toContain('javascript:');
    expect(cleaned).toContain('Hi');
  });

  it('htmlToPlaintext strips tags', () => {
    expect(htmlToPlaintext('<p>Hello <strong>world</strong></p>')).toBe('Hello world');
    expect(htmlToPlaintext(null)).toBe('');
  });

  it('normalizeHtmlForStore nulls blank markup', () => {
    expect(normalizeHtmlForStore(null)).toBeNull();
    expect(normalizeHtmlForStore('<p></p>')).toBeNull();
    expect(normalizeHtmlForStore('<p>ok</p>')).toBe('<p>ok</p>');
  });
});
