// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive } from 'vue';
import type { QueryCondition } from '@/core/service/api/query';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { PaginationState, OrderByState } from '@/web/web/query/state';
import type { DataSetSnapshot, GroupRow, RecordRow, ListViewModel, GroupBySpec, QueryKind, ConditionGroup } from '@/web/web/query/types';
import { buildSearchOrGroupContext, buildUnifiedQuery } from '@/web/web/query/context';
import { buildPlan, stableStringify } from '@/web/web/query/planner';
import { execute } from '@/web/web/query/executor';
import { createAbortableRequests, isCancellation, CancellationError } from '@/web/web/query/utils/abortable';
import type { NamedFilter, NamedGrouping, IListController } from '@/web/web/query/types';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { userClearedDefaultFilters, userClearedDefaultGroups } from '@/web/web/query/utils/search/initialQueryState';
import { exportFieldSelection } from '@/web/web/query/utils/registry/field';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { asPresentCondition } from '@/web/web/query/utils/condition/absent';

// ListViewModel & GroupBySpec now centralized in query/types.ts

/**
 * Creates a list controller for flat and grouped list views.
 */
export function createListController(store: WebModelStore<any>): IListController {
  const vm = reactive<ListViewModel>({
    loading: false,
    error: null,
    result: null,
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  });

  // Abort management (last-one-wins)
  const aborts = createAbortableRequests();
  let applySeq = 0;

  // Caches for on-demand tree expansion
  // Children groups under a group key
  const groupChildren = new Map<string, GroupRow[]>();
  // Leaf records under a group key
  const groupRecords = new Map<string, RecordRow[]>();
  // Remaining count for leaf records (for potential "more" rows)
  const groupRecordRemain = new Map<string, number>();
  // Remaining count for child groups (for group-level "more" rows)
  const groupChildrenRemain = new Map<string, number>();

  // Normalize group-by input to string[] so it matches queryState storage.
  function normalizeGroupbyToStrings(input?: Array<string | GroupBySpec> | null): string[] {
    if (!input) return [];
    return input
      .map(g => {
        if (typeof g === 'string') return g;
        const f = g.field?.trim();
        if (!f) return '';
        const gran = (g.granularity || '').trim();
        return gran ? `${f}:${gran}` : f;
      })
      .filter(Boolean);
  }

  function currentGroupbyFromStore(): string[] {
    const g = ((store.state as any)?.queryState?.appliedGroups as Array<string | GroupBySpec> | undefined) ?? [];
    return normalizeGroupbyToStrings(g);
  }

  // Bootstrap defaults previously injected by view (moved here)
  function toArray<V>(x?: V | V[]): V[] {
    return x == null ? [] : Array.isArray(x) ? x : [x];
  }
  function bootstrapDefaults(defaultFilter?: NamedFilter[] | NamedFilter | undefined, defaultGroups?: NamedGrouping[] | NamedGrouping | undefined) {
    // Default filters: only when initially empty
    const dfArr = toArray(defaultFilter);
    const qs0: any = (store.state as any).queryState || {};
    const wasEmptyDefaultFilters = (qs0.defaultFilters?.length ?? 0) === 0;
    if (!userClearedDefaultFilters(qs0) && wasEmptyDefaultFilters && dfArr.length > 0) {
      const next = dfArr.filter((f: any) => !!f?.name).slice();
      (store.state as any).queryState = {
        ...(store.state as any).queryState,
        defaultFilters: next,
      };
    }

    // Inject default groups only once while appliedGroups are still empty.
    const incoming = toArray(defaultGroups);
    const qs = (store.state as any).queryState || {};
    if (!userClearedDefaultGroups(qs) && (qs.appliedGroups?.length ?? 0) === 0 && incoming.length > 0) {
      const flat = incoming.flatMap(g => toArray((g as any)?.groupby));
      const normalized = normalizeGroupby(flat as any);
      (store.state as any).queryState = {
        ...qs,
        appliedGroups: normalized,
        defaultGroups: incoming.slice(),
        kind: 'group',
        pagination: qs.pagination ?? { limit: 100, offset: 0 },
      };
    } else if (!userClearedDefaultGroups(qs) && incoming.length > 0) {
      (store.state as any).queryState = {
        ...qs,
        defaultGroups: incoming.slice(),
      };
    }
  }

  function recomputeVisible() {
    const r = vm.result;
    if (!r) {
      vm.visibleNodes = [];
      return;
    }
    // Flatten roots + expanded descendants
    if (r.kind !== 'group') {
      vm.visibleNodes = r.rows as any[];
      return;
    }

    const gb = normalizeGroupbyToStrings(((store.state as any)?.queryState?.appliedGroups as Array<string | GroupBySpec>) ?? []);
    const out: Array<GroupRow | RecordRow> = [];

    function pushNode(node: GroupRow) {
      out.push(node);
      const d = Number(node.depth ?? 0);
      const isLeaf = gb.length > 0 ? d >= gb.length - 1 : true;
      const expanded = vm.expandedKeys.has(node.key);
      if (!expanded) return;
      if (!isLeaf) {
        const children = groupChildren.get(node.key) || [];
        for (const ch of children) pushNode(ch);
        const remainGroups = Math.max(0, Number(groupChildrenRemain.get(node.key) || 0));
        if (remainGroups > 0) {
          out.push({ kind: 'more', key: `more-g:${node.key}`, groupKey: node.key, remain: remainGroups, target: 'groups' } as any);
        }
      } else {
        const recs = groupRecords.get(node.key) || [];
        for (const rr of recs) out.push(rr);
        const remainCount = Math.max(0, Number(groupRecordRemain.get(node.key) || 0));
        if (remainCount > 0) {
          out.push({ kind: 'more', key: `more:${node.key}`, groupKey: node.key, remain: remainCount, target: 'records' } as any);
        }
      }
    }

    const roots = (r.rows as GroupRow[]) || [];
    roots.forEach(g => pushNode(g));
    vm.visibleNodes = out as any[];
  }

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
    const seq = ++applySeq;
    vm.loading = true;
    vm.error = null;
    try {
      // Wait briefly for field registration so the first request has field selections.
      try {
        if (!exportFieldSelection((store as any).storeId)?.length) {
          await awaitFieldSelection(store as any, { maxTries: 5, requireNonEmpty: false });
        }
      } catch {}

      // Make the target shape explicit when overrides provide grouping or kind hints.
      const gbStrings = overrides?.appliedGroups != null ? normalizeGroupbyToStrings(overrides.appliedGroups) : undefined;
      let shape: 'collection' | 'groups' | undefined = undefined;
      const incomingKind = overrides?.kind;
      if (!shape && incomingKind) {
        shape = incomingKind === 'group' ? 'groups' : 'collection';
      }
      if (!shape && gbStrings != null) {
        shape = gbStrings.length > 0 ? 'groups' : 'collection';
      }

      // Only inherit existing search state when callers did not explicitly override it.
      const effectiveOverrides = { ...overrides } as typeof overrides;
      const hasKeywordOverride = overrides && Object.prototype.hasOwnProperty.call(overrides, 'keyword');
      const hasForcedConditionOverride = overrides && Object.prototype.hasOwnProperty.call(overrides, 'forcedCondition');
      const hasAppliedFiltersOverride = overrides && Object.prototype.hasOwnProperty.call(overrides, 'appliedFilters');
      const existingKw = (store.state as any)?.queryState?.keyword;
      // queryState.appliedFilters stores the UI condition tree.
      const existingUiFilters = (store.state as any)?.queryState?.appliedFilters;
      const prevFiltersLen = Array.isArray(existingUiFilters) ? existingUiFilters.length : 0;
      const existingGroups = (store.state as any)?.queryState?.appliedGroups;
      const prevGroupsLen = Array.isArray(existingGroups) ? existingGroups.length : 0;
      if (!hasKeywordOverride && existingKw !== undefined) {
        (effectiveOverrides as any).keyword = existingKw;
      }
      if (!hasAppliedFiltersOverride && existingUiFilters !== undefined) {
        (effectiveOverrides as any).appliedFilters = existingUiFilters;
      }

      // Compile UI filters and keyword search, then merge forced conditions.
      const uiFiltersEff: any[] | undefined = (effectiveOverrides as any).appliedFilters;
      const kwEff: string | undefined = (effectiveOverrides as any).keyword;
      let kwFieldsEff: string[] | undefined = (effectiveOverrides as any).keywordFields ?? (store.state as any)?.queryState?.keywordFields;
      let compiledFromUi = filtersToQuery(Array.isArray(uiFiltersEff) ? uiFiltersEff : [], kwEff, kwFieldsEff, (store as any)?.fieldsMetadata);

      // Merge forced conditions stored in queryState first.
      const forcedInState = (store.state as any)?.queryState?.forcedCondition;
      if (forcedInState !== undefined) {
        compiledFromUi = combineFilters(compiledFromUi, forcedInState);
      }

      // Merge the call-site forced condition last.
      const forcedCondition = hasForcedConditionOverride ? (effectiveOverrides as any).forcedCondition : undefined;
      const finalCondition = (() => {
        const A = normalizeFilter(compiledFromUi);
        const B = normalizeFilter(forcedCondition);
        if (!A && !B) return undefined;
        if (!A) return B;
        if (!B) return A;
        return { And: [A, B] } as any;
      })();

      // Persist keyword, keywordFields, and UI filters so cross-view restores remain stable.
      if (effectiveOverrides) {
        // Explicit keyword overrides may also clear a previous keyword value.
        if (hasKeywordOverride) {
          (store.state as any).queryState = {
            ...(store.state as any).queryState,
            keyword: (effectiveOverrides as any).keyword,
          };
        }
        if (effectiveOverrides.keywordFields !== undefined) {
          (store.state as any).queryState = {
            ...(store.state as any).queryState,
            keywordFields: effectiveOverrides.keywordFields,
          };
        }

        // Keep appliedFilters as the single source of truth for the UI condition tree.
        if (uiFiltersEff !== undefined) {
          (store.state as any).queryState = {
            ...(store.state as any).queryState,
            appliedFilters: uiFiltersEff as any,
          };

          // Remember explicit clears so default filters are not re-injected later.
          if (prevFiltersLen > 0 && Array.isArray(uiFiltersEff) && uiFiltersEff.length === 0) {
            (store.state as any).queryState = {
              ...(store.state as any).queryState,
              userClearedDefaultFilters: true,
            };
          }
        }
      }

      const ctx = buildSearchOrGroupContext(store, {
        shape,
        filters: finalCondition,
        queryState:
          gbStrings && gbStrings.length > 0
            ? {
                appliedGroups: gbStrings,
                orderBy: effectiveOverrides?.orderBy,
                pagination: effectiveOverrides?.pagination as any,
              }
            : {
                orderBy: effectiveOverrides?.orderBy,
                pagination: effectiveOverrides?.pagination as any,
              },
      });

      // Keep controlled pagination in sync with store state immediately.
      if (effectiveOverrides?.pagination) {
        const p = effectiveOverrides.pagination;
        (store.state as any).queryState = {
          ...(store.state as any).queryState,
          pagination: { ...p },
        };
      }

      // Write applied groups back so the search header reflects controlled state.
      if (overrides?.appliedGroups !== undefined) {
        const gb = gbStrings ?? [];
        if (gb.length > 0) {
          (store.state as any).queryState = {
            ...(store.state as any).queryState,
            kind: 'group',
            appliedGroups: gb,
          };
        } else {
          // Explicit clears must also clear stale grouped header state.
          (store.state as any).queryState = {
            ...(store.state as any).queryState,
            kind: 'search',
            appliedGroups: [],
          };

          // Collapse expanded groups when leaving grouped mode.
          vm.expandedKeys.clear();
          if (prevGroupsLen > 0 && !userClearedDefaultGroups((store.state as any).queryState)) {
            (store.state as any).queryState = {
              ...(store.state as any).queryState,
              userClearedDefaultGroups: true,
            };
          }
        }
      } else if (ctx.shape === 'collection') {
        // Collection mode should also clear stale grouped header state.
        (store.state as any).queryState = {
          ...(store.state as any).queryState,
          kind: 'search',
          appliedGroups: [],
        };
        vm.expandedKeys.clear();
      }
      const bundle = buildPlan(ctx);

      // Record the stable plan signature for debugging and reuse tracking.
      try {
        const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
        const now = Date.now();
        const hit = (store.state as any).planCache.get(key);
        if (hit) {
          hit.hit += 1;
          hit.lastUsed = now;
        } else {
          (store.state as any).planCache.set(key, { signature: key, kind: bundle.main.kind, hit: 0, lastUsed: now, createdAt: now });
        }
      } catch {}
      const snap = await aborts.execute('list.apply', async signal => {
        const res = await execute(bundle, store, 'list', { signal });
        if (signal.aborted) throw new CancellationError();
        return res;
      });
      if (snap.kind === 'group' && !snap.groupRecords) snap.groupRecords = new Map();

      // Backend rows already provide stable grouped keys and labels.
      // Only commit if still the latest apply
      if (seq !== applySeq) throw new CancellationError('Superseded');
      vm.result = snap;
      store.state.result = snap;
      // Reset caches when dataset changes
      groupChildren.clear();
      groupRecords.clear();
      groupRecordRemain.clear();
      recomputeVisible();
      return snap;
    } catch (e) {
      if (isCancellation(e)) {
        // Swallow cancellation as non-error; keep loading state governed by latest request
        return;
      }
      vm.error = e;
      throw e;
    } finally {
      // Only clear loading if this call is the latest
      if (seq === applySeq) vm.loading = false;
    }
  }

  // Controlled operations return Promise<void> while reusing apply() internally.
  async function paginate(p: PaginationState): Promise<void> {
    await apply({ pagination: p });
  }

  async function sort(orderBy: OrderByState[]): Promise<void> {
    const isGroups = vm.result?.kind === 'group';
    await apply({
      orderBy,
      kind: isGroups ? 'group' : 'search',
      appliedGroups: isGroups ? currentGroupbyFromStore() : undefined,
    });
  }

  async function setForcedCondition(filters: QueryCondition<any> | QueryCondition<any>[]): Promise<void> {
    await apply({
      forcedCondition: filters as any,
      kind: vm.result?.kind as any,
      appliedGroups: vm.result?.kind === 'group' ? currentGroupbyFromStore() : undefined,
    });
  }

  async function setAppliedFilters(filters: ConditionGroup[]): Promise<void> {
    await apply({
      appliedFilters: filters,
      kind: vm.result?.kind as any,
      appliedGroups: vm.result?.kind === 'group' ? currentGroupbyFromStore() : undefined,
    });
  }

  async function setKeyword(keyword: string | undefined): Promise<void> {
    await apply({
      keyword,
      kind: vm.result?.kind as any,
      appliedGroups: vm.result?.kind === 'group' ? currentGroupbyFromStore() : undefined,
    });
  }

  async function setKeywordFields(keywordFields: string[]): Promise<void> {
    await apply({
      keywordFields,
      kind: vm.result?.kind as any,
      appliedGroups: vm.result?.kind === 'group' ? currentGroupbyFromStore() : undefined,
    });
  }

  // Accept string[] or GroupBySpec[] and persist normalized group-by values.
  async function setAppliedGroups(groupby: Array<GroupBySpec>): Promise<void> {
    vm.expandedKeys.clear();
    const gb = normalizeGroupbyToStrings(groupby as any);
    await apply({ appliedGroups: gb, kind: gb.length > 0 ? 'group' : 'search' });
  }

  function normalizeFilter<F = any>(f?: F): F | undefined {
    return asPresentCondition(f);
  }
  function combineFilters(a?: any, b?: any): any | undefined {
    const A = normalizeFilter(a);
    const B = normalizeFilter(b);
    if (!A && !B) return undefined;
    if (!A) return B;
    if (!B) return A;
    return { And: [A, B] } as any;
  }

  // buildUnifiedQuery now owns QueryContext construction.

  function findGroupRow(key: string): GroupRow | undefined {
    const r = vm.result;
    if (!r || r.kind !== 'group') return undefined;
    const stack: GroupRow[] = ((r.rows as GroupRow[]) || []).slice();
    while (stack.length) {
      const cur = stack.shift()!;
      if (cur.key === key) return cur;
      const kids = groupChildren.get(cur.key) || [];
      for (const k of kids) stack.push(k);
    }
    return undefined;
  }

  async function ensureGroupLoaded(key: string): Promise<void> {
    const r = vm.result;
    if (!r || r.kind !== 'group') return;
    const row = findGroupRow(key);
    if (!row) return;
    const gb = normalizeGroupbyToStrings(((store.state as any)?.queryState?.appliedGroups as Array<string | GroupBySpec>) ?? []);
    const depth = Number(row.depth ?? 0);
    const isLeaf = gb.length > 0 ? depth >= gb.length - 1 : true;
    if (!isLeaf) {
      if (!groupChildren.has(key)) {
        const kids = await fetchGroupChildren(key);
        groupChildren.set(key, kids || []);
      }
    } else {
      if (!groupRecords.has(key)) {
        const rows = await fetchGroupRecords(key, (store.state as any)?.queryState?.pagination);
        groupRecords.set(key, rows || []);

        // Remaining counts can power future "more" rows.
        try {
          const snap: any = (store.state as any).result;
          // Child-query totals are not exposed here yet.
        } catch {}
      }
    }
  }

  async function expandGroup(key: string, expanded: boolean) {
    if (expanded) vm.expandedKeys.add(key);
    else vm.expandedKeys.delete(key);
    if (expanded) await ensureGroupLoaded(key);
    recomputeVisible();
  }

  async function fetchGroupChildren(_key: string): Promise<GroupRow[]> {
    const parent = findGroupRow(_key);
    if (!parent) return [];
    const gb = normalizeGroupbyToStrings(((store.state as any)?.queryState?.appliedGroups as Array<string | GroupBySpec>) ?? []);
    const depth = Number(parent.depth ?? 0);
    const remaining = gb.slice(depth + 1);
    if (remaining.length === 0) return [];
    const ctx = buildUnifiedQuery(store, {
      groupby: remaining,
      pagination: { ...(store.state as any)?.queryState?.pagination, offset: 0 },
      orderBy: (store.state as any)?.queryState?.orderBy,
      parentCondition: parent.__condition,
    });
    const bundle = buildPlan(ctx);
    try {
      const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
      const now = Date.now();
      const hit = (store.state as any).planCache.get(key);
      if (hit) {
        hit.hit += 1;
        hit.lastUsed = now;
      } else {
        (store.state as any).planCache.set(key, { signature: key, kind: bundle.main.kind, hit: 0, lastUsed: now, createdAt: now });
      }
    } catch {}
    const snap = await aborts.execute(`list.group.children:${_key}`, async signal => {
      const r = await execute(bundle, store, 'list', { signal });
      if (signal.aborted) throw new CancellationError();
      return r;
    });
    const totalChildren = Number((snap as any).total ?? (snap as any).rows?.length ?? 0);
    const existingKeys = new Set<string>();
    const prev = groupChildren.get(parent.key) || [];
    prev.forEach(p => existingKeys.add(p.key));
    const children = (snap.rows as GroupRow[]).map((g, i) => {
      const child: GroupRow = {
        ...g,
        depth: depth + 1,
        key: (() => {
          let raw = (g as any).key;
          if (typeof raw !== 'string' || !raw || raw.includes('[object Object]')) {
            const c = (g as any).__condition;
            let frag = '';
            if (Array.isArray(c) && c.length >= 3) frag = `${c[0]}=${c[2] ?? '__null__'}`;
            if (!frag) frag = g.label || `child@${i}`;
            raw = frag;
          }
          let composed = `${parent.key} > ${raw}`;
          let n = 1;
          let test = composed;
          while (existingKeys.has(test)) {
            n++;
            test = `${composed}#${n}`;
          }
          existingKeys.add(test);
          return test;
        })(),
        __condition: combineFilters(parent.__condition, g.__condition),
      };
      return child;
    });
    groupChildrenRemain.set(parent.key, Math.max(0, totalChildren - children.length));
    return children;
  }

  async function fetchGroupRecords(key: string, pagination?: PaginationState, options?: { skipCount?: boolean }): Promise<RecordRow[] | null> {
    const r = vm.result;
    if (!r || r.kind !== 'group') return null;
    const g = findGroupRow(key);
    if (!g || !g.__condition) return null;

    // Use explicit pagination first, then fall back to queryState.pagination.
    const effectivePagination: PaginationState | undefined = pagination ?? (store.state as any)?.queryState?.pagination;
    const ctx = buildUnifiedQuery(store, {
      pagination: effectivePagination,
      orderBy: (store.state as any)?.queryState?.orderBy,
      parentCondition: g.__condition,
      execOptions: { skipCount: options?.skipCount },
    });
    const bundle = buildPlan(ctx);

    // Record the stable plan signature for debugging and reuse tracking.
    try {
      const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
      const now = Date.now();
      const hit = (store.state as any).planCache.get(key);
      if (hit) {
        hit.hit += 1;
        hit.lastUsed = now;
      } else {
        (store.state as any).planCache.set(key, { signature: key, kind: bundle.main.kind, hit: 0, lastUsed: now, createdAt: now });
      }
    } catch {}
    const snap = await aborts.execute(`list.group.records:${key}`, async signal => {
      const r2 = await execute(bundle, store, 'list', { signal });
      if (signal.aborted) throw new CancellationError();
      return r2;
    });
    let rows = snap.rows as RecordRow[];
    if (!r.groupRecords) r.groupRecords = new Map();

    // Number group rows from 1 and keep numbering across appended pages.
    const existing = groupRecords.get(key) || [];
    rows = rows.map((rr, idx) => ({ ...rr, groupIndex: existing.length + idx + 1 }));
    r.groupRecords.set(key, rows);

    // Sync the internal cache so later loadMore calls compute offsets correctly.
    groupRecords.set(key, rows);

    // Track the remaining row count for future "more" affordances.
    const total = Number(snap.total ?? rows.length);
    groupRecordRemain.set(key, Math.max(0, total - rows.length));
    return rows;
  }

  // Append the next page of leaf rows under a grouped node.
  const loadingMore = new Set<string>();
  async function loadMoreGroupRecords(key: string): Promise<void> {
    // Cancel the previous request so the newest load-more wins.
    if (loadingMore.has(key)) {
      try {
        aborts.cancel(`list.group.more:${key}`);
      } catch {}
    }
    const r = vm.result;
    if (!r || r.kind !== 'group') return;
    const g = findGroupRow(key);
    if (!g || !g.__condition) return;
    try {
      loadingMore.add(key);
      const existing = groupRecords.get(key) || [];

      // Use the shared pagination limit for incremental loading.
      const limit = (store.state as any)?.queryState?.pagination?.limit as number | undefined;
      const offset = existing.length;
      const ctx = buildUnifiedQuery(store, {
        pagination: { limit: limit as any, offset },
        orderBy: (store.state as any)?.queryState?.orderBy,
        parentCondition: g.__condition,
      });
      const bundle = buildPlan(ctx);

      // Record the stable plan signature for debugging and reuse tracking.
      try {
        const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
        const now = Date.now();
        const hit = (store.state as any).planCache.get(key);
        if (hit) {
          hit.hit += 1;
          hit.lastUsed = now;
        } else {
          (store.state as any).planCache.set(key, { signature: key, kind: bundle.main.kind, hit: 0, lastUsed: now, createdAt: now });
        }
      } catch {}
      const snap = await aborts.execute(`list.group.more:${key}`, async signal => {
        const rr = await execute(bundle, store, 'list', { signal });
        if (signal.aborted) throw new CancellationError();
        return rr;
      });
      const rows = ((snap.rows as RecordRow[]) || []).map((rr, idx) => ({ ...rr, groupIndex: existing.length + idx + 1 }));
      const next = existing.concat(rows);

      // Update internal caches and vm.result.groupRecords for tree-style consumers.
      groupRecords.set(key, next);
      if (!r.groupRecords) r.groupRecords = new Map();
      r.groupRecords.set(key, next);
      const total = Number(snap.total ?? next.length);
      groupRecordRemain.set(key, Math.max(0, total - next.length));
      recomputeVisible();
    } catch (e) {
      if (isCancellation(e)) {
        // Cancellation does not count as an error.
        return;
      }
      throw e;
    } finally {
      loadingMore.delete(key);
    }
  }

  async function loadMoreGroupChildren(parentKey: string): Promise<void> {
    const parent = findGroupRow(parentKey);
    if (!parent) return;
    const gb = normalizeGroupbyToStrings(((store.state as any)?.queryState?.appliedGroups as Array<string | GroupBySpec>) ?? []);
    const depth = Number(parent.depth ?? 0);
    const remaining = gb.slice(depth + 1);
    if (remaining.length === 0) return;
    const existing = groupChildren.get(parentKey) || [];
    const limit = (store.state as any)?.queryState?.pagination?.limit ?? 100;
    const offset = existing.length;
    const ctx = buildUnifiedQuery(store, {
      groupby: remaining,
      pagination: { limit, offset },
      orderBy: (store.state as any)?.queryState?.orderBy,
      parentCondition: parent.__condition,
    });
    const bundle = buildPlan(ctx);
    try {
      const key = `${ctx.model}/${ctx.shape}:${stableStringify(ctx.queryState)}`;
      const now = Date.now();
      const hit = (store.state as any).planCache.get(key);
      if (hit) {
        hit.hit += 1;
        hit.lastUsed = now;
      } else {
        (store.state as any).planCache.set(key, { signature: key, kind: bundle.main.kind, hit: 0, lastUsed: now, createdAt: now });
      }
    } catch {}
    const snap = await aborts.execute(`list.group.children.more:${parentKey}`, async signal => {
      const r = await execute(bundle, store, 'list', { signal });
      if (signal.aborted) throw new CancellationError();
      return r;
    });
    const totalChildren = Number((snap as any).total ?? (snap as any).rows?.length ?? offset + ((snap as any).rows?.length || 0));
    const existingKeys = new Set(existing.map(c => c.key));
    const appended = (snap.rows as GroupRow[]).map((g, i) => {
      const child: GroupRow = {
        ...g,
        depth: depth + 1,
        key: (() => {
          let raw = (g as any).key;
          if (typeof raw !== 'string' || !raw || raw.includes('[object Object]')) {
            const c = (g as any).__condition;
            let frag = '';
            if (Array.isArray(c) && c.length >= 3) frag = `${c[0]}=${c[2] ?? '__null__'}`;
            if (!frag) frag = g.label || `child@${i + offset}`;
            raw = frag;
          }
          let composed = `${parent.key} > ${raw}`;
          let n = 1;
          let test = composed;
          while (existingKeys.has(test)) {
            n++;
            test = `${composed}#${n}`;
          }
          existingKeys.add(test);
          return test;
        })(),
        __condition: combineFilters(parent.__condition, g.__condition),
      };
      return child;
    });
    const next = existing.concat(appended);
    groupChildren.set(parentKey, next);
    groupChildrenRemain.set(parentKey, Math.max(0, totalChildren - next.length));
    recomputeVisible();
  }

  const api: IListController = {
    vm,
    bootstrapDefaults,
    apply,
    paginate,
    sort,
    setForcedCondition,
    setAppliedFilters,
    setKeyword,
    setKeywordFields,
    setAppliedGroups,
    expandGroup,
    fetchGroupChildren,
    fetchGroupRecords,
    loadMoreGroupRecords,
    loadMoreGroupChildren,
  };
  return api;
}

export type ListController = ReturnType<typeof createListController>;
