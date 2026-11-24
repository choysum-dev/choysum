// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from 'vue';

export type UseViewTitleOverflowOptions = {
  /** Page IO menu payload from useHostPageTitleInView (truthy → show ⋮). */
  pageTitleActionsIo: MaybeRefOrGetter<unknown>;
  /** Caller-owned refresh visibility (ACL / display / header gates). */
  showRefresh: MaybeRefOrGetter<boolean>;
  /** Form create/edit Reset in ⋮; omit for collection views. */
  showReset?: MaybeRefOrGetter<boolean>;
  /** True when the product view provides `#overflow-actions`. */
  hasOverflowSlot: () => boolean;
  /** When false, hide the entire overflow chrome (Form settings). Default true. */
  enabled?: MaybeRefOrGetter<boolean>;
};

/**
 * Shared formula for ChoyViewBase `showTitleOverflow` + refresh gate.
 * Product views still compute `showRefresh` / `showReset` themselves.
 */
export function useViewTitleOverflow(opts: UseViewTitleOverflowOptions): {
  showRefreshInOverflow: ComputedRef<boolean>;
  showTitleOverflow: ComputedRef<boolean>;
} {
  const showRefreshInOverflow = computed(() => !!toValue(opts.showRefresh));
  const showTitleOverflow = computed(() => {
    if (opts.enabled != null && !toValue(opts.enabled)) return false;
    return (
      !!toValue(opts.pageTitleActionsIo) ||
      showRefreshInOverflow.value ||
      !!(opts.showReset != null && toValue(opts.showReset)) ||
      !!opts.hasOverflowSlot()
    );
  });
  return { showRefreshInOverflow, showTitleOverflow };
}
