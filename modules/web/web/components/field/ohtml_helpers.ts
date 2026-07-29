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

let domPurifyHooksInstalled = false;

function ensureDomPurifyHooks(): void {
  if (domPurifyHooksInstalled) return;
  domPurifyHooksInstalled = true;
  DOMPurify.addHook('afterSanitizeAttributes', node => {
    if (node.nodeName === 'A' && node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer');
    }
  });
}

/** FE display / outbound sanitize (defense in depth; server remains authoritative). */
export function sanitizeHtmlForClient(html: string | null | undefined): string {
  if (html == null) return '';
  const raw = String(html);
  if (!raw) return '';
  ensureDomPurifyHooks();
  return DOMPurify.sanitize(raw, purifyConfig);
}

/** Strip tags for List / Search plaintext projection (D9). Prefer DOM textContent. */
export function htmlToPlaintext(html: string | null | undefined): string {
  if (html == null) return '';
  const raw = String(html);
  if (!raw) return '';
  if (typeof document !== 'undefined') {
    const el = document.createElement('div');
    el.innerHTML = sanitizeHtmlForClient(raw);
    return String(el.textContent || '')
      .replace(/\s+/g, ' ')
      .trim();
  }
  // Non-DOM fallback: drop tags only (no entity decode chain).
  return raw
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

/** Empty / blank-tag HTML → null for store writes (align D10). */
export function normalizeHtmlForStore(html: string | null | undefined): string | null {
  if (html == null) return null;
  const cleaned = sanitizeHtmlForClient(html);
  if (!cleaned) return null;
  // Allowlisted void/structural markup (e.g. <hr>) is content even without text.
  if (/<hr\b/i.test(cleaned)) return cleaned;
  if (htmlToPlaintext(cleaned) === '') return null;
  return cleaned;
}
