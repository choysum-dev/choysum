// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import DOMPurify from 'dompurify';

/** TipTap StarterKit + Link aligned allowlist (mirrors Go bluemonday policy). */
export const HTML_ALLOWED_TAGS = [
  'p',
  'br',
  'hr',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'strong',
  'b',
  'em',
  'i',
  'u',
  's',
  'strike',
  'del',
  'ul',
  'ol',
  'li',
  'blockquote',
  'code',
  'pre',
  'a',
] as const;

export const HTML_ALLOWED_ATTR = ['href', 'target', 'rel', 'class'] as const;

const purifyConfig = {
  ALLOWED_TAGS: [...HTML_ALLOWED_TAGS],
  ALLOWED_ATTR: [...HTML_ALLOWED_ATTR],
  ALLOW_DATA_ATTR: false,
};

/** FE display / outbound sanitize (defense in depth; server remains authoritative). */
export function sanitizeHtmlForClient(html: string | null | undefined): string {
  if (html == null) return '';
  const raw = String(html);
  if (!raw) return '';
  return DOMPurify.sanitize(raw, purifyConfig);
}

/** Strip tags for List / Search plaintext projection (D9). */
export function htmlToPlaintext(html: string | null | undefined): string {
  if (html == null) return '';
  return String(html)
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/&#160;/gi, ' ')
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/&quot;/gi, '"')
    .replace(/\s+/g, ' ')
    .trim();
}

/** Empty / blank-tag HTML → null for store writes (align D10). */
export function normalizeHtmlForStore(html: string | null | undefined): string | null {
  if (html == null) return null;
  const cleaned = sanitizeHtmlForClient(html);
  if (!cleaned || htmlToPlaintext(cleaned) === '') return null;
  return cleaned;
}
