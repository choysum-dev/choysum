// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, reactive, ref, watch } from 'vue';
import type { ComputedRef, Ref } from 'vue';
import type { QueryCondition } from '@/core/service/api/query';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createListController } from '@/web/web/controllers/listController';
import type { RecordRow, Lane, IKanbanController, TemporalGranularity, QueryKind, GroupBySpec, ConditionGroup } from '@/web/web/query/types';
import type { PaginationState, OrderByState } from '@/web/web/query/state';
import { exportMetrics } from '@/web/web/query/utils/registry/metric';
import { exportFieldSelection, pathsToFieldSelection, ensureRootId } from '@/web/web/query/utils/registry/field';
import { buildUnifiedQuery } from '@/web/web/query/context';
import { stableStringify } from '@/web/web/query/planner';
import { handoffCache } from '@/web/web/query/utils/handoff';
import { asPresentCondition } from '@/web/web/query/utils/condition/absent';

// Lane & IKanbanController centralized in query/types.ts

/**
 * Creates a kanban controller backed by the shared list controller.
 */
export function createKanbanController(store: WebModelStore<any>): IKanbanController {
  const list = createListController(store);
  const laneRecords = ref<Record<string, RecordRow[]>>({});

  // Keyword-field fallback inference now lives in filtersToQuery.

  const lanes = computed<Lane[]>(() => {
    const r = list.vm.result;
    if (!r || r.kind !== 'group') return [];
    const rows = (r.rows as any[]).filter(x => x?.kind === 'group' && Number(x.depth ?? 0) === 0);
    return rows.map(g => {
      // Prefer the first label alias returned by the backend when available.
      let label = String(g.label ?? '');
      try {
        const raw = (g as any).raw;
        if (raw && raw.labels && typeof raw.labels === 'object') {
          const aliases = Object.keys(raw.labels).sort();
          const first = aliases[0];
          if (first && raw.labels[first]) label = String(raw.labels[first]);
        }
      } catch {}
      const lane: Lane = {
        ...g,
        key: String(g.key),
        label,
        count: (g as any).count ?? (g as any).__count,
        condition: (g as any).__condition,
        raw: g,
      };
      return lane;
    });
  });

  function getLaneField(): string | null {
    const gb: string[] = ((store.state as any)?.queryState?.appliedGroups as any) ?? [];
    if (!gb || gb.length === 0) return null;
    const raw = String(gb[0] || '');
    const base = raw.split(':')[0];
    return base || null;
  }

  function getLaneValue(lane: Lane): any {
    const field = getLaneField();
    if (!field) return lane.key;
    const cond = lane.condition;
    const findEq = (c: any): any => {
      if (!c) return undefined;
      if (Array.isArray(c)) {
        if (c.length >= 3 && c[0] === field && (c[1] === '=' || c[1] === '==' || c[1] === 'eq')) return c[2];
        for (const it of c) {
          const r = findEq(it);
          if (r !== undefined) return r;
        }
        return undefined;
      }
      if (typeof c === 'object') {
        if (Array.isArray(c.And)) {
          for (const it of c.And) {
            const r = findEq(it);
            if (r !== undefined) return r;
          }
        }
        if (Array.isArray((c as any).Or)) {
          for (const it of (c as any).Or) {
            const r = findEq(it);
            if (r !== undefined) return r;
          }
        }
        return undefined;
      }
      return undefined;
    };
    const val = findEq(cond);
    if (val !== undefined) return val;
    const m = String(lane.key).match(/^([^=]+)=(.+?)(?:#\d+)?$/);
    if (m && m[1] === field) return m[2] === '__null__' ? null : m[2];
    return lane.key;
  }

  function getLaneRemain(lane: Lane): number {
    const loaded = (laneRecords.value[lane.key] || []).length;
    const total = Number(lane.count ?? 0);
    return Math.max(0, total - loaded);
  }

  async function preloadLane(key: string): Promise<void> {
    if (laneRecords.value[key]) return;
    const rows = await (list as any).fetchGroupRecords?.(key, (store.state as any)?.queryState?.pagination, { skipCount: true });
    const arr = (rows || list.vm.result?.groupRecords?.get?.(key) || []) as RecordRow[];
    laneRecords.value[key] = Array.isArray(arr) ? arr.slice() : [];
  }

  async function loadMoreLane(key: string): Promise<void> {
    await (list as any).loadMoreGroupRecords?.(key);
    await (list as any).loadMoreGroupRecords?.(key);
    const arr = (list.vm.result?.groupRecords?.get?.(key) as any) || [];
    laneRecords.value[key] = Array.isArray(arr) ? arr.slice() : [];
  }

  async function moveCard(cardId: string, _from: string, to: string, _index: number): Promise<void> {
    // _from is the source lane key and to is the target lane key.
    const field = getLaneField();
    if (!field) return;
    if (!_from || !to || _from === to) return;
    const toLane = lanes.value.find(l => l.key === to);
    const fromLane = lanes.value.find(l => l.key === _from);
    if (!toLane) return;
    const toValue = getLaneValue(toLane);

    // Update the lane field through the backend and request the current view fields.
    const rawPaths = exportFieldSelection(store.storeId);
    const returnFields = ensureRootId(pathsToFieldSelection(rawPaths));
    const updatedRecord = await store.UpdateById?.(String(cardId), { [field]: toValue } as any, returnFields);

    if (updatedRecord && typeof updatedRecord === 'object') {
      const id = (updatedRecord as any).Id || (updatedRecord as any).id || cardId;
      handoffCache.set(String(id), updatedRecord);
    }

    // Optimistically update grouped counts for the source and target lanes.
    try {
      const snap = list.vm.result;
      if (snap && snap.kind === 'group') {
        const rows: any[] = (snap.rows as any[]) || [];
        const fr = rows.find(g => String(g.key) === String(_from));
        const tr = rows.find(g => String(g.key) === String(to));
        if (fr && typeof fr.count === 'number' && fr.count > 0) fr.count = Number(fr.count) - 1;
        if (tr) tr.count = Number(tr.count ?? 0) + 1;
      }
    } catch {}

    // Update cached lane records so the board does not need an immediate reload.
    try {
      // 1. Remove the card from the source lane if it is still present.
      if (laneRecords.value[_from]) {
        const idx = laneRecords.value[_from].findIndex(r => r.payload?.Id === cardId || r.key === cardId);
        if (idx > -1) laneRecords.value[_from].splice(idx, 1);
      }

      // 2. Update or insert the card in the target lane.
      if (laneRecords.value[to]) {
        const targetList = laneRecords.value[to];

        // vuedraggable may already have moved the element into the target array.
        const existingIdx = targetList.findIndex(r => r.payload?.Id === cardId || r.key === cardId);

        // Prefer the backend record, otherwise patch the existing payload locally.
        let finalPayload = updatedRecord && typeof updatedRecord === 'object' ? updatedRecord : null;

        if (existingIdx > -1) {
          // Case A: the dragged card is already in the target lane.
          const row = targetList[existingIdx];
          if (!finalPayload) {
            // Fall back to patching the existing payload with the new lane value.
            finalPayload = { ...row.payload, [field]: toValue };
          }
          row.payload = finalPayload;
        } else {
          // Case B: insert a new row when the card is not already in the target lane.
          if (!finalPayload) {
            finalPayload = { Id: cardId, [field]: toValue };
          }
          const newRow: RecordRow = { kind: 'record', key: String((finalPayload as any).Id || cardId), payload: finalPayload };
          const safeIndex = Math.min(Math.max(0, _index), targetList.length);
          targetList.splice(safeIndex, 0, newRow);
        }
      }
    } catch {}

    // Refresh source and target lane aggregates after the drag finishes.
    try {
      await refreshLaneAggregates([_from, to]);
    } catch {}
  }

  // Refresh aggregate fields for specific lanes without clearing laneRecords.
  async function refreshLaneAggregates(keys: string[]): Promise<void> {
    const snapshot = list.vm.result;
    if (!snapshot || snapshot.kind !== 'group') return;
    const gbArr: string[] = ((store.state as any)?.queryState?.appliedGroups as any) ?? [];
    if (!gbArr.length) return;
    const firstGbRaw = String(gbArr[0] || '');
    const firstGbBase = firstGbRaw.split(':')[0];
    if (!firstGbBase) return;

    // Reuse the metric selection contract used by query plan building.
    const metricSpecs: Array<{ field: string; agg: string; alias?: string }> = exportMetrics(store.storeId) || [];
    const fields = metricSpecs.map(m => ({ field: m.field, agg: m.agg, alias: m.alias })) as any[];

    const AGG_SUFFIX = ['__sum', '__avg', '__min', '__max', '__count', '__count_distinct'];

    // Default aggregate values for empty result sets.
    const defaultForSuffix = (suffix: string): any => {
      switch (suffix) {
        case '__sum':
        case '__count':
          return 0;
        case '__avg':
        case '__min':
        case '__max':
        case '__count_distinct':
        default:
          return null;
      }
    };

    // Batch-refresh by combining all target lane conditions.
    const uniqueKeys = Array.from(new Set(keys.filter(Boolean)));
    const targetLanes = uniqueKeys.map(k => lanes.value.find(l => l.key === k)).filter(Boolean) as Lane[];
    if (!targetLanes.length) return;

    // Combine lane conditions with OR. A lane without a condition naturally means "all".
    const combinedCondition =
      targetLanes.length === 1 ? targetLanes[0].condition : { Or: targetLanes.map(l => l.condition).filter(c => asPresentCondition(c) !== undefined) };

    // Let the unified query builder merge global and lane-specific conditions.
    const ctx = buildUnifiedQuery(store, {
      groupby: [firstGbBase],
      pagination: { limit: 1000, offset: 0 },
      parentCondition: combinedCondition,
      execOptions: { skipCount: true },
    });

    // Record the plan signature using the same scheme as list views.
    try {
      const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
      const now = Date.now();
      const hit = (store.state as any).planCache.get(key);
      if (hit) {
        hit.hit += 1;
        hit.lastUsed = now;
      } else {
        (store.state as any).planCache.set(key, { signature: key, kind: 'readGroup', hit: 0, lastUsed: now, createdAt: now });
      }
    } catch {}

    try {
      // Use the shared executor path so aggregation matches list-view behavior.
      const { buildPlan } = await import('@/web/web/query/planner');
      const { execute } = await import('@/web/web/query/executor');
      const bundle = buildPlan(ctx as any);
      const snap: any = await execute(bundle, store, 'kanban-lane-agg');
      let groups: any[] = [];
      if (snap?.kind === 'group') {
        groups = [];
        for (const rr of snap.rows as any[]) groups.push((rr as any)?.raw);
      }

      // Apply refreshed aggregate values back to each target lane.
      for (const lane of targetLanes) {
        // Find the existing group row for the lane.
        const target = (snapshot.rows as any[]).find(r => r.kind === 'group' && String(r.key) === String(lane.key));
        if (!target) continue;

        // Match the refreshed group using either the emitted key or grouped values.
        const gRaw = groups.find(g => {
          if (String(g.key) === String(lane.key)) return true;
          if (g.keys && g.keys[firstGbBase] == getLaneValue(lane)) return true;
          return false;
        });

        if (!gRaw) {
          // Missing group results mean the lane currently has no matching records.
          target.count = 0;
          for (const prop of Object.keys(target)) {
            const matched = AGG_SUFFIX.find(s => prop.endsWith(s));
            if (matched) {
              (target as any)[prop] = defaultForSuffix(matched);
              target.metrics = { ...(target.metrics || {}), [prop]: (target as any)[prop] };
            }
          }

          // Clear any stale metric cache entries as well.
          if (target.metrics) {
            for (const mk of Object.keys(target.metrics)) {
              const matched = AGG_SUFFIX.find(s => mk.endsWith(s));
              if (matched) target.metrics[mk] = defaultForSuffix(matched);
            }
          }
          continue;
        }

        if (typeof gRaw.count === 'number') target.count = gRaw.count;
        else if (typeof gRaw.__count === 'number') target.count = gRaw.__count;

        // Prefer aggregates stored in the metrics map when present.
        if (gRaw.metrics && typeof gRaw.metrics === 'object') {
          for (const [k, v] of Object.entries(gRaw.metrics)) {
            if (AGG_SUFFIX.some(s => k.endsWith(s))) {
              (target as any)[k] = v;
              target.metrics = { ...(target.metrics || {}), [k]: v };
            }
          }
        }

        for (const prop of Object.keys(gRaw)) {
          if (AGG_SUFFIX.some(s => prop.endsWith(s))) {
            (target as any)[prop] = gRaw[prop];
            target.metrics = { ...(target.metrics || {}), [prop]: gRaw[prop] };
          }
        }
        target.raw = gRaw;
      }
    } catch {
      // Silent failure keeps drag-and-drop responsive.
    }
  }

  // Wrap list ops to clear lane cache accordingly
  async function apply(overrides?: {
    forcedCondition?: QueryCondition<any>;
    appliedFilters?: ConditionGroup[];
    keyword?: string;
    keywordFields?: string[];
    orderBy?: OrderByState[];
    pagination?: PaginationState;
    appliedGroups?: Array<GroupBySpec>;
    kind?: QueryKind;
  }) {
    const r = await list.apply(overrides);
    laneRecords.value = {};
    return r;
  }
  async function paginate(p: PaginationState) {
    await list.paginate(p);
    laneRecords.value = {};
  }
  async function setAppliedGroups(gb: Array<GroupBySpec>) {
    await list.setAppliedGroups(gb as any);
    laneRecords.value = {};
  }
  async function setKeyword(keyword: string | undefined) {
    await list.setKeyword(keyword);
    laneRecords.value = {};
  }
  async function setKeywordFields(keywordFields: string[]) {
    await list.setKeywordFields(keywordFields);
    laneRecords.value = {};
  }
  async function setForcedCondition(filters: QueryCondition<any> | QueryCondition<any>[]) {
    await list.setForcedCondition(filters as any);
    laneRecords.value = {};
  }
  async function setAppliedFilters(filters: ConditionGroup[]) {
    await list.setAppliedFilters(filters as any);
    laneRecords.value = {};
  }
  async function sort(orderBy: OrderByState[]) {
    await list.sort(orderBy);
    laneRecords.value = {};
  }

  return {
    vm: list.vm,
    lanes,
    laneRecords,
    apply,
    paginate,
    setAppliedGroups,
    setKeyword,
    setKeywordFields,
    setForcedCondition,
    setAppliedFilters,
    sort,
    getLaneField,
    getLaneValue,
    getLaneRemain,
    preloadLane,
    loadMoreLane,
    moveCard,
    refreshLaneAggregates,
  } as IKanbanController;
}
