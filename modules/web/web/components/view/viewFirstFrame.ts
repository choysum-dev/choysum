// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared first-frame load contract for search-bound collection views
 * (List / Kanban / Chart).
 *
 * ChoySearchView emits exactly one mount-time `query-update` after UserFilter
 * defaults are ready. Hosts that mount it must not apply an empty query first,
 * or the board will double-fetch (empty apply + first-frame search).
 */

export type DeferToSearchFirstFrameOpts = {
  /** List: `:searchView` prop compared by identity to ChoySearchView. */
  searchView?: unknown;
  /** List: the ChoySearchView component reference closed over by the host. */
  canonicalSearchView?: unknown;
  /** Kanban/Chart: true when the `#search` slot is filled. */
  hasSearchSlot?: boolean;
};

/**
 * Whether the host should skip its own mount/bootstrap apply and wait for
 * Search's first-frame `query-update`.
 */
export function shouldDeferToSearchFirstFrame(opts: DeferToSearchFirstFrameOpts): boolean {
  if (opts.hasSearchSlot) return true;
  if (opts.searchView != null && opts.canonicalSearchView != null) {
    return opts.searchView === opts.canonicalSearchView;
  }
  return false;
}

/**
 * List-style helper: true only when `searchView` is the same component reference
 * as `canonicalSearchView` (typically ChoySearchView).
 */
export function shouldDeferViewFirstFrame(searchView: unknown, canonicalSearchView: unknown): boolean {
  return shouldDeferToSearchFirstFrame({ searchView, canonicalSearchView });
}

/**
 * Initial load for search-bound collection views.
 * - deferToSearch: no empty apply; first applySearch / onSearch owns the load
 *   (unless a search query already landed).
 * - otherwise: applyEmpty once, then optional afterEmpty (e.g. kanban lane sync).
 */
export async function finishInitialSearchBoundLoad(opts: {
  getLastSearchQuery?: () => unknown;
  applyEmpty: () => Promise<void>;
  afterEmpty?: () => Promise<void>;
  deferToSearch?: boolean;
}): Promise<void> {
  if (opts.getLastSearchQuery?.()) return;
  if (opts.deferToSearch) return;
  await opts.applyEmpty();
  if (opts.afterEmpty) await opts.afterEmpty();
}
