// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient } from './ohtml_helpers';

const purifyState = vi.hoisted(() => {
  const hooks: Array<(node: any) => void> = [];
  return { hooks };
});

vi.mock('dompurify', () => ({
  default: {
    addHook: (_name: string, fn: (node: any) => void) => {
      purifyState.hooks.push(fn);
    },
    sanitize: (html: string) => {
      const run = (node: any) => {
        for (const fn of purifyState.hooks) fn(node);
      };
      // Exercise hook predicate branches.
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
      };
      if (html === 'HOOK') return blankAttrs.rel || '';
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

  it('sanitizeHtmlForClient handles null/empty and installs hooks once', () => {
    expect(sanitizeHtmlForClient(null)).toBe('');
    expect(sanitizeHtmlForClient(undefined)).toBe('');
    expect(sanitizeHtmlForClient('')).toBe('');
    const before = purifyState.hooks.length;
    expect(sanitizeHtmlForClient('HOOK')).toBe('noopener noreferrer');
    sanitizeHtmlForClient('HOOK');
    expect(purifyState.hooks.length).toBe(before);
  });

  it('htmlToPlaintext strips tags and supports non-DOM fallback', () => {
    expect(htmlToPlaintext('<p>Hello <strong>world</strong></p>')).toBe('Hello world');
    expect(htmlToPlaintext(null)).toBe('');
    expect(htmlToPlaintext('')).toBe('');

    const originalDocument = globalThis.document;
    Object.defineProperty(globalThis, 'document', { value: undefined, configurable: true });
    try {
      expect(htmlToPlaintext('<b>hi</b>')).toBe('hi');
    } finally {
      Object.defineProperty(globalThis, 'document', { value: originalDocument, configurable: true });
    }
  });

  it('normalizeHtmlForStore nulls blank markup', () => {
    expect(normalizeHtmlForStore(null)).toBeNull();
    expect(normalizeHtmlForStore('<p></p>')).toBeNull();
    expect(normalizeHtmlForStore('<p>ok</p>')).toBe('<p>ok</p>');
  });
});
