// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { onActivated, onMounted, toValue, type MaybeRefOrGetter } from 'vue';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

export type UseDetailPathRecordSetHygieneOptions = {
  /** When null/undefined, this host does not own Form pager sessions. */
  detailPath: MaybeRefOrGetter<unknown>;
  /** Collection model name; skips clear when active session is another model. */
  modelName: MaybeRefOrGetter<string | undefined | null>;
};

/**
 * Collection detailPath hosts (List / Kanban / future Calendar…): clear the
 * record-set session when returning to this surface so a stale pager does not
 * leak across KeepAlive remounts. Form (the detail target) must not call this.
 */
export function useDetailPathRecordSetHygiene(opts: UseDetailPathRecordSetHygieneOptions): {
  clearRecordSetSessionIfDetailHost: () => void;
} {
  function clearRecordSetSessionIfDetailHost(): void {
    if (toValue(opts.detailPath) == null) return;
    try {
      const rs = useRecordSetStore();
      const active = rs.active;
      if (!active) return;
      const model = String(toValue(opts.modelName) || '').trim();
      // Bail when model is not yet known — do not wipe another model's pager.
      if (!model) return;
      if (active.model && active.model !== model) return;
      rs.clear();
    } catch {
      // Pinia may be unavailable in unit mounts without an app shell.
    }
  }

  onMounted(clearRecordSetSessionIfDetailHost);
  onActivated(clearRecordSetSessionIfDetailHost);

  return { clearRecordSetSessionIfDetailHost };
}
