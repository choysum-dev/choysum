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
  // dompurify's default export is a factory without `sanitize` when no DOM is
  // present (QuickJS/SSR); strip tags instead of crashing or returning raw markup.
  if (typeof purify.sanitize !== 'function') {
    // No DOM → DOMPurify cannot run. `>?` keeps stripping unterminated
    // tags (`<img src=x onerror=…`) so no active `<` can survive.
    return raw.replace(/<[^>]*>?/g, '');
  }
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
      .replace(/<hr\s*\/?>/gi, ' ')
      .replace(/<\/(p|div|h[1-6]|li|blockquote|pre|tr)>/gi, ' </$1>');
    el.innerHTML = withBreaks;
    return String(el.textContent || '')
      .replace(/\s+/g, ' ')
      .trim();
  }
  return raw
    // Match the DOM path: drop script/style bodies before stripping tags.
    .replace(/<(script|style)\b[^>]*>[\s\S]*?<\/\1>/gi, ' ')
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
  // Protocol-relative URLs are not relative paths; do not prepend `https://`.
  if (trimmed.startsWith('//')) return `https:${trimmed}`;
  // `host:port` (e.g. localhost:3000) is a host, not a scheme: only treat a
  // leading `word:` as a scheme when it is not followed solely by a port.
  const schemeMatch = /^([a-z][a-z0-9+.-]*):(?!\d+(?:[/?#]|$))/i.exec(trimmed);
  // Link.protocols only affects autolink; reject javascript:/data: etc. here.
  if (schemeMatch && !/^(https?|mailto):/i.test(trimmed)) return false;
  return schemeMatch ? trimmed : `https://${trimmed}`;
}
