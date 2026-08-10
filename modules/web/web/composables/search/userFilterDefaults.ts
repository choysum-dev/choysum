// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { NamedFilter } from '@/web/web/query/types';

export type UserFilterRow = {
  Id?: string;
  Name?: string;
  Condition?: any;
  IsDefault?: boolean;
  UserId?: string | null | { Id?: string | null };
  CreatedUid?: string | null;
  UpdatedAt?: string | Date | null;
  CreatedAt?: string | Date | null;
};

/** Normalize ManyToOneRef nested `{ Id }` or bare id to a trimmed string (empty if shared/missing). */
export function resolveUserFilterUserId(userId: unknown): string {
  if (userId == null || userId === '') return '';
  if (typeof userId === 'object') {
    if (Array.isArray(userId)) return '';
    return String((userId as { Id?: unknown }).Id ?? '').trim();
  }
  return String(userId).trim();
}

/**
 * Convert a UserFilter row into a NamedFilter for OSearch menus / first-frame defaults.
 */
export function userFilterToNamedFilter(row: UserFilterRow, selected = false): NamedFilter {
  return {
    name: String(row.Name || '').trim(),
    query: (row.Condition ?? {}) as any,
    selected,
  };
}

export function isSharedUserFilterUserId(userId: unknown): boolean {
  return !resolveUserFilterUserId(userId);
}

function toMillis(raw: unknown): number {
  if (raw == null || raw === '') return 0;
  if (raw instanceof Date) {
    const t = raw.getTime();
    return Number.isFinite(t) ? t : 0;
  }
  const t = Date.parse(String(raw));
  return Number.isFinite(t) ? t : 0;
}

/** True when `a` should win over `b` as the effective default (newest first). */
export function isNewerUserFilter(a: UserFilterRow, b: UserFilterRow): boolean {
  const ua = toMillis(a.UpdatedAt);
  const ub = toMillis(b.UpdatedAt);
  if (ua !== ub) return ua > ub;
  const ca = toMillis(a.CreatedAt);
  const cb = toMillis(b.CreatedAt);
  if (ca !== cb) return ca > cb;
  return String(a.Id || '') > String(b.Id || '');
}

/**
 * Among IsDefault rows, pick the newest private or shared favorite (Odoo-aligned: no server mutex).
 */
export function pickLatestIsDefault(
  rows: UserFilterRow[] | null | undefined,
  kind: 'private' | 'shared'
): UserFilterRow | null {
  const list = (rows || []).filter(r => {
    if (!r?.IsDefault) return false;
    const shared = isSharedUserFilterUserId(r.UserId);
    return kind === 'shared' ? shared : !shared;
  });
  if (!list.length) return null;
  let best = list[0];
  for (let i = 1; i < list.length; i++) {
    if (isNewerUserFilter(list[i], best)) best = list[i];
  }
  return best;
}

/**
 * Merge order: newest private IsDefault > newest shared IsDefault > code defaultFilters.selected.
 * Multiple server IsDefault rows are allowed; callers should pass the latest per bucket
 * (see {@link pickLatestIsDefault}).
 */
export function mergeUserFilterDefaults(opts: {
  privateDefault?: UserFilterRow | null;
  sharedDefault?: UserFilterRow | null;
  codeDefaults?: NamedFilter[] | NamedFilter | null;
}): NamedFilter[] {
  const codeList: NamedFilter[] = !opts.codeDefaults
    ? []
    : Array.isArray(opts.codeDefaults)
      ? opts.codeDefaults
      : [opts.codeDefaults];

  const serverRow =
    opts.privateDefault && opts.privateDefault.IsDefault
      ? opts.privateDefault
      : opts.sharedDefault && opts.sharedDefault.IsDefault
        ? opts.sharedDefault
        : null;
  if (serverRow) {
    const winner = userFilterToNamedFilter(serverRow, true);
    const rest = codeList
      .filter(nf => nf && typeof nf.name === 'string' && nf.name.length > 0 && nf.name !== winner.name)
      .map(nf => ({ ...nf, selected: false }));
    return [winner, ...rest];
  }

  return codeList.map(nf => ({ ...nf }));
}
