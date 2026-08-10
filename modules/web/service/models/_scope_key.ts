// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Normalize a route path into SavedFilter.ScopeKey (keep in sync with
 * modules/web/web/composables/search/scopeKey.ts).
 */
export function normalizeScopeKey(path: string | null | undefined): string {
  let p = String(path ?? '').trim();
  if (!p) return '';
  const q = p.indexOf('?');
  if (q >= 0) p = p.slice(0, q);
  const h = p.indexOf('#');
  if (h >= 0) p = p.slice(0, h);
  p = p.replace(/\\/g, '/').replace(/\/+/g, '/');
  if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1);
  if (!p.startsWith('/') && p.length > 0) p = `/${p}`;
  return p
    .split('/')
    .map(seg => {
      if (!seg) return seg;
      if (/^\d+$/.test(seg)) return ':id';
      if (/^[a-z0-9_-]{16,}$/i.test(seg)) return ':id';
      return seg;
    })
    .join('/');
}
