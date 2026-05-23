// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Central helpers for computing initial search and grouping state.
// These helpers decide when default filters or groups should be injected on first render.
// They work with the controller-level userClearedDefaultFilters and userClearedDefaultGroups flags.

import type { NamedFilter, ConditionGroup, Condition } from '@/web/web/query/types';
import type { QueryCondition } from '@/core/service/api/query';
import { toFilters, normalizeFilters } from '@/web/web/query/utils/filter/structures';

// Flag readers.
export function userClearedDefaultFilters(qs: any): boolean {
  return !!qs?.userClearedDefaultFilters;
}
export function userClearedDefaultGroups(qs: any): boolean {
  return !!qs?.userClearedDefaultGroups;
}

// Default-filter conversion helpers.
// Converts selected named defaults into ConditionGroup[] values used by the tag UI.
export function buildSelectedDefaultFilterGroups(defs: NamedFilter[] | undefined): ConditionGroup[] {
  const list = Array.isArray(defs) ? defs : [];
  const selected = list.filter(d => d && (d as any).selected === true);
  const out: ConditionGroup[] = [];
  for (const nf of selected) {
    const raw = (nf as any).query as QueryCondition<any> | ConditionGroup | undefined;
    if (!raw) continue;
    if (typeof raw === 'object' && !Array.isArray(raw) && (raw as any).logic && Array.isArray((raw as any).children)) {
      out.push({ ...(raw as any), name: nf.name } as ConditionGroup);
      continue;
    }
    if (Array.isArray(raw) && raw.length >= 3) {
      const [field, operator, value] = raw as any[];
      const cond: Condition = {
        id: `cond_${Math.random().toString(36).slice(2, 10)}`,
        field: String(field),
        operator: String(operator),
        value,
      };
      out.push({
        id: `group_${Math.random().toString(36).slice(2, 10)}`,
        logic: 'And',
        children: [cond],
        name: nf.name,
      });
      continue;
    }
    const f = (normalizeFilters(toFilters(nf as any)) || [])[0];
    if (f) out.push({ ...(f as any), name: nf.name } as ConditionGroup);
  }
  return out;
}

// Initial filter computation.
// Returns the applied filter array exposed to child components before the first mount.
export function computeInitialAppliedFilters(opts: {
  qs: any;
  mounted: boolean;
  initialEmit: boolean;
  explicitFilters?: ConditionGroup[];
  defaultFilters?: NamedFilter[];
}): ConditionGroup[] {
  const { qs, mounted, initialEmit, explicitFilters, defaultFilters } = opts;
  if (Array.isArray(explicitFilters)) return explicitFilters;
  const hasDefined = qs && Object.prototype.hasOwnProperty.call(qs, 'appliedFilters');
  if (hasDefined) {
    const arr = Array.isArray(qs.appliedFilters) ? (qs.appliedFilters as ConditionGroup[]) : [];
    if (!mounted && initialEmit && arr.length === 0 && !userClearedDefaultFilters(qs)) {
      return buildSelectedDefaultFilterGroups(defaultFilters);
    }
    return arr;
  }
  if (userClearedDefaultFilters(qs)) return [];
  return buildSelectedDefaultFilterGroups(defaultFilters);
}

// Applied-group computation.
export function computeAppliedGroups(
  qs: any,
  explicitGroups?: Array<any>,
  opts?: { mounted: boolean; initialEmit: boolean; defaultGroups?: Array<any> }
): Array<any> {
  // Controlled values always win when the caller provides them explicitly.
  if (Array.isArray(explicitGroups)) return explicitGroups;

  const gb = qs?.appliedGroups;
  const hasDefined = qs && Object.prototype.hasOwnProperty.call(qs, 'appliedGroups');
  const arr: Array<any> = Array.isArray(gb) ? (gb as any[]) : gb ? [gb] : [];

  // Inject default groups exactly once on first render when no groups already exist.
  if (hasDefined && !opts?.mounted && opts?.initialEmit === true && arr.length === 0 && !userClearedDefaultGroups(qs)) {
    const defs = Array.isArray(opts?.defaultGroups) ? (opts!.defaultGroups as any[]) : [];
    if (defs.length > 0) return defs;
  }

  if (!hasDefined) {
    if (userClearedDefaultGroups(qs)) return [];
    const defs = Array.isArray(opts?.defaultGroups) ? (opts!.defaultGroups as any[]) : [];
    if (!opts?.mounted && opts?.initialEmit === true && defs.length > 0) return defs;
  }

  return arr;
}
