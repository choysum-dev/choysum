// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import DOMPurify from 'dompurify';

/** TipTap StarterKit + Link aligned allowlist. */
export const CHOY_HTML_ALLOWED_TAGS = [
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

export const CHOY_HTML_ALLOWED_ATTR = ['href', 'target', 'rel', 'class'] as const;

const purifyConfig = {
  ALLOWED_TAGS: [...CHOY_HTML_ALLOWED_TAGS],
  ALLOWED_ATTR: [...CHOY_HTML_ALLOWED_ATTR],
  ALLOW_DATA_ATTR: false,
};

export type DomPurifyLike = {
  addHook: (name: string, fn: (node: any) => void) => void;
  sanitize: (html: string, config?: unknown) => string;
};

export type SanitizeHtmlDeps = {
  purify?: DomPurifyLike;
};

const hooksInstalled = new WeakSet<object>();

function resolvePurify(deps?: SanitizeHtmlDeps): DomPurifyLike {
  return deps?.purify ?? (DOMPurify as unknown as DomPurifyLike);
}

function ensureDomPurifyHooks(purify: DomPurifyLike): void {
  if (hooksInstalled.has(purify)) return;
  hooksInstalled.add(purify);
  purify.addHook('afterSanitizeAttributes', node => {
    if (node.nodeName === 'A' && node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer');
    }
  });
}

/** FE display / outbound sanitize (defense in depth; server remains authoritative). */
export function sanitizeHtmlForClient(html: string | null | undefined, deps?: SanitizeHtmlDeps): string {
  if (html == null) return '';
  const raw = String(html);
  if (!raw) return '';
  const purify = resolvePurify(deps);
  ensureDomPurifyHooks(purify);
  return purify.sanitize(raw, purifyConfig);
}

/** Strip tags for list / search plaintext projection. Prefer DOM textContent. */
export function htmlToPlaintext(html: string | null | undefined, deps?: SanitizeHtmlDeps): string {
  if (html == null) return '';
  const raw = String(html);
  if (!raw) return '';
  if (typeof document !== 'undefined') {
    const el = document.createElement('div');
    // Insert separators for block / br so adjacent blocks do not concatenate
    // (e.g. <p>Hello</p><p>world</p> → "Hello world", not "Helloworld").
    const withBreaks = sanitizeHtmlForClient(raw, deps)
      .replace(/<br\s*\/?>/gi, ' ')
      .replace(/<\/(p|div|h[1-6]|li|blockquote|pre|tr|hr)>/gi, ' </$1>');
    el.innerHTML = withBreaks;
    return String(el.textContent || '')
      .replace(/\s+/g, ' ')
      .trim();
  }
  return raw
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

/** Empty / blank-tag HTML → null for store writes. */
export function normalizeHtmlForStore(html: string | null | undefined, deps?: SanitizeHtmlDeps): string | null {
  if (html == null) return null;
  const cleaned = sanitizeHtmlForClient(html, deps);
  if (!cleaned) return null;
  if (/<hr\b/i.test(cleaned)) return cleaned;
  if (htmlToPlaintext(cleaned, deps) === '') return null;
  return cleaned;
}

/**
 * Normalize a user-entered link href for TipTap setLink.
 * Returns `null` to unset, `false` when the scheme is rejected, otherwise the href.
 */
export function resolveChoyHtmlLinkHref(raw: string): string | null | false {
  const trimmed = String(raw ?? '').trim();
  if (!trimmed) return null;
  const hasScheme = /^[a-z][a-z0-9+.-]*:/i.test(trimmed);
  // Link.protocols only affects autolink; reject javascript:/data: etc. here.
  if (hasScheme && !/^(https?|mailto):/i.test(trimmed)) return false;
  return hasScheme ? trimmed : `https://${trimmed}`;
}
