// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata, OnchangeHandlerMeta } from '../metadata';
import type { OnchangeResult } from '../../runtime/onchange/types';
import { parseOnchangeReadsEx } from '../../runtime/onchange/reads';
import { PathPlanBuilder } from '../../runtime/onchange/plan';
import { ENABLE_CHILD_ONCHANGE_IN_PREVIEW, PREVIEW_CASCADE_MAX_DEPTH } from '../../runtime/onchange/constants';
import { createPreviewProxy } from '../../runtime/proxy';
import type { ParsedChangedSelectors, RowSelector } from '../../runtime/onchange/selectors';
import { getModelRuntimeMetadata, recomputeModelMetadata, runModelOnchangePreviewEngine } from './model_runtime_service_facade';
import type BaseModel from './model';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

type ModelOnchangePreviewOptions = {
  withCompute?: boolean;
  maxIterations?: number;
  loopThreshold?: number;
};

type MutableOnchangeResult = OnchangeResult & {
  value?: ObjectRecord;
  messages?: Array<ObjectRecord>;
  condition?: Array<{ field: string; condition: unknown }>;
  selection?: Array<{ field: string; selection: unknown; disabled?: unknown[] }>;
  computeRecomputed?: string[];
};

function matchesSelection(sel: RowSelector | undefined, row: ObjectRecord | undefined, idx: number): boolean {
  if (!sel || sel.kind === 'all') return true;
  if (sel.kind === 'pos') return sel.positions.has(idx);
  if (sel.kind === 'id') {
    const id = row?.Id ?? row?.id;
    return id != null && sel.ids.has(String(id));
  }
  return true;
}

function formatSelector(sel: RowSelector | undefined, row: ObjectRecord | undefined, idx: number): string {
  if (!sel || sel.kind === 'all') return '';
  if (sel.kind === 'pos') return `[${idx}]`;
  if (sel.kind === 'id') {
    const id = row?.Id ?? row?.id;
    return id != null ? `(id=${String(id)})` : '';
  }
  return '';
}

export async function applyModelOnchangePreviewCascade(params: {
  meta: ModelMetadata;
  previewProxy: ObjectRecord;
  changedFields: string[];
  selParsed: ParsedChangedSelectors;
  opts?: ModelOnchangePreviewOptions;
  res: OnchangeResult;
}): Promise<void> {
  const { meta, previewProxy, changedFields, selParsed, opts, res } = params;
  const mutableRes = res as MutableOnchangeResult;

  try {
    const collectionRootKeys = new Set<string>(selParsed.collectionRoots);
    const childSignals = new Map<string, Set<string>>(selParsed.fieldSignals);

    const topLevelO2M = new Set<string>();
    meta.fields?.forEach((f, name) => {
      if (f?.type === 'OneToMany') topLevelO2M.add(name);
    });

    for (const c of changedFields) {
      if (!c) continue;
      const segs = c.split('.').filter(Boolean);
      if (!segs.length) continue;

      if (segs.length === 1 && topLevelO2M.has(segs[0])) {
        collectionRootKeys.add(segs[0]);
        continue;
      }

      if (segs.length >= 2) {
        if (topLevelO2M.has(segs[0])) {
          const rootKey = segs[0];
          collectionRootKeys.add(rootKey);
          const leaf = segs[segs.length - 1];
          if (!childSignals.has(rootKey)) childSignals.set(rootKey, new Set<string>());
          childSignals.get(rootKey)!.add(leaf);
        }
        if (segs.length >= 3 && topLevelO2M.has(segs[0])) {
          const key = `${segs[0]}.${segs[1]}`;
          collectionRootKeys.add(key);
          const leaf = segs[segs.length - 1];
          if (!childSignals.has(key)) childSignals.set(key, new Set<string>());
          childSignals.get(key)!.add(leaf);
        }
      }
    }

    const parentChanged = new Set<string>([...changedFields, ...Object.keys(mutableRes.value || {})]);
    for (const [fieldName, fMeta] of meta.fields) {
      if (fMeta?.type !== 'OneToMany') continue;
      const arr = previewProxy[fieldName];
      if (!Array.isArray(arr) || arr.length === 0) continue;

      const childCtor = fMeta.relation?.targetModel?.();
      const inverseField: string | undefined = fMeta.relation?.inverseField;
      if (!childCtor || !inverseField) continue;

      const childMeta = getModelRuntimeMetadata(childCtor);
      const g = childMeta.computeGraph;
      if (!g?.computePathDeps) continue;

      let hit = false;
      for (const deps of g.computePathDeps.values()) {
        for (const d of deps || []) {
          if (d.root !== inverseField) continue;
          const first = Array.isArray(d.chain) && d.chain.length ? d.chain[0] : null;
          if (first && parentChanged.has(first)) {
            hit = true;
            break;
          }
        }
        if (hit) break;
      }
      if (hit) collectionRootKeys.add(fieldName);
    }

    const collectionPatch: Record<string, ObjectRecord[]> = {};
    const affectedTopRoots = new Set<string>();

    const parentToChildren = new Map<string, string[]>();
    for (const key of collectionRootKeys) {
      const parts = key.split('.').filter(Boolean);
      if (parts.length >= 2) {
        const parentKey = parts.slice(0, parts.length - 1).join('.');
        if (!parentToChildren.has(parentKey)) parentToChildren.set(parentKey, []);
        parentToChildren.get(parentKey)!.push(key);
      }
    }

    const rowSelectors = selParsed.selectors;

    const cascadeCollection = async (ownerMeta: ModelMetadata, ownerObj: ObjectRecord, pathKey: string, depth: number, labelPrefix?: string) => {
      if (depth > PREVIEW_CASCADE_MAX_DEPTH) return;

      const segs = pathKey.split('.').filter(Boolean);
      const collectionField = segs[segs.length - 1];
      const fieldMeta = ownerMeta.fields.get(collectionField);
      if (!fieldMeta || fieldMeta.type !== 'OneToMany') return;

      const childCtor = fieldMeta.relation?.targetModel?.();
      const inverseField: string | undefined = fieldMeta.relation?.inverseField;
      if (!childCtor || !inverseField) return;

      const childMeta = getModelRuntimeMetadata(childCtor);
      const childGraph = childMeta.computeGraph;

      const rows: unknown[] = Array.isArray(ownerObj?.[collectionField]) ? ownerObj[collectionField] : [];
      if (!rows.length) return;

      const computeSet = new Set<string>();
      const neededParentFields = new Set<string>();
      const pathDeps = childGraph?.computePathDeps || new Map<string, Array<{ root: string; chain: string[] }>>();
      for (const [computeField, deps] of pathDeps.entries()) {
        for (const d of deps || []) {
          if (!d || d.root !== inverseField) continue;
          computeSet.add(computeField);
          const chain0 = Array.isArray(d.chain) && d.chain.length ? d.chain[0] : null;
          if (chain0) neededParentFields.add(chain0);
        }
      }

      const childSignalSet = new Set<string>(childSignals.get(pathKey) || []);

      const parentStub: ObjectRecord = {};
      if ('Id' in ownerObj && ownerObj.Id != null) parentStub.Id = ownerObj.Id;
      neededParentFields.forEach(k => {
        if (k in ownerObj) parentStub[k] = ownerObj[k];
      });

      const rowPatches: ObjectRecord[] = [];
      const sel = rowSelectors.get(pathKey);
      const baseLabel = labelPrefix ? `${labelPrefix}.${collectionField}` : collectionField;

      for (let i = 0; i < rows.length; i++) {
        const src = asObjectRecord(rows[i]);
        if (!src) continue;
        if (!matchesSelection(sel, src, i)) continue;

        const inverseObj = asObjectRecord(src[inverseField]) || {};

        const childDraft: ObjectRecord = {
          ...src,
          [inverseField]: { ...inverseObj, ...parentStub },
        };

        let childActive: OnchangeHandlerMeta[] = [];
        if (ENABLE_CHILD_ONCHANGE_IN_PREVIEW && childSignalSet.size) {
          childActive = (childMeta.onchangeHandlers || []).filter(h => h.triggers.some(t => childSignalSet.has(t)));
          try {
            const parsed = parseOnchangeReadsEx(childMeta, childActive);
            const cached = PathPlanBuilder.getCachedOrBuildV2(childCtor, parsed.m2o, new Map(), new Map(), new Map());
            PathPlanBuilder.executeWithPlan(childCtor, childMeta, childDraft, cached.plan);
          } catch {
            // ignore
          }
        }

        const childPreviewInstance = Object.assign(Object.create(childCtor.prototype), childDraft);

        const childAllFieldNames = new Set<string>(Array.from(childMeta.fields?.keys?.() || []));
        const childProxy = createPreviewProxy(childPreviewInstance as BaseModel, {
          meta: childMeta,
          triggers: new Set(childSignalSet),
          reads: childAllFieldNames,
          loaded: new Set(Object.keys(childDraft)),
        });

        let childRes: OnchangeResult | null = null;
        if (ENABLE_CHILD_ONCHANGE_IN_PREVIEW && childSignalSet.size) {
          try {
            childRes = await runModelOnchangePreviewEngine(childMeta, childProxy, [...childSignalSet], {
              ...opts,
              withCompute: true,
            });
          } catch {
            // ignore
          }
        }

        const computeSeed = new Set<string>(childSignalSet);
        computeSeed.add(inverseField);
        try {
          await recomputeModelMetadata(childMeta, childProxy, computeSeed, 'preview');
        } catch {
          // ignore
        }

        const labelForRow = `${baseLabel}${formatSelector(sel, src, i)}`;

        if (childRes?.messages?.length) {
          if (!Array.isArray(mutableRes.messages)) mutableRes.messages = [];
          const prefixed = childRes.messages.map(message =>
            message && typeof message.field === 'string' ? { ...message, field: `${labelForRow}.${message.field}` } : message
          ) as Array<ObjectRecord>;
          mutableRes.messages.push(...prefixed);
        }

        if (childRes?.condition?.length) {
          if (!Array.isArray(mutableRes.condition)) mutableRes.condition = [];
          for (const condition of childRes.condition) {
            if (!condition || typeof condition.field !== 'string') continue;
            mutableRes.condition.push({
              field: `${labelForRow}.${condition.field}`,
              condition: condition.condition,
            });
          }
        }

        if (childRes?.selection?.length) {
          if (!Array.isArray(mutableRes.selection)) mutableRes.selection = [];
          for (const selection of childRes.selection) {
            if (!selection || typeof selection.field !== 'string') continue;
            const entry: { field: string; selection: unknown; disabled?: unknown[] } = {
              field: `${labelForRow}.${selection.field}`,
              selection: selection.selection,
            };
            if (Array.isArray(selection.disabled) && selection.disabled.length > 0) {
              entry.disabled = selection.disabled;
            }
            mutableRes.selection.push(entry);
          }
        }

        const patch: ObjectRecord = {};
        const before = src;
        const after = asObjectRecord(childProxy) || {};
        const keys = computeSet.size ? Array.from(new Set<string>([...computeSet, ...Object.keys(after)])) : Object.keys(after);
        for (const k of keys) {
          if (k === inverseField) continue;
          if (after[k] !== before[k]) patch[k] = after[k];
        }
        if (Object.keys(patch).length) {
          if (src.Id != null) patch.Id = src.Id;
          else patch.pos = i;
          rowPatches.push(patch);
          Object.assign(src, patch);
        }

        const childrenOfThis = parentToChildren.get(pathKey) || [];
        if (childrenOfThis.length && depth + 1 <= PREVIEW_CASCADE_MAX_DEPTH) {
          for (const nextPath of childrenOfThis) {
            await cascadeCollection(childMeta, src, nextPath, depth + 1, labelForRow);
          }
        }
      }

      if (rowPatches.length) {
        if (!collectionPatch[pathKey]) collectionPatch[pathKey] = [];
        collectionPatch[pathKey].push(...rowPatches);
        affectedTopRoots.add(segs[0]);
      }
    };

    for (const rootKey of Array.from(collectionRootKeys).filter(k => k.indexOf('.') < 0)) {
      await cascadeCollection(meta, previewProxy, rootKey, 1, undefined);
    }

    if (affectedTopRoots.size && (opts?.withCompute ?? true)) {
      try {
        const g2 = meta.computeGraph;
        const affectedComputes = new Set<string>();
        if (g2) {
          const queue: string[] = [...affectedTopRoots];
          const seenFields = new Set<string>(queue);
          while (queue.length) {
            const src = queue.shift()!;
            const affectedCFs = g2.fastReverseDeps.get(src) || [];
            for (const cf of affectedCFs) {
              if (!affectedComputes.has(cf)) affectedComputes.add(cf);
              if (!seenFields.has(cf)) {
                seenFields.add(cf);
                queue.push(cf);
              }
              const deps = g2.computeScalarDeps?.get(cf) || [];
              for (const dep of deps) {
                if (!seenFields.has(dep)) {
                  seenFields.add(dep);
                  queue.push(dep);
                }
              }
            }
          }
        }

        const before: ObjectRecord = {};
        affectedComputes.forEach(k => (before[k] = previewProxy[k]));

        await recomputeModelMetadata(meta, previewProxy, affectedTopRoots, 'preview');

        const parentScalarPatch: ObjectRecord = {};
        const recomputedSet = new Set<string>();
        affectedComputes.forEach(k => {
          const nv = previewProxy[k];
          if (nv !== before[k]) {
            parentScalarPatch[k] = nv;
            recomputedSet.add(k);
          }
        });
        if (Object.keys(parentScalarPatch).length) {
          const baseVal = mutableRes.value ?? {};
          mutableRes.value = { ...baseVal, ...parentScalarPatch };
          try {
            const exist = new Set<string>(mutableRes.computeRecomputed || []);
            recomputedSet.forEach(f => exist.add(f));
            mutableRes.computeRecomputed = Array.from(exist);
          } catch {
            // ignore
          }
        }
      } catch {
        // ignore
      }
    }

    if (Object.keys(collectionPatch).length) {
      const baseVal = mutableRes.value ?? {};
      mutableRes.value = { ...baseVal, __collectionPatch: collectionPatch };
    }
  } catch {
    // ignore
  }
}
