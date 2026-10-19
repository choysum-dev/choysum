// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Shared list / kanban / chart paths for Token management views. */
export type TokenViewKind = 'list' | 'kanban' | 'chart';

export function resolveTokenViewKind(path: string): TokenViewKind {
  const p = String(path || '');
  if (p.includes('/kanban')) return 'kanban';
  if (p.includes('/chart')) return 'chart';
  return 'list';
}

export function tokenViewPath(kind: TokenViewKind): string {
  if (kind === 'kanban') return '/auth/tokens/kanban';
  if (kind === 'chart') return '/auth/tokens/chart';
  return '/auth/tokens';
}

/**
 * Default groupby for each Token management view. Chart needs nested
 * TokenType×Revoked; Kanban lanes are Revoked-only; List has no groups.
 */
export function tokenViewDefaultGroups(kind: TokenViewKind): string[] {
  if (kind === 'chart') return ['TokenType', 'Revoked'];
  if (kind === 'kanban') return ['Revoked'];
  return [];
}

/**
 * Write view-appropriate appliedGroups onto the shared Token store so Chart
 * defaults are not masked by a prior Kanban Revoked-only selection (and vice versa).
 */
export function ensureTokenViewGroups(store: unknown, kind: TokenViewKind): void {
  if (!store || typeof store !== 'object') return;
  const root = store as { state?: Record<string, unknown> };
  const state = (root.state || (root.state = {})) as Record<string, unknown>;
  const qs = (state.queryState || (state.queryState = {})) as Record<string, unknown>;
  qs.appliedGroups = tokenViewDefaultGroups(kind);
}

/**
 * Map Revoked group labels for chart axes / legend (Kanban uses the same wording).
 */
export function formatTokenRevokedChartLabel(
  label: string,
  labels: { revoked: string; notRevoked: string },
): string {
  const raw = String(label || '').trim();
  if (/^Revoked\s*=\s*true$/i.test(raw) || raw === 'true') return labels.revoked;
  if (/^Revoked\s*=\s*false$/i.test(raw) || raw === 'false') return labels.notRevoked;
  return raw;
}
