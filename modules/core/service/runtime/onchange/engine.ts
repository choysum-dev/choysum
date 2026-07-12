// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata, OnchangeHandlerMeta } from '../../orm/metadata/model';
import type BaseModel from '../../orm/model/model';
import { DEFAULT_LOOP_THRESHOLD, MAX_ITERATIONS } from './constants';
import { createOnchangeContext, normalizeMessages, normalizeCondition, normalizeSelection, applyValuePatch } from './context';
import type { OnchangeDraft, OnchangeMessage, OnchangeCondition, SelectionCondition, OnchangeRunOptions, OnchangeEngineResult } from './types';
import { createOnchangeDraft } from '../proxy';
import Decimal, { normalizeDecimalByMeta, isDecimal } from '@/core/utils/decimal';
import { MetadataStorage } from '../../orm/metadata/storage';
import type { FieldMetadata } from '../../orm/metadata/field';
import type { ObjectRecord } from '../../../utils/types';

type OnchangePatch = ObjectRecord;

type OnchangeHandlerReturn = {
  value?: unknown;
  message?: unknown;
  messages?: unknown;
  condition?: unknown;
  selection?: unknown;
};

function asOnchangePatch(value: unknown): OnchangePatch | undefined {
  return value !== null && typeof value === 'object' ? (value as OnchangePatch) : undefined;
}

function getPathRoot(path: string): string | undefined {
  return String(path).split('.').filter(Boolean)[0];
}

function getModelField(meta: ModelMetadata, key: string): FieldMetadata | undefined {
  return meta.fields?.get?.(key) as FieldMetadata | undefined;
}

function getModelFieldKeys(meta: ModelMetadata): string[] {
  return Array.from(meta.fields?.keys?.() || []);
}

function quantizeDecimalFieldValue(meta: ModelMetadata, field: string, value: unknown): unknown {
  const fieldMeta = getModelField(meta, field);
  if (fieldMeta?.type !== 'decimal') {
    return value;
  }

  try {
    return normalizeDecimalByMeta(fieldMeta, value) ?? value;
  } catch {
    return value;
  }
}

function quantizeDecimalPatch(meta: ModelMetadata, patch: OnchangePatch): OnchangePatch {
  const normalized: OnchangePatch = {};
  for (const [key, value] of Object.entries(patch)) {
    normalized[key] = quantizeDecimalFieldValue(meta, key, value);
  }
  return normalized;
}

function extractReferenceId(value: unknown): unknown {
  if (value === null || typeof value !== 'object') {
    return value;
  }

  const obj = value as { Id?: unknown; id?: unknown };
  return obj.Id ?? obj.id ?? null;
}

export class OnchangeEngine {
  static async run(meta: ModelMetadata, draft: OnchangeDraft, changed: string[], opts: OnchangeRunOptions): Promise<OnchangeEngineResult> {
    const messages: OnchangeMessage[] = [];
    const condition: OnchangeCondition[] = [];
    const selection: SelectionCondition[] = []; // Collect selection conditions.
    const appliedTopLevel: OnchangePatch = {};
    const emittedPatch: OnchangePatch = {};
    const touchedHandlers: string[] = [];
    const computeRecomputedSet = new Set<string>();
    const fieldChangeCount: Record<string, number> = {};
    const effectiveLoopThreshold = opts.loopThreshold ?? DEFAULT_LOOP_THRESHOLD;
    const max = opts.maxIterations ?? MAX_ITERATIONS;
    const stopOnError = opts.stopOnError ?? true;

    const initial = this.maybeNormalize(changed);
    const triggerIndex = this.buildTriggerIndex(meta.onchangeHandlers || []);

    // Pre-normalize ManyToOneRef and ManyToManyRef object values into Id or Id arrays.
    this.normalizeRelationRefs(meta, draft, changed);

    let pending = initial;
    let iter = 0;
    let hasError = false;

    // Collect write-through patches and quantize top-level decimal values.
    const patchSink = (path: string, value: unknown) => {
      const root = getPathRoot(path);
      if (root) {
        emittedPatch[path] = quantizeDecimalFieldValue(meta, root, value);
        return;
      }
      emittedPatch[path] = value;
    };

    // Compose proxies with writeProxy as the inner layer and previewProxy as the outer layer.
    const onchangeDraft = createOnchangeDraft(draft as unknown as BaseModel, {
      meta,
      triggers: new Set<string>(initial),
      // Allow all roots conservatively to avoid false positives.
      reads: new Set<string>(getModelFieldKeys(meta)),
      // Top-level fields already loaded on the current draft.
      loaded: new Set<string>(Object.keys(draft)),
      patchSink,
    });

    const pushNormalizedMessages = (payload: unknown) => {
      const normalized = normalizeMessages(payload);
      if (normalized.some(item => item.level === 'error')) {
        hasError = true;
      }
      messages.push(...normalized);
    };

    // Create the OnchangeContext with selection support.
    const ctx = createOnchangeContext({
      draft: onchangeDraft,
      changed: new Set<string>(changed),
      pushMessages: (m: OnchangeMessage[]) => {
        if (m?.some(x => x?.level === 'error')) hasError = true;
        messages.push(...m);
      },
      pushCondition: (q: OnchangeCondition[]) => condition.push(...q),
      // Collect selection conditions.
      pushSelection: (s: SelectionCondition[]) => selection.push(...s),
      applyValue: (v: OnchangePatch) => {
        // Apply the same top-level decimal quantization used by the sink.
        const normalized = quantizeDecimalPatch(meta, v);
        Object.assign(emittedPatch, normalized);
        applyValuePatch(draft, normalized);
      },
    });

    while (pending.size && iter < max && !(stopOnError && hasError)) {
      iter++;
      const handlers = this.collectHandlers(triggerIndex, pending);
      handlers.sort((a, b) => (a.priority || 100) - (b.priority || 100));

      const next = new Set<string>();
      let aborted = false;

      for (const handler of handlers) {
        touchedHandlers.push(handler.method);

        const beforeSnapshot: OnchangePatch = {};
        for (const key in draft) {
          beforeSnapshot[key] = draft[key];
        }

        try {
          const rawResult = this.invokeOnchangeHandler(handler, draft, onchangeDraft, ctx);
          const result = rawResult instanceof Promise ? await rawResult : rawResult;
          const ret = (result ?? {}) as OnchangeHandlerReturn;

          // Process returned payloads while keeping compatibility with legacy object returns.
          const valuePatch = asOnchangePatch(ret.value);
          if (valuePatch) {
            const normalized = quantizeDecimalPatch(meta, valuePatch);
            Object.assign(emittedPatch, normalized);
            applyValuePatch(draft, normalized);
          }
          if (ret.message) {
            pushNormalizedMessages(ret.message);
          }
          if (ret.messages) {
            pushNormalizedMessages(ret.messages);
          }
          if (ret.condition) {
            condition.push(...normalizeCondition(ret.condition));
          }
          // Process returned selection conditions.
          if (ret.selection) {
            selection.push(...normalizeSelection(ret.selection));
          }
        } catch (error: unknown) {
          hasError = true;
          const message = error instanceof Error ? error.message : String(error);
          messages.push({ level: 'error', message });
        }

        if (stopOnError && hasError) {
          aborted = true;
          break;
        }

        // Detect field changes and schedule the next round.
        for (const key in draft) {
          if (draft[key] !== beforeSnapshot[key]) {
            appliedTopLevel[key] = draft[key];
            fieldChangeCount[key] = (fieldChangeCount[key] || 0) + 1;
            const isSelfTrigger = pending.has(key);
            const withinLimit = fieldChangeCount[key] <= effectiveLoopThreshold;

            if (!isSelfTrigger && withinLimit) {
              next.add(key);
            } else if (!withinLimit) {
              // Only log loop suppression instead of returning it to the frontend.
              console.warn(`[OnchangeEngine] Loop suppressed on field "${key}" in model "${meta.name}"`, {
                field: key,
                model: meta.name,
                changeCount: fieldChangeCount[key],
                threshold: effectiveLoopThreshold,
              });
            }

            opts.collectField?.(key);
          }
        }
      }

      if (aborted) {
        pending = new Set();
      } else {
        pending = next;
      }
    }

    // Only log iteration overruns instead of returning them to the frontend.
    if (!hasError && pending.size && iter >= max) {
      console.warn(`[OnchangeEngine] Max iterations reached in model "${meta.name}"`, {
        model: meta.name,
        iterations: iter,
        maxIterations: max,
        pendingFields: [...pending],
      });
    }

    const anyError = hasError || messages.some(m => m?.level === 'error');

    // Run compute preview when it is enabled.
    if (!anyError && opts.withCompute && opts.computePreview) {
      const computeSeed = new Set<string>(initial);
      Object.keys(appliedTopLevel).forEach(key => computeSeed.add(key));
      Object.keys(emittedPatch).forEach(key => {
        const root = getPathRoot(key);
        if (root) computeSeed.add(root);
      });

      // As a preview fallback, convert decimal values on the entity and attached ManyToOne children to Decimal when possible.
      try {
        normalizeDecimalFields(meta, onchangeDraft as unknown as OnchangePatch);
      } catch {
        // ignore
      }

      const beforeCompute: OnchangePatch = {};
      for (const key in draft) {
        beforeCompute[key] = draft[key];
      }

      const computeFn = opts.computePreview;
      // Use the composed proxy for consistent read-only and patch behavior.
      await computeFn(onchangeDraft as unknown as OnchangeDraft, computeSeed);

      for (const key in draft) {
        if (draft[key] !== beforeCompute[key]) {
          appliedTopLevel[key] = draft[key];
          computeRecomputedSet.add(key);
          opts.collectField?.(key);
        }
      }
    }

    // Compute preview may have produced top-level values that did not pass through handler quantization.
    // Quantize top-level decimal fields one more time before assembling the final result.
    const quantizedTopLevel = quantizeDecimalPatch(meta, appliedTopLevel);

    const finalValue = anyError && stopOnError ? {} : { ...quantizedTopLevel, ...emittedPatch };

    return {
      value: Object.keys(finalValue).length > 0 ? (finalValue as OnchangeEngineResult['value']) : undefined,
      messages: messages.length > 0 ? messages : undefined,
      condition: condition.length > 0 ? condition : undefined,
      selection: selection.length > 0 ? selection : undefined,
      iterations: iter, // Always returned.
      touchedHandlers, // Always returned, even when empty.
      computeRecomputed: [...computeRecomputedSet], // Always returned.
    };
  }

  /**
   * Normalize changed field paths, including array-index paths.
   *
   * @remarks
   * - Convert 'Lines.0.Quantity' into 'Lines.Quantity' and 'Lines'.
   * - Keep other paths unchanged.
   */
  private static maybeNormalize(changed: string[]): Set<string> {
    const result = new Set<string>();
    let need = false;
    const indexPattern = /\.\d+(\.|$)/;

    for (const raw of changed) {
      if (indexPattern.test(raw)) {
        need = true;
        break;
      }
    }
    if (!need) {
      changed.forEach(c => c && result.add(c));
      return result;
    }

    for (const raw of changed) {
      if (!raw) continue;
      const segs = raw.split('.').filter(Boolean);
      if (segs.length >= 2 && /^\d+$/.test(segs[1])) {
        const clone = [...segs];
        clone.splice(1, 1);
        result.add(clone.join('.'));
        result.add(segs[0]);
      } else {
        result.add(raw);
      }
    }
    return result;
  }

  /**
   * Build the trigger index from field name to handler list.
   */
  private static buildTriggerIndex(list: OnchangeHandlerMeta[]) {
    const m = new Map<string, OnchangeHandlerMeta[]>();
    for (const h of list) {
      for (const t of h.triggers) {
        if (!m.has(t)) m.set(t, []);
        m.get(t)!.push(h);
      }
    }
    return m;
  }

  /**
   * Collect handlers that should run for the current changed-field set.
   */
  private static collectHandlers(index: Map<string, OnchangeHandlerMeta[]>, changed: Set<string>) {
    const result = new Set<OnchangeHandlerMeta>();
    for (const c of changed) {
      const hs = index.get(c);
      if (hs) hs.forEach(h => result.add(h));
    }
    return [...result];
  }

  /**
   * Invoke a single onchange handler with the calling convention determined by
   * its {@link OnchangeHandlerMeta.signature}.
   *
   * - `legacyCtx` (or absent): call with `(ctx)` — existing behavior.
   * - `instanceNoArgs`: call with no arguments — `this`-only.
   *
   * The return value may be a Promise; the caller is responsible for awaiting.
   */
  private static invokeOnchangeHandler(
    handler: OnchangeHandlerMeta,
    draft: OnchangeDraft,
    onchangeDraft: BaseModel,
    ctx: unknown
  ): unknown {
    const fn = draft[handler.method] as ((...args: unknown[]) => unknown) | undefined;
    if (typeof fn !== 'function') return undefined;

    const sig = handler.signature;
    if (sig === 'instanceNoArgs') {
      return fn.call(onchangeDraft);
    }
    // legacyCtx or unset — pass ctx for backward compatibility.
    return fn.call(onchangeDraft, ctx);
  }

  /**
   * Normalize relation references on the draft into plain Id or Id arrays so handlers do not see object-shaped references.
   * Only fields touched by the current changed paths are processed to reduce overhead.
   */
  private static normalizeRelationRefs(meta: ModelMetadata, draft: OnchangeDraft, changed: string[]) {
    // Only process top-level ManyToOneRef and ManyToManyRef fields.
    const roots = new Set<string>();
    for (const path of changed) {
      if (!path) continue;
      const root = getPathRoot(path);
      if (root) roots.add(root);
    }

    for (const root of roots) {
      const fieldMeta = getModelField(meta, root);
      if (!fieldMeta) continue;

      const value = draft[root];
      if (value == null) continue;

      if (fieldMeta.type === 'ManyToOneRef') {
        // Object to Id; keep strings as-is; preserve other shapes unchanged.
        const id = extractReferenceId(value);
        if (id != null) {
          draft[root] = id;
        }
      } else if (fieldMeta.type === 'ManyToManyRef') {
        // Array case: convert object items to Id and drop nullish values.
        if (Array.isArray(value)) {
          const ids = value.map(item => extractReferenceId(item)).filter(item => item != null);
          draft[root] = ids;
        }
      }
    }
  }
}

/* ======================= Preview Decimal fallback normalization ======================= */
function isObject(value: unknown): value is ObjectRecord {
  return value !== null && typeof value === 'object';
}

function normalizeDecimalFields(meta: ModelMetadata, entity: OnchangePatch) {
  for (const key of getModelFieldKeys(meta)) {
    const fieldMeta = getModelField(meta, key);
    if (!fieldMeta) continue;

    const value = entity[key];
    if (value == null) continue;

    if (fieldMeta.type === 'decimal') {
      const normalized = quantizeDecimalFieldValue(meta, key, value);
      entity[key] = normalized;

      if (!isDecimal(normalized)) {
        try {
          const source = isObject(normalized) ? ((normalized as { $bigdecimal?: unknown }).$bigdecimal ?? normalized) : normalized;
          entity[key] = new Decimal(source as Decimal.Value);
        } catch {
          // keep original
        }
      }
      continue;
    }

    if (fieldMeta.type === 'ManyToOne' && isObject(value)) {
      const ctor = fieldMeta.relation?.targetModel?.();
      if (ctor) {
        const nestedMeta = MetadataStorage.instance.getModelMetadata(ctor);
        normalizeDecimalFields(nestedMeta, value as OnchangePatch);
      }
    }
  }
}
