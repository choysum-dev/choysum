// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getCurrentInstance,
  onActivated,
  onDeactivated,
  toValue,
  type MaybeRefOrGetter,
} from 'vue';

export type UseDetailPathCollectionSoftRefreshOptions = {
  /** When null/undefined, this host is not a Form detailPath owner — no soft refresh. */
  detailPath: MaybeRefOrGetter<unknown>;
  /** True after the collection has applied at least once (skip empty first paint races). */
  hasSettled: () => boolean;
  /** Silent re-apply / engine.refresh; KeepAlive reactivate uses stale busy when content exists. */
  refresh: () => void | Promise<void>;
};

/**
 * DetailPath collection hosts (List / Kanban): soft-refresh rows when KeepAlive
 * reactivates after visiting Form (or another sibling). Instance cache ≠ data
 * freshness — without this, breadcrumb back shows pre-edit Search rows.
 *
 * Skips the first KeepAlive activate (same tick as mount bootstrap) via
 * onDeactivated → onActivated; relation pickers without detailPath are no-ops.
 */
export function useDetailPathCollectionSoftRefresh(
  opts: UseDetailPathCollectionSoftRefreshOptions
): {
  softRefreshIfDetailHost: () => void;
} {
  let wasDeactivated = false;

  function softRefreshIfDetailHost(): void {
    if (toValue(opts.detailPath) == null) return;
    if (!wasDeactivated) return;
    wasDeactivated = false;
    if (!opts.hasSettled()) return;
    try {
      void Promise.resolve(opts.refresh()).catch(() => undefined);
    } catch {
      // Host refresh may throw sync before returning a promise.
    }
  }

  if (getCurrentInstance()) {
    onDeactivated(() => {
      wasDeactivated = true;
    });
    onActivated(softRefreshIfDetailHost);
  }

  return { softRefreshIfDetailHost };
}
