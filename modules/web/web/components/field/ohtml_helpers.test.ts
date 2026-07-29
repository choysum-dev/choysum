// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { htmlToPlaintext, normalizeHtmlForStore, sanitizeHtmlForClient } from './ohtml_helpers';

vi.mock('dompurify', () => ({
  default: {
    sanitize: (html: string, cfg?: { ALLOWED_TAGS?: string[] }) => {
      let out = String(html);
      out = out.replace(/<script[\s\S]*?<\/script>/gi, '');
      out = out.replace(/\son\w+="[^"]*"/gi, '');
      out = out.replace(/javascript:/gi, '');
      if (cfg?.ALLOWED_TAGS && !cfg.ALLOWED_TAGS.includes('img')) {
        out = out.replace(/<img\b[^>]*>/gi, '');
      }
      if (cfg?.ALLOWED_TAGS && !cfg.ALLOWED_TAGS.includes('table')) {
        out = out.replace(/<\/?table\b[^>]*>/gi, '').replace(/<\/?tr\b[^>]*>/gi, '').replace(/<\/?td\b[^>]*>/gi, '');
      }
      return out;
    },
  },
}));

describe('ohtml_helpers', () => {
  it('sanitizeHtmlForClient strips script and dangerous protocols', () => {
    const cleaned = sanitizeHtmlForClient(`<script>alert(1)</script><p onclick="x">Hi</p><a href="javascript:alert(1)">x</a>`);
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
