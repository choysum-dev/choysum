// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type DetailPathTarget =
  | string
  | {
      name: string;
      params?: Record<string, string>;
      query?: Record<string, string>;
    };

export type DetailPathProp =
  | string
  | ((row: Record<string, unknown>) => DetailPathTarget | null | undefined);

/**
 * Resolve a detailPath template or factory for a row.
 * String templates replace `:id` / `{id}` with the record Id.
 */
export function resolveDetailPath(
  detailPath: DetailPathProp | null | undefined,
  row: Record<string, unknown> | null | undefined
): DetailPathTarget | null {
  if (detailPath == null || !row) return null;
  const id = String(row.Id ?? row.id ?? '').trim();
  if (typeof detailPath === 'function') {
    const out = detailPath(row);
    if (out == null || out === '') return null;
    if (typeof out === 'string') {
      return applyIdToPathTemplate(out, id);
    }
    return out;
  }
  const path = String(detailPath).trim();
  if (!path) return null;
  if (!id) return null;
  return applyIdToPathTemplate(path, id);
}

export function applyIdToPathTemplate(template: string, id: string): string | null {
  const t = String(template || '').trim();
  if (!t || !id) return null;
  const encodedId = encodeURIComponent(id);
  if (t.includes(':id') || t.includes('{id}')) {
    // Function replacer avoids `$` sequences in id being treated as replace tokens.
    return t
      .replace(/:id\b/g, () => encodedId)
      .replace(/\{id\}/g, () => encodedId);
  }
  // Preserve ?query and #hash when appending /id to a bare prefix.
  let pathPart = t;
  let suffix = '';
  const hashIdx = t.indexOf('#');
  const queryIdx = t.indexOf('?');
  const cut =
    hashIdx >= 0 && queryIdx >= 0
      ? Math.min(hashIdx, queryIdx)
      : hashIdx >= 0
        ? hashIdx
        : queryIdx;
  if (cut >= 0) {
    pathPart = t.slice(0, cut);
    suffix = t.slice(cut);
  }
  const base = pathPart.replace(/\/+$/, '');
  if (base.endsWith(`/${encodedId}`) || base.endsWith(`/${id}`)) {
    return `${base}${suffix}`;
  }
  return `${base}/${encodedId}${suffix}`;
}

export function canOpenDetailRow(
  canOpenDetail: boolean | ((row: Record<string, unknown>) => boolean) | null | undefined,
  row: Record<string, unknown> | null | undefined
): boolean {
  if (canOpenDetail == null) return true;
  if (typeof canOpenDetail === 'function') {
    // Parameterless ACL getters (e.g. () => hasAction(...)) still run when row is missing.
    return !!canOpenDetail(row ?? {});
  }
  return !!canOpenDetail;
}
