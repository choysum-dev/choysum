// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ConditionGroup } from '@/web/web/query/types';
import { normalizeFilters } from '@/web/web/query/utils/filter/structures';

/** Stable content fingerprint for controlled filter sync (ignores object identity). */
export function filtersSignature(list: ConditionGroup[] | undefined | null): string {
  if (!list || !list.length) return '';
  try {
    return JSON.stringify(list);
  } catch {
    return `len:${list.length}`;
  }
}

/**
 * Decide whether props.currentAppliedFilters should overwrite local filter tags.
 * Skips no-ops, stale echoes of our last emit while local has moved ahead, and
 * lagging parent snapshots while we are still awaiting acknowledgment of that emit.
 */
export function shouldApplyControlledFilters(opts: {
  local: ConditionGroup[];
  incoming: ConditionGroup[] | undefined | null;
  lastEmittedSig: string;
  awaitingEcho?: boolean;
}): { apply: boolean; normalized: ConditionGroup[]; acknowledged: boolean } {
  const normalized = normalizeFilters((opts.incoming || []) as any);
  const nextSig = filtersSignature(normalized);
  const localSig = filtersSignature(opts.local);

  if (nextSig === localSig) {
    // Parent caught up to local (including an empty-filter echo).
    return {
      apply: false,
      normalized,
      acknowledged: !!opts.awaitingEcho && nextSig === opts.lastEmittedSig,
    };
  }

  // Parent echoed our last emit, but the user already changed local filters further.
  if (opts.lastEmittedSig && nextSig === opts.lastEmittedSig && localSig !== opts.lastEmittedSig) {
    return { apply: false, normalized, acknowledged: true };
  }

  // Still waiting for parent to acknowledge our last emit — ignore lagging snapshots
  // (older or empty) that would otherwise clobber the filters we just emitted.
  if (opts.awaitingEcho && nextSig !== opts.lastEmittedSig) {
    return { apply: false, normalized, acknowledged: false };
  }

  return { apply: true, normalized, acknowledged: false };
}
