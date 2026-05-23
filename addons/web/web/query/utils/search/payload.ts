// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Pure helper that builds QueryUpdatePayload values without UI dependencies.
import type { ConditionGroup, GroupBySpec, QueryUpdatePayload } from '@/web/web/query/types';

export function buildQueryUpdatePayload<T = any>(
  keyword: string | undefined,
  appliedFilters: ConditionGroup[] | undefined,
  appliedGroups: Array<GroupBySpec<T>> | undefined,
  options?: { explicitGroups?: boolean }
): QueryUpdatePayload<T> {
  const kw = keyword?.trim() || undefined;
  const filtersArr: ConditionGroup[] = Array.isArray(appliedFilters) ? appliedFilters : [];
  const specs = Array.isArray(appliedGroups) ? appliedGroups : [];
  const explicit = options?.explicitGroups === true;
  return {
    keyword: kw,
    appliedFilters: filtersArr,
    appliedGroups: explicit ? specs : specs.length ? specs : undefined,
  } as QueryUpdatePayload<T>;
}
