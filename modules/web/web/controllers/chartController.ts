// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive } from 'vue';
import type { GroupBySpec, QueryCondition } from '@/core/service/api/query';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { DataSetSnapshot } from '@/web/web/query/types';
import type { OrderByState } from '@/web/web/query/state';
import { buildUnifiedQuery } from '@/web/web/query/context';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { buildPlan } from '@/web/web/query/planner';
import { execute } from '@/web/web/query/executor';
import { createAbortableRequests, CancellationError, isCancellation } from '@/web/web/query/utils/abortable';

/**
 * View-model state exposed by the chart controller.
 */
export interface ChartControllerVM {
  loading: boolean;
  error: unknown | null;
  result: DataSetSnapshot | null;
}

/**
 * Chart controller contract used by chart views.
 */
export interface IChartController {
  vm: ChartControllerVM;
  apply(overrides?: {
    forcedCondition?: QueryCondition<any> | QueryCondition<any>[];
    appliedFilters?: any[];
    appliedGroups?: Array<GroupBySpec<any>>;
    // defaultGroups only fill the gap when appliedGroups are absent.
    defaultGroups?: Array<GroupBySpec<any>>;
    keyword?: string;
    keywordFields?: string[];
    orderBy?: OrderByState[];
  }): Promise<DataSetSnapshot | void>;
  sort(orderBy: OrderByState[]): Promise<void>;
  setForcedCondition(cond: QueryCondition<any> | QueryCondition<any>[] | undefined): Promise<void>;
  setAppliedGroups(groups: Array<GroupBySpec<any>> | undefined): Promise<void>;
  setKeyword(keyword: string | undefined): Promise<void>;
  setKeywordFields(fields: string[] | undefined): Promise<void>;
  refresh(): Promise<void>;
}

/**
 * Creates a chart controller backed by a web model store.
 */
export function createChartController(store: WebModelStore<any>): IChartController {
  const vm = reactive<ChartControllerVM>({ loading: false, error: null, result: null });
  const aborts = createAbortableRequests();
  let seq = 0;

  async function run(overrides?: {
    forcedCondition?: QueryCondition<any> | QueryCondition<any>[];
    appliedFilters?: any[];
    appliedGroups?: Array<GroupBySpec<any>>;
    defaultGroups?: Array<GroupBySpec<any>>;
    keyword?: string;
    keywordFields?: string[];
    orderBy?: OrderByState[];
  }): Promise<DataSetSnapshot | void> {
    const qs: any = (store.state as any)?.queryState || {};
    // Merge overrides into store.state.queryState (write only necessary fields)
    if (overrides?.appliedGroups) qs.appliedGroups = overrides.appliedGroups as any;
    if (overrides?.appliedFilters !== undefined) qs.appliedFilters = overrides.appliedFilters as any;
    if (overrides?.orderBy) qs.orderBy = overrides.orderBy as any;
    if (overrides?.keyword !== undefined) qs.keyword = overrides.keyword;
    if (overrides?.keywordFields !== undefined) qs.keywordFields = overrides.keywordFields;
    if (overrides?.forcedCondition !== undefined) qs.forcedCondition = overrides.forcedCondition as any;
    (store.state as any).queryState = qs;

    // Preserve user-selected groups before applying fallback defaults.
    const appliedGroups: Array<GroupBySpec<any>> = Array.isArray(qs.appliedGroups) ? (qs.appliedGroups as Array<GroupBySpec<any>>) : [];
    // Keep default groups transient so store state reflects explicit selections only.
    const effectiveGroups: Array<GroupBySpec<any>> =
      appliedGroups.length > 0 ? appliedGroups : overrides?.defaultGroups && overrides.defaultGroups.length > 0 ? overrides.defaultGroups : [];
    // Normalize grouping so granularity and aliases follow the shared query contract.
    const normalized = normalizeGroupby(effectiveGroups as any);
    const currentGroups: string[] = normalized.map(g => (g.granularity ? `${g.field}:${g.granularity}` : g.field)).filter(x => x && x.length > 0);
    const hasGroups = currentGroups.length > 0;
    if (!hasGroups) {
      // Return an empty grouped snapshot when no grouping is available.
      vm.result = { kind: 'group', rows: [], total: 0, ts: Date.now(), uiView: 'chart' } as any;
      return vm.result as DataSetSnapshot;
    }

    const ctx = buildUnifiedQuery(store, {
      groupby: currentGroups,
      orderBy: qs.orderBy as OrderByState[] | undefined,
      pagination: undefined,
      execOptions: { fullGroupHierarchy: true, skipPagination: true },
    });
    const bundle = buildPlan(ctx);

    const mySeq = ++seq;
    vm.loading = true;
    vm.error = null;
    try {
      const snap = await aborts.execute('chart.apply', async signal => {
        const r = await execute(bundle as any, store, 'chart', { signal });
        if (signal.aborted) throw new CancellationError();
        return r;
      });
      if (mySeq !== seq) throw new CancellationError('superseded');
      snap.uiView = 'chart';
      vm.result = snap;
      (store.state as any).result = snap;
      return snap; // DataSetSnapshot
    } catch (e) {
      if (isCancellation(e)) return; // swallow
      vm.error = e;
      throw e;
    } finally {
      if (mySeq === seq) vm.loading = false;
    }
  }

  async function apply(overrides?: Parameters<typeof run>[0]) {
    return run(overrides);
  }

  async function sort(orderBy: OrderByState[]) {
    await apply({ orderBy });
  }
  async function setForcedCondition(cond: QueryCondition<any> | QueryCondition<any>[] | undefined) {
    await apply({ forcedCondition: cond as any });
  }
  async function setAppliedGroups(groups: Array<GroupBySpec<any>> | undefined) {
    await apply({ appliedGroups: groups as any });
  }
  async function setKeyword(keyword: string | undefined) {
    await apply({ keyword });
  }
  async function setKeywordFields(fields: string[] | undefined) {
    await apply({ keywordFields: fields });
  }
  async function refresh() {
    await apply();
  }

  return {
    vm,
    apply,
    sort,
    setForcedCondition,
    setAppliedGroups,
    setKeyword,
    setKeywordFields,
    refresh,
  } as IChartController;
}

/**
 * Concrete chart controller type.
 */
export type ChartController = ReturnType<typeof createChartController>;
