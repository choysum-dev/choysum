// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Creates an automatic onchange controller with nested-relation support and structural-change guards.
 *
 * Features:
 * 1. Deeply watches the draft or record and collects semantic diffs.
 * 2. Tracks both collapsed top-level paths and full leaf paths.
 * 3. Filters structural relation changes to avoid index-shift false positives.
 * 4. Bubbles row-level edits to their relation roots when needed.
 * 5. Uses a single parent Onchange RPC and merges returned collection patches in one pass.
 */

import { ref, watch, nextTick, provide, inject, type Ref } from 'vue';
import { useDebounceFn } from '@vueuse/core';
import { getFieldMetadataView, isRelationFieldType, type WebModelStore, type FieldMetadata } from '@/web/web/stores/modelStore';
import { collectChangedPaths } from '@/core/utils/diff';
import { deepClonePreserve as deepClone } from '@/core/utils/clone';
import type { OnchangeResult } from '@/core/service/api/onchange';
import type { ViewMode, ViewContainer } from '@/web/web/components/view/OViewScope.vue';
/* ============================= Type definitions ============================= */

export interface OnchangeFlushPayload {
  draft: any;
  changed: string[];
  opts: {
    withCompute: boolean;
    maxIterations?: number;
  };
  // Reuse the backend's canonical onchange result type.
  result?: OnchangeResult;
}

export interface OnchangeController {
  running: Ref<boolean>;
  pending: Ref<Set<string>>;

  flush: () => Promise<void>;
  force: (paths: string | string[]) => Promise<void>;
  dispatch: (paths?: string | string[]) => Promise<void>;

  markChanged: (paths: string | string[], opts?: { immediate?: boolean; flush?: boolean }) => Promise<void>;
  hasPending: () => boolean;

  pause: () => void;
  resume: () => void;
  reset: () => void;

  registerAfterFlush: (cb: (p: OnchangeFlushPayload) => void) => void;
  unregisterAfterFlush: (cb: (p: OnchangeFlushPayload) => void) => void;
}

export interface CreateOnchangeOptions {
  debounceMs?: number;
  withCompute?: boolean;
  maxIterations?: number;
  immediateFirst?: boolean;
  collapseRelationChildren?: boolean;

  // Injected root accessor so callers do not depend on store.state._draftRecord/record.
  getRoot?: () => any;
  onPatch?: (value: any, fullResult?: OnchangeResult) => void;
}

/* ============================= Helper functions ============================= */

function looksLikeRelation(meta: FieldMetadata | undefined): boolean {
  if (!meta) return false;
  return isRelationFieldType(meta.type);
}

function buildDiffFieldsMeta(store: WebModelStore<any>): Record<string, { relation?: 'ManyToOne'; type?: string }> {
  const raw = (store as any).fieldsMetadata || {};
  const normalized: Record<string, { relation?: 'ManyToOne'; type?: string }> = {};
  for (const [fieldName, meta] of Object.entries(raw as Record<string, FieldMetadata | undefined>)) {
    const typed = meta as FieldMetadata | undefined;
    const view = getFieldMetadataView(typed);
    normalized[fieldName] = {
      type: typed?.type,
      relation: view.isRelation ? 'ManyToOne' : undefined,
    };
  }
  return normalized;
}

function buildRelationFieldSet(store: WebModelStore<any>): Set<string> {
  const rel = new Set<string>();
  const fields: Record<string, FieldMetadata | undefined> = store.fieldsMetadata || {};
  for (const [fieldName, meta] of Object.entries(fields)) {
    if (looksLikeRelation(meta)) rel.add(fieldName);
  }
  return rel;
}

function relationAwareMinimize(paths: string[], store: WebModelStore<any>, collapseRelationChildren: boolean): string[] {
  if (!paths.length) return [];
  const relationFields = collapseRelationChildren ? buildRelationFieldSet(store) : new Set<string>();

  const preliminary = new Set<string>();
  for (const p of paths) {
    if (!p) continue;
    if (!collapseRelationChildren) {
      preliminary.add(p);
      continue;
    }
    const top = extractBaseRoot(p);
    if (relationFields.has(top)) {
      preliminary.add(top);
    } else {
      preliminary.add(p);
    }
  }

  const sorted = Array.from(preliminary).sort();
  const keep: string[] = [];
  for (const p of sorted) {
    if (keep.some(parent => p !== parent && p.startsWith(parent + '.'))) continue;
    for (let i = keep.length - 1; i >= 0; i--) {
      const k = keep[i];
      if (k !== p && k.startsWith(p + '.')) keep.splice(i, 1);
    }
    keep.push(p);
  }
  return keep;
}

/**
 * Adds top-level relation roots even when collapsed diffs did not detect them.
 */
function augmentCollapsedWithRelationRoots(target: Set<string>, fullLeafPaths: Set<string>) {
  if (!fullLeafPaths.size) return;
  for (const leaf of fullLeafPaths) {
    if (!leaf || !leaf.includes('.')) continue;
    const root = extractBaseRoot(leaf);
    if (root && !target.has(root)) target.add(root);
  }
}

function isIndexSeg(seg: string) {
  return /^\d+$/.test(seg);
}

// Normalize array-index syntax, for example Lines[0].Qty -> Lines.0.Qty.
function normalizeArrayIndexInPath(path: string): string {
  return path ? path.replace(/\[(\d+)\]/g, '.$1') : path;
}

// Extract the raw collection root from a selector path by removing (...) or [...].
function extractBaseRoot(path: string): string {
  const head = (path.split('.')[0] || '').trim();
  const m = head.match(/^([A-Za-z_][A-Za-z0-9_]*)/);
  return m ? m[1] : head;
}

/**
 * Converts a leaf path into a one-hop field signal with indexes removed.
 * For example: Lines.1.UnitPrice => Lines.UnitPrice
 *              Lines.1.Batches.0.Qty => Lines.Batches.Qty
 */
function toOneHopFieldSignal(leafPath: string): string | null {
  if (!leafPath || !leafPath.includes('.')) return null;
  const segs = leafPath.split('.').filter(Boolean);
  if (segs.length < 2) return null;
  const cleaned: string[] = [];
  for (const s of segs) {
    if (isIndexSeg(s)) continue;
    cleaned.push(s);
  }
  return cleaned.join('.');
}

/**
 * Collects ancestor collection roots from a leaf path when a segment is followed by an index.
 * For example: Lines.1.UnitPrice => ['Lines']
 *              Lines.1.Batches.0.Qty => ['Lines', 'Lines.Batches']
 */
function collectAncestorCollectionRoots(leafPath: string): string[] {
  const out: string[] = [];
  if (!leafPath) return out;
  const segs = leafPath.split('.').filter(Boolean);
  const accum: string[] = [];
  for (let i = 0; i < segs.length - 1; i++) {
    const cur = segs[i];
    const next = segs[i + 1];
    if (isIndexSeg(next)) {
      accum.push(cur);
      out.push(accum.join('.'));
      i++;
    }
  }
  return out;
}

// Slim reference-like values only for the currently changed fields.
function slimRelationRefsForChanged(draft: any, changed: string[], fieldsMeta?: Record<string, FieldMetadata | undefined>): any {
  if (!draft || !changed?.length) return draft;

  const out = deepClone(draft);
  const lowerType = (meta?: FieldMetadata) => (typeof meta?.type === 'string' ? meta.type.toLowerCase() : '');
  const metaMap = fieldsMeta || {};

  const isRefLike = (v: any) => {
    if (!v || typeof v !== 'object') return false;
    const keys = Object.keys(v);
    const hasId = 'Id' in v || 'id' in v;
    if (!hasId) return false;
    // Only convert values that already look like compact reference payloads.
    return keys.every(k => k === 'Id' || k === 'id' || k === 'DisplayName' || k === '__rowKey');
  };

  const refId = (v: any) => {
    if (v == null) return v;
    if (typeof v === 'object') {
      if ('Id' in v) return (v as any).Id;
      if ('id' in v) return (v as any).id;
    }
    return v;
  };

  const isNumeric = (seg: string) => /^\d+$/.test(seg);

  const getAtPath = (obj: any, segs: string[]) => {
    let cur = obj;
    for (const s of segs) {
      if (cur == null) return undefined;
      const key: any = isNumeric(s) ? Number(s) : s;
      cur = cur?.[key];
    }
    return cur;
  };

  const ensurePathCloned = (obj: any, segs: string[]) => {
    let cur = obj;
    for (let i = 0; i < segs.length - 1; i++) {
      const key: any = isNumeric(segs[i]) ? Number(segs[i]) : segs[i];
      const next = cur?.[key];
      const cloned = Array.isArray(next) ? next.slice() : next && typeof next === 'object' ? { ...next } : isNumeric(segs[i + 1]) ? [] : {};
      cur[key] = cloned;
      cur = cloned;
    }
    return cur;
  };

  const setAtPath = (obj: any, segs: string[], value: any) => {
    if (!segs.length) return;
    if (segs.length === 1) {
      const key: any = isNumeric(segs[0]) ? Number(segs[0]) : segs[0];
      obj[key] = value;
      return;
    }
    const parent = ensurePathCloned(obj, segs);
    const key: any = isNumeric(segs[segs.length - 1]) ? Number(segs[segs.length - 1]) : segs[segs.length - 1];
    parent[key] = value;
  };

  for (const raw of changed) {
    if (!raw) continue;
    const segs = normalizeArrayIndexInPath(raw).split('.').filter(Boolean);
    if (!segs.length) continue;

    const val = getAtPath(out, segs);
    if (val === undefined) continue;

    const root = extractBaseRoot(raw);
    const rootMeta = metaMap[root];
    const rootType = lowerType(rootMeta);

    if (isRefLike(val) || rootType === 'manytooneref') {
      setAtPath(out, segs, refId(val));
      continue;
    }

    if (Array.isArray(val)) {
      const mapped = val.map(item => {
        if (isRefLike(item) || rootType === 'manytomanyref') return refId(item);
        return item;
      });
      // Drop empty values for many-to-many reference payloads.
      const finalVal = rootType === 'manytomanyref' ? mapped.filter(v => v !== undefined && v !== null) : mapped;
      setAtPath(out, segs, finalVal);
    }
  }

  return out;
}

/** Returns the array at a path whose final segment names the collection, or null on failure. */
function getArrayAtPath(root: any, pathKey: string): any[] | null {
  const segs = pathKey.split('.').filter(Boolean);
  if (!segs.length) return null;
  let node: any = root;
  for (let i = 0; i < segs.length; i++) {
    const seg = segs[i];
    if (i === segs.length - 1) {
      const arr = node?.[seg];
      return Array.isArray(arr) ? arr : null;
    }
    node = node?.[seg];
    if (!node) return null;
  }
  return null;
}

function findIndexById(arr: any[], id: any): number {
  if (!Array.isArray(arr)) return -1;
  const sid = String(id);
  for (let i = 0; i < arr.length; i++) {
    const rid = arr[i]?.Id ?? arr[i]?.id;
    if (rid != null && String(rid) === sid) return i;
  }
  return -1;
}

/**
 * Recursively finds an array row by Id across nested collections.
 * Segments may look like ['Lines'] or ['Lines', 'Batches'].
 */
function deepFindById(root: any, segments: string[], targetId: any): { arr: any[]; idx: number } | null {
  if (!segments.length) return null;
  const sid = String(targetId);

  function scanLevel(node: any, level: number): { arr: any[]; idx: number } | null {
    const key = segments[level];
    const col = node?.[key];
    if (!Array.isArray(col)) return null;

    if (level === segments.length - 1) {
      const idx = findIndexById(col, sid);
      return idx >= 0 ? { arr: col, idx } : null;
    }

    // Continue scanning into the next nested collection level.
    for (const row of col) {
      const hit = scanLevel(row, level + 1);
      if (hit) return hit;
    }
    return null;
  }

  return scanLevel(root, 0);
}

function applyRowPatchToArray(arr: any[], idx: number, patch: Record<string, any>) {
  if (!Array.isArray(arr) || idx < 0 || idx >= arr.length) return;
  for (const [k, v] of Object.entries(patch)) {
    if (k === 'Id' || k === 'id' || k === 'pos' || k === 'ParentId' || k === 'parentId' || k === 'parentPos' || k === 'ParentPos') continue;
    arr[idx][k] = v;
  }
}

/**
 * Converts a leaf path such as Lines.0.UnitPrice into selector syntax.
 * Supports nested collection paths.
 */
function toSelectorPath(rootObj: any, leafPath: string): string | null {
  if (!leafPath || !leafPath.includes('.')) return null;
  const segs = leafPath.split('.').filter(Boolean);
  let obj = rootObj;
  const out: string[] = [];
  for (let i = 0; i < segs.length; i++) {
    const seg = segs[i];

    if (isIndexSeg(seg)) {
      const arrKey = segs[i - 1];
      const arr = Array.isArray(obj) ? obj : obj?.[arrKey];
      const idx = Number(seg);

      let resolved = false;
      if (Array.isArray(arr)) {
        const row = arr[idx];
        if (row != null) {
          const id = row?.Id ?? row?.id;
          out[out.length - 1] = id != null ? `${arrKey}(id=${id})` : `${arrKey}[${idx}]`;
          obj = row;
          resolved = true;
        }
      }
      if (!resolved) {
        out[out.length - 1] = `${arrKey}[${idx}]`;
        obj = Array.isArray(arr) ? arr[idx] : undefined;
      }
      continue;
    }

    out.push(seg);
    obj = obj?.[seg];
  }
  return out.join('.');
}

/* ====================== Structural change detection ====================== */

/**
 * Detects whether a top-level relation changed structurally.
 * Structural changes include different lengths or different Id sequences.
 */
function detectStructuralChangedRelations(collapsed: Set<string>, baseline: any, current: any, store: WebModelStore<any>): Set<string> {
  const structural = new Set<string>();
  if (!collapsed.size) return structural;
  const meta = (store as any).fieldsMetadata || {};

  for (const top of collapsed) {
    const m = meta[top];
    const fieldType = typeof m?.type === 'string' ? m.type : '';
    const lowerType = fieldType.toLowerCase();
    const isRelArrayExplicit = isRelationFieldType(fieldType) && (lowerType === 'onetomany' || lowerType === 'manytomany' || lowerType === 'manytomanyref');
    const oldArr = baseline && Array.isArray(baseline[top]) ? baseline[top] : undefined;
    const newArr = current && Array.isArray(current[top]) ? current[top] : undefined;

    const looksArrayRelation = isRelArrayExplicit || Array.isArray(oldArr) || Array.isArray(newArr);

    if (!looksArrayRelation) continue;

    const a: any[] = Array.isArray(oldArr) ? oldArr : [];
    const b: any[] = Array.isArray(newArr) ? newArr : [];

    if (a.length !== b.length) {
      structural.add(top);
      continue;
    }

    // Compare row Id sequences.
    const changedSeq = a.some((row, i) => {
      const idA = row?.Id ?? row?.id ?? null;
      const idB = b[i]?.Id ?? b[i]?.id ?? null;
      return idA !== idB;
    });
    if (changedSeq) {
      structural.add(top);
      continue;
    }
  }

  return structural;
}

/* ====================== Core controller ====================== */

function createAutoOnchangeController(store: WebModelStore<any>, opts?: CreateOnchangeOptions): OnchangeController {
  const debounceMs = opts?.debounceMs ?? 120;

  const running = ref(false);
  const pending = ref<Set<string>>(new Set());
  const fullPathsRef = ref<Set<string>>(new Set());
  const paused = ref(false);

  let baseline: any = null;
  let applying = false;
  let initialized = false;

  const afterFlushCallbacks = new Set<(p: OnchangeFlushPayload) => void>();

  // Prefer the injected root accessor while keeping the legacy fallback path.
  const root = () => (opts?.getRoot ? opts.getRoot() : (store as any).state?._draftRecord || (store as any).state?.record);

  function ensureBaseline() {
    if (baseline == null && root()) baseline = deepClone(root());
  }

  // Define the debounced flush early to avoid temporal dead-zone issues.
  const debouncedFlush = useDebounceFn(() => {
    if (paused.value || !pending.value.size) return;
    void internalFlush();
  }, debounceMs);

  // Shared debounce cancellation helper.
  function cancelDebounced() {
    (debouncedFlush as any)?.cancel?.();
  }

  // Pause in display mode and resume in edit/create mode while resetting tracked state.
  const injectedViewMode = inject<Ref<ViewMode> | null>('view-mode', null as any);
  if (injectedViewMode) {
    watch(
      injectedViewMode,
      mode => {
        // Reset tracked state so initial detail-page values are not treated as changes.
        const r = root();
        baseline = r ? deepClone(r) : null;
        pending.value.clear();
        fullPathsRef.value.clear();

        if (mode === 'display') {
          paused.value = true;
          cancelDebounced();
        } else {
          const wasPaused = paused.value;
          paused.value = false;
          if (wasPaused && pending.value.size) {
            debouncedFlush();
          }
        }
      },
      { immediate: true }
    );
  }

  function invokeAfterFlush(payload: OnchangeFlushPayload) {
    if (!afterFlushCallbacks.size) return;
    for (const cb of Array.from(afterFlushCallbacks)) {
      try {
        cb(payload);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn('[Onchange] afterFlush callback error', e);
      }
    }
  }

  async function internalFlush(): Promise<void> {
    if (running.value || !pending.value.size) return;
    const r = root();
    if (!r || typeof store.Onchange !== 'function') {
      pending.value.clear();
      fullPathsRef.value.clear();
      return;
    }
    running.value = true;
    applying = true;
    try {
      const changedRaw = Array.from(pending.value);
      const fullSnapshot = new Set(fullPathsRef.value);
      pending.value.clear();
      fullPathsRef.value.clear();

      if (!changedRaw.length) return;

      const minimized = relationAwareMinimize(changedRaw, store, opts?.collapseRelationChildren !== false);

      const selectorPaths = new Set<string>();
      for (const leaf of fullSnapshot) {
        const selPath = toSelectorPath(r, normalizeArrayIndexInPath(leaf));
        if (selPath) selectorPaths.add(selPath);
      }

      for (const raw of changedRaw) {
        const normalized = normalizeArrayIndexInPath(raw);
        const selPath = toSelectorPath(r, normalized);
        if (selPath && selPath !== raw) selectorPaths.add(selPath);
      }

      const selectorRoots = new Set<string>();
      for (const p of selectorPaths) {
        selectorRoots.add(extractBaseRoot(p));
      }
      const changed: string[] = [...minimized.filter(f => !selectorRoots.has(extractBaseRoot(f))), ...selectorPaths];

      if (!selectorPaths.size) {
        const oneHopFields = new Set<string>();
        const ancestorRoots = new Set<string>();
        for (const leaf of fullSnapshot) {
          const normalized = normalizeArrayIndexInPath(leaf);
          const sig = toOneHopFieldSignal(normalized);
          if (sig) oneHopFields.add(sig);
          const roots = collectAncestorCollectionRoots(normalized);
          roots.forEach(rr => ancestorRoots.add(rr));
        }
        changed.push(...oneHopFields, ...ancestorRoots);
      }

      if (!changed.length) return;

      const draftForSend = slimRelationRefsForChanged(r, changed, (store as any).fieldsMetadata);

      const payload: OnchangeFlushPayload = {
        draft: draftForSend,
        changed: Array.from(new Set(changed)),
        opts: {
          withCompute: opts?.withCompute !== false,
          maxIterations: opts?.maxIterations,
        },
      };

      const res = await store.Onchange(payload.draft, payload.changed as any, payload.opts);

      const hasError = Array.isArray(res?.messages) && res!.messages!.some(m => m?.level === 'error');

      if (hasError) {
        payload.result = res as any;
        invokeAfterFlush(payload);
        await nextTick();
        baseline = deepClone(root());
        return;
      }

      if (res?.value) {
        const draft = root();
        // Allow external patch handling when no root object is available.
        if (!draft) {
          opts?.onPatch?.(res.value as any, res as any);
          await nextTick();
          baseline = deepClone(root());
          payload.result = res as any;
          invokeAfterFlush(payload);
          return;
        }
        const patchContainer = (res.value as any).__collectionPatch;

        if (patchContainer && typeof patchContainer === 'object') {
          for (const [pathKey, rows] of Object.entries(patchContainer) as [string, any[]][]) {
            if (!Array.isArray(rows) || !rows.length) continue;
            const segs = pathKey.split('.').filter(Boolean);

            if (segs.length === 1) {
              const coll = segs[0];
              const arr: any[] = Array.isArray(draft[coll]) ? draft[coll] : [];
              if (!arr.length) continue;

              const idIndex = new Map<string, number>();
              for (let i = 0; i < arr.length; i++) {
                const id = arr[i]?.Id ?? arr[i]?.id;
                if (id != null) idIndex.set(String(id), i);
              }

              for (const rp of rows) {
                const Id = rp?.Id ?? rp?.id;
                if (Id != null && idIndex.has(String(Id))) {
                  applyRowPatchToArray(arr, idIndex.get(String(Id))!, rp);
                } else if (typeof rp?.pos === 'number') {
                  applyRowPatchToArray(arr, rp.pos, rp);
                }
              }
              continue;
            }

            for (const rp of rows) {
              const Id = rp?.Id ?? rp?.id;
              if (Id != null) {
                const found = deepFindById(draft, segs, Id);
                if (found) {
                  applyRowPatchToArray(found.arr, found.idx, rp);
                  continue;
                }
              }

              const parentId = rp?.ParentId ?? rp?.parentId ?? null;
              const parentPos = rp?.ParentPos ?? rp?.parentPos ?? null;
              const lastSeg = segs[segs.length - 1];

              if (segs.length >= 2) {
                const parentPathSegs = segs.slice(0, segs.length - 1);

                let parentNode: any = draft;
                let ok = true;

                if (Array.isArray(rp?.__parents) && rp.__parents.length) {
                  for (let level = 0; level < parentPathSegs.length; level++) {
                    const key = parentPathSegs[level];
                    const arrAt = Array.isArray(parentNode?.[key]) ? parentNode[key] : null;
                    if (!arrAt) {
                      ok = false;
                      break;
                    }
                    const loc = rp.__parents[level];
                    let idx = -1;
                    if (loc?.Id != null) idx = findIndexById(arrAt, loc.Id);
                    else if (typeof loc?.pos === 'number') idx = loc.pos;
                    if (idx < 0 || idx >= arrAt.length) {
                      ok = false;
                      break;
                    }
                    parentNode = arrAt[idx];
                  }
                } else {
                  if (parentPathSegs.length === 1) {
                    const parentArr = getArrayAtPath(draft, parentPathSegs[0]);
                    if (!parentArr) ok = false;
                    else {
                      let pIdx = -1;
                      if (parentId != null) pIdx = findIndexById(parentArr, parentId);
                      else if (typeof parentPos === 'number') pIdx = parentPos;
                      if (pIdx < 0 || pIdx >= parentArr.length) ok = false;
                      else parentNode = parentArr[pIdx];
                    }
                  } else {
                    ok = false;
                  }
                }

                if (ok && parentNode && Array.isArray(parentNode[lastSeg])) {
                  const childArr: any[] = parentNode[lastSeg];
                  if (Id != null) {
                    const idx = findIndexById(childArr, Id);
                    if (idx >= 0) {
                      applyRowPatchToArray(childArr, idx, rp);
                      continue;
                    }
                  }
                  if (typeof rp?.pos === 'number') {
                    applyRowPatchToArray(childArr, rp.pos, rp);
                    continue;
                  }
                }
              }

              // eslint-disable-next-line no-console
              console.warn('[Onchange] cannot apply nested patch for key:', pathKey, rp);
            }
          }
          delete (res.value as any).__collectionPatch;
        }

        // Merge the remaining patch payload onto the root object.
        Object.assign(draft, res.value);
      }

      await nextTick();
      baseline = deepClone(root());

      payload.result = res as any;
      invokeAfterFlush(payload);
    } finally {
      applying = false;
      running.value = false;
    }
  }

  async function flush() {
    cancelDebounced();
    if (!pending.value.size) {
      const r = root();
      if (r && baseline) {
        try {
          const diffFieldsMeta = buildDiffFieldsMeta(store);

          // 1) Top-level collapsed signals such as Lines or DiscountRate.
          const collapsed = collectChangedPaths(baseline, r, {
            pruneRelationChildren: true,
            fieldsMeta: diffFieldsMeta as any,
          });
          if (collapsed.size) collapsed.forEach(p => pending.value.add(p));

          // 2) Full leaf paths with indexes preserved for later one-hop and ancestor signals.
          const fullRaw = collectChangedPaths(baseline, r, {
            pruneRelationChildren: false,
            includeTopLevel: false,
            includeFullPath: true,
            normalizeArrayIndex: false,
            fieldsMeta: diffFieldsMeta as any,
            collapseFinal: false,
          });

          // 3) Drop indexed leaf paths when a relation changed structurally.
          const structuralRels = detectStructuralChangedRelations(collapsed, baseline, r, store);
          const filtered = new Set<string>();
          for (const p of fullRaw) {
            const rootRel = p.split('.')[0];
            if (structuralRels.has(rootRel)) continue;
            filtered.add(p);
          }
          fullPathsRef.value = filtered;

          // 4) Bubble relation roots in case collapsed diffs missed them.
          augmentCollapsedWithRelationRoots(pending.value, fullPathsRef.value);
        } catch {
          /* ignore */
        }
      }
    }
    await internalFlush();
  }

  async function force(paths: string | string[]) {
    const arr = Array.isArray(paths) ? paths : [paths];
    arr.filter(Boolean).forEach(p => pending.value.add(p));
    await flush();
  }

  async function dispatch(paths?: string | string[]) {
    if (!paths) return flush();
    return force(paths);
  }

  function schedule(immediate = false) {
    if (paused.value || !pending.value.size) return;
    if (immediate) {
      cancelDebounced();
      void flush();
    } else {
      debouncedFlush();
    }
  }

  function pause() {
    paused.value = true;
  }
  function resume() {
    if (!paused.value) return;
    paused.value = false;
    if (pending.value.size) schedule(false);
  }

  function reset() {
    cancelDebounced();
    pending.value.clear();
    fullPathsRef.value.clear();
    baseline = null;
    applying = false;
    running.value = false;
    paused.value = false;
    initialized = false;
  }

  function markChanged(paths: string | string[], mOpts?: { immediate?: boolean; flush?: boolean }) {
    const arr = Array.isArray(paths) ? paths : [paths];
    arr.filter(Boolean).forEach(p => pending.value.add(p));
    if (mOpts?.flush) return flush();
    schedule(!!mOpts?.immediate);
    return Promise.resolve();
  }

  function hasPending() {
    return pending.value.size > 0;
  }

  // Watch the draft or record and collect changed paths.
  watch(
    () => root(),
    cur => {
      if (applying || !cur) return;
      ensureBaseline();
      if (!baseline) {
        baseline = deepClone(cur);
        return;
      }
      try {
        const diffFieldsMeta = buildDiffFieldsMeta(store);
        // 1) Top-level collapsed signals.
        const collapsed = collectChangedPaths(baseline, cur, {
          pruneRelationChildren: true,
          fieldsMeta: diffFieldsMeta as any,
        });

        // 2) Full leaf paths with indexes preserved.
        const fullRaw = collectChangedPaths(baseline, cur, {
          pruneRelationChildren: false,
          includeTopLevel: false,
          includeFullPath: true,
          normalizeArrayIndex: false,
          fieldsMeta: diffFieldsMeta as any,
          collapseFinal: false,
        });

        // 3) Keep only safe leaf paths after structural-change filtering.
        const structuralRels = detectStructuralChangedRelations(collapsed, baseline, cur, store);
        const full = new Set<string>();
        for (const p of fullRaw) {
          const rel = p.split('.')[0];
          if (structuralRels.has(rel)) continue;
          full.add(p);
        }

        if (collapsed.size) collapsed.forEach(p => pending.value.add(p));
        fullPathsRef.value = full;
        augmentCollapsedWithRelationRoots(pending.value, fullPathsRef.value);

        if (!pending.value.size) return;

        if (opts?.immediateFirst && !initialized) {
          initialized = true;
          schedule(true);
        } else {
          initialized = true;
          schedule(false);
        }
      } catch (e) {
        // eslint-disable-next-line no-console
        console.warn('[Onchange] diff error:', e);
      }
    },
    { deep: true }
  );

  return {
    running,
    pending,
    flush,
    force,
    dispatch,
    markChanged,
    hasPending,
    pause,
    resume,
    reset,
    registerAfterFlush(cb) {
      afterFlushCallbacks.add(cb);
    },
    unregisterAfterFlush(cb) {
      afterFlushCallbacks.delete(cb);
    },
  };
}

/* ====================== Controller cache (session) ====================== */

const SINGLETON = new WeakMap<WebModelStore<any>, OnchangeController>();
const PER_SESSION = new WeakMap<WebModelStore<any>, Map<string, OnchangeController>>();

export function getOnchangeController(store: WebModelStore<any>, sessionId?: string, opts?: CreateOnchangeOptions): OnchangeController {
  if (!sessionId) {
    let c = SINGLETON.get(store);
    if (!c) {
      c = createAutoOnchangeController(store, opts);
      SINGLETON.set(store, c);
    }
    return c;
  }
  let map = PER_SESSION.get(store);
  if (!map) {
    map = new Map();
    PER_SESSION.set(store, map);
  }
  let c = map.get(sessionId);
  if (!c) {
    c = createAutoOnchangeController(store, opts);
    map.set(sessionId, c);
  }
  return c;
}

/* ====================== Provide / Inject ====================== */

export const OnchangeControllerKey: unique symbol = Symbol('OnchangeController');

export function provideOnchange(store: WebModelStore<any>, sessionId?: string, opts?: CreateOnchangeOptions): OnchangeController {
  const ctrl = getOnchangeController(store, sessionId, opts);
  provide(OnchangeControllerKey, ctrl);
  return ctrl;
}

export function useProvidedOnchange(): OnchangeController | null {
  return inject<OnchangeController | null>(OnchangeControllerKey, null);
}

export function useOrCreateOnchangeController(store: WebModelStore<any>, sessionId?: string, opts?: CreateOnchangeOptions): OnchangeController {
  const injected = useProvidedOnchange();
  if (injected) return injected;
  return getOnchangeController(store, sessionId, opts);
}

/* ====================== Cleanup helpers ====================== */

export function disposeOnchange(store: WebModelStore<any>) {
  const single = SINGLETON.get(store);
  if (single) {
    try {
      single.pause();
      single.reset();
    } catch {}
    SINGLETON.delete(store);
  }
  const map = PER_SESSION.get(store);
  if (map) {
    for (const [, ctrl] of map) {
      try {
        ctrl.pause();
        ctrl.reset();
      } catch {}
    }
    PER_SESSION.delete(store);
  }
}
