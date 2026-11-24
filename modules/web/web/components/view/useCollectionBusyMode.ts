// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from 'vue';
import type { ViewBusyMode } from './viewBusy';

export type UseCollectionBusyModeOptions = {
  /** True when the surface already has a content frame (rows / lanes / chart spec). */
  hasContent: MaybeRefOrGetter<boolean>;
  /** Force block overlay (e.g. first bootstrap with no frame). */
  forceBlock?: MaybeRefOrGetter<boolean>;
};

/**
 * Collection soft-stale heuristic: keep prior content readable while refreshing.
 * With content → `stale`; otherwise → `block`.
 */
export function useCollectionBusyMode(opts: UseCollectionBusyModeOptions): ComputedRef<ViewBusyMode> {
  return computed(() => {
    if (opts.forceBlock != null && toValue(opts.forceBlock)) return 'block';
    return toValue(opts.hasContent) ? 'stale' : 'block';
  });
}
