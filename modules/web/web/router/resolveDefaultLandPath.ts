// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';

/** Module Board fallback when no navigable menu leaf exists. */
export const MODULE_BOARD_PATH = '/meta/modules';

export type ResolveDefaultLandPathOptions = {
  /** Full menu tree (preferred). Sibling order follows `order` then id. */
  menus?: readonly MenuItem[];
  /**
   * Flat leaf paths (optional). Used when callers already flattened the tree.
   * Ignored when `menus` is provided.
   */
  menuLeafPaths?: readonly string[];
  /** Optional path filter (permission / reachability). */
  canNavigate?: (path: string) => boolean;
  /** Override Module Board path (tests). */
  moduleBoardPath?: string;
};

function normalizeAppPath(raw: unknown): string | null {
  const path = String(raw ?? '').trim();
  if (!path || !path.startsWith('/') || path.startsWith('//')) return null;
  if (path.length > 1 && path.endsWith('/')) return path.slice(0, -1);
  return path;
}

function isNavigableLeaf(item: MenuItem): boolean {
  if (item.hidden || item.disabled || item.externalLink) return false;
  const path = normalizeAppPath(item.path);
  if (!path) return false;
  const kids = item.children;
  // Path-bearing nodes without children are leaves; parents with children are walked.
  return !kids || kids.length === 0;
}

function compareMenuOrder(a: MenuItem, b: MenuItem): number {
  const ao = Number.isFinite(a.order) ? Number(a.order) : Number.POSITIVE_INFINITY;
  const bo = Number.isFinite(b.order) ? Number(b.order) : Number.POSITIVE_INFINITY;
  if (ao !== bo) return ao - bo;
  return String(a.id || '').localeCompare(String(b.id || ''));
}

/**
 * DFS the first navigable in-app leaf path under `menus`, or null.
 */
export function findFirstNavigableMenuPath(
  menus: readonly MenuItem[],
  canNavigate?: (path: string) => boolean,
): string | null {
  const roots = [...menus].sort(compareMenuOrder);
  const walk = (items: readonly MenuItem[]): string | null => {
    const ordered = [...items].sort(compareMenuOrder);
    for (const item of ordered) {
      if (item.hidden || item.disabled) continue;
      const kids = item.children;
      if (kids && kids.length > 0) {
        const hit = walk(kids);
        if (hit) return hit;
        continue;
      }
      if (!isNavigableLeaf(item)) continue;
      const path = normalizeAppPath(item.path)!;
      if (canNavigate && !canNavigate(path)) continue;
      return path;
    }
    return null;
  };
  return walk(roots);
}

function firstAllowedLeafPath(
  paths: readonly string[],
  canNavigate?: (path: string) => boolean,
): string | null {
  for (const raw of paths) {
    const path = normalizeAppPath(raw);
    if (!path) continue;
    if (canNavigate && !canNavigate(path)) continue;
    return path;
  }
  return null;
}

/**
 * Default land path: first navigable menu leaf, else Module Board.
 * Callers that need permission filtering pass `canNavigate`. When the board
 * itself is not navigable, still return it so the permission guard can send
 * the user to the Error page (contract §6).
 */
export function resolveDefaultLandPath(opts?: ResolveDefaultLandPathOptions): string {
  const board = normalizeAppPath(opts?.moduleBoardPath) || MODULE_BOARD_PATH;
  const leaf = opts?.menus
    ? findFirstNavigableMenuPath(opts.menus, opts?.canNavigate)
    : firstAllowedLeafPath(opts?.menuLeafPaths ?? [], opts?.canNavigate);
  return leaf || board;
}
