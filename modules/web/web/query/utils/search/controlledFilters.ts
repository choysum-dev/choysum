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
 * Skips no-ops and stale echoes of our last emit while local has already moved ahead.
 */
export function shouldApplyControlledFilters(opts: {
  local: ConditionGroup[];
  incoming: ConditionGroup[] | undefined | null;
  lastEmittedSig: string;
}): { apply: boolean; normalized: ConditionGroup[] } {
  const normalized = normalizeFilters((opts.incoming || []) as any);
  const nextSig = filtersSignature(normalized);
  const localSig = filtersSignature(opts.local);

  if (nextSig === localSig) {
    return { apply: false, normalized };
  }

  // Parent echoed our last emit, but the user already changed local filters further.
  if (opts.lastEmittedSig && nextSig === opts.lastEmittedSig && localSig !== opts.lastEmittedSig) {
    return { apply: false, normalized };
  }

  return { apply: true, normalized };
}
