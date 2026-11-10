// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { toValue, type MaybeRefOrGetter } from 'vue';
import type { SearchFirstFrameGate } from './search/searchFirstFrame';

/**
 * Shared first-frame load helpers for search-bound collection views
 * (List / Kanban / Chart). Prefer the Search first-frame gate from
 * `useSearchFirstFrameHost` / `useSearchFirstFrameReporter`; fallbacks cover
 * hosts that have not joined the protocol yet.
 */

export type DeferToSearchFirstFrameOpts = {
  /** Preferred: gate from useSearchFirstFrameHost. */
  gate?: SearchFirstFrameGate | null;
  /** List fallback: `:searchView` prop compared by identity to ChoySearchView. */
  searchView?: unknown;
  /** List fallback: the ChoySearchView component reference. */
  canonicalSearchView?: unknown;
  /** Kanban/Chart fallback: true when the `#search` slot is filled. */
  hasSearchSlot?: boolean;
};

/**
 * Whether the host should skip its own mount/bootstrap apply and wait for
 * Search's first-frame `query-update`.
 */
export function shouldDeferToSearchFirstFrame(opts: DeferToSearchFirstFrameOpts): boolean {
  if (opts.gate) return opts.gate.shouldDefer.value;
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
 * - deferToSearch / gate.shouldDefer: no empty apply; first applySearch owns the load
 *   (unless a search query already landed).
 * - otherwise: applyEmpty once, then optional afterEmpty (e.g. kanban lane sync).
 */
export async function finishInitialSearchBoundLoad(opts: {
  getLastSearchQuery?: () => unknown;
  applyEmpty: () => Promise<void>;
  afterEmpty?: () => Promise<void>;
  deferToSearch?: MaybeRefOrGetter<boolean>;
  gate?: SearchFirstFrameGate | null;
}): Promise<void> {
  if (opts.getLastSearchQuery?.()) return;
  const defer = opts.gate
    ? opts.gate.shouldDefer.value
    : !!toValue(opts.deferToSearch);
  if (defer) return;
  await opts.applyEmpty();
  if (opts.afterEmpty) await opts.afterEmpty();
}
