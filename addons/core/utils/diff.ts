// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import microdiff from 'microdiff';
import equal from 'fast-deep-equal';
import { isDecimalLike, decimalEqual } from './decimal';
import { asObjectRecord, hasOwnKey } from './object';
import type { ObjectRecord } from './types';

type Plain = ObjectRecord;

type DiffChange = {
  path: Array<string | number>;
  value?: unknown;
};

export type RelationType = 'OneToMany' | 'ManyToMany' | 'ManyToOne';
export type FieldsMeta = Record<string, { relation?: RelationType; type?: string }>;
export type RelationArrayOps = Partial<{
  create: ObjectRecord[];
  update: ObjectRecord[];
  delete: Array<{ Id: unknown }>;
  replace: ObjectRecord[];
}>;

// Explicit relation operation kinds.
export type RelationOpsKind = 'create' | 'update' | 'delete' | 'replace';

const toId = (x: unknown): unknown => {
  const record = asObjectRecord(x);
  return record ? (record.Id ?? null) : (x ?? null);
};

// Extracts an ID list from string values or { Id } objects.
function toIdList(v: unknown): string[] {
  if (!Array.isArray(v)) return [];
  return v
    .map(it => {
      if (it == null) return null;
      if (typeof it === 'string') return it;
      const record = asObjectRecord(it);
      if (record) return record.Id ?? record.id ?? null;
      return null;
    })
    .filter((x): x is string => !!x)
    .map(String);
}

function toArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

const isArrayPath = (p: (string | number)[]) => p.some(x => typeof x === 'number');
const stripClientKeys = (o: unknown): unknown => {
  const record = asObjectRecord(o);
  if (!record) return o;
  const { __cid, __rowKey, ...rest } = record;
  void __cid;
  void __rowKey;
  return rest;
};

// NEW: Detects values that may participate in Decimal-aware comparisons.
function isDecimalCandidate(v: unknown): boolean {
  const record = asObjectRecord(v);
  return isDecimalLike(v) || (record ? hasOwnKey(record, '$bigdecimal') : false);
}

/* ------------------------------------------------
 * Builds a single-record field patch while ignoring arrays, Id, and client temp keys.
 * - Compare top-level Decimal values first to avoid microdiff blind spots.
 * ------------------------------------------------ */
function objectPatch(orig: Plain, curr: Plain): Plain {
  const patch: Plain = {};
  const handledTopKeys = new Set<string>();

  // Compare top-level Decimal values first.
  const allTopKeys = new Set<string>([...Object.keys(orig ?? {}), ...Object.keys(curr ?? {})]);
  for (const k of allTopKeys) {
    if (k === 'Id' || k === '__cid' || k === '__rowKey') continue;
    const oa = (orig ?? {})[k];
    const ob = (curr ?? {})[k];

    // Only enter this branch when at least one side looks Decimal-like.
    if (isDecimalCandidate(oa) || isDecimalCandidate(ob)) {
      if (!decimalEqual(oa, ob)) {
        patch[k] = ob !== undefined ? ob : null;
      }
      handledTopKeys.add(k);
    }
  }

  // Let microdiff handle the remaining fields after excluding processed top-level keys.
  const changes = microdiff(orig ?? {}, curr ?? {}) as DiffChange[];
  for (const c of changes) {
    if (isArrayPath(c.path)) continue;

    const topKey = String(c.path[0]);
    if (handledTopKeys.has(topKey)) continue; // Already handled by the Decimal branch.

    // Collapse Decimal internals back to the top-level field and take the current value as a fallback.
    if (c.path.length > 1) {
      const oa = (orig ?? {})[topKey];
      const ob = (curr ?? {})[topKey];
      if (isDecimalLike(oa) || isDecimalLike(ob)) {
        patch[topKey] = ob !== undefined ? ob : null;
        handledTopKeys.add(topKey);
        continue;
      }
    }

    if (c.path.length === 1 && (topKey === 'Id' || topKey === '__cid' || topKey === '__rowKey')) continue;

    let target: Plain = patch;
    for (let i = 0; i < c.path.length - 1; i++) {
      const key = String(c.path[i]);
      target[key] = asObjectRecord(target[key]) ?? {};
      target = target[key] as Plain;
    }
    const leaf = String(c.path[c.path.length - 1]);
    target[leaf] = 'value' in c ? (c.value !== undefined ? c.value : null) : null;
  }
  return patch;
}

/* ------------------------------------------------
 * Normalizes child ManyToOne fields so only Id is kept and unchanged values are removed.
 * ------------------------------------------------ */
function normalizePatchForRelationUpdate(a: Plain, b: Plain, patch: Plain): Plain {
  const out: Plain = { ...patch };
  for (const key of Object.keys(out)) {
    const v = out[key];
    const cur = b?.[key];
    const looksLikeM2O =
      (asObjectRecord(v) ? hasOwnKey(asObjectRecord(v) as ObjectRecord, 'Id') : false) ||
      (asObjectRecord(cur) ? hasOwnKey(asObjectRecord(cur) as ObjectRecord, 'Id') : false);

    if (looksLikeM2O) {
      const newId = toId(cur ?? v);
      const oldId = toId(a?.[key]);
      if (oldId === newId) {
        delete out[key];
      } else {
        out[key] = newId ?? null;
      }
    }
  }
  return out;
}

/* ------------------------------------------------
 * Computes diffs for relation arrays.
 * kind: 'o2m' | 'm2m'
 * Output: RelationArrayOps (create/delete/update)
 * ------------------------------------------------ */
function diffArrayRelation(kind: 'o2m' | 'm2m', origArr: unknown[] = [], currArr: unknown[] = []): RelationArrayOps | undefined {
  const oById = new Map<unknown, unknown>();
  for (const it of origArr) {
    const id = toId(it);
    if (id != null) oById.set(id, it);
  }
  const cById = new Map<unknown, unknown>();
  for (const it of currArr) {
    const id = toId(it);
    if (id != null) cById.set(id, it);
  }

  // create: rows without Id.
  const createNoId = currArr
    .filter(it => toId(it) == null)
    .map(stripClientKeys)
    .map(asObjectRecord)
    .filter((item): item is ObjectRecord => Boolean(item));

  // delete: rows that existed before but are now missing.
  const oIds = new Set(oById.keys());
  const cIds = new Set(cById.keys());
  const del = [...oIds].filter(x => !cIds.has(x)).map(Id => ({ Id }));

  // m2m additions are represented as Id-only links.
  const addIdsAsCreate: ObjectRecord[] = kind === 'm2m' ? [...cIds].filter(x => !oIds.has(x)).map(Id => ({ Id })) : [];

  // update: same Id, different content.
  const update: ObjectRecord[] = [];
  for (const id of cById.keys()) {
    if (!oById.has(id)) continue;
    const a = (oById.get(id) ?? {}) as Plain;
    const b = (cById.get(id) ?? {}) as Plain;
    if (equal(a, b)) continue;
    const rawPatch = objectPatch(a, b);
    const patch = normalizePatchForRelationUpdate(a, b, rawPatch);
    if (Object.keys(patch).length) {
      const sanitized = asObjectRecord(stripClientKeys(patch)) ?? {};
      update.push({ Id: id, ...sanitized });
    }
  }

  const ops: RelationArrayOps = {};
  const create = [...createNoId, ...addIdsAsCreate];
  if (create.length) ops.create = create;
  if (del.length) ops.delete = del;
  if (update.length) ops.update = update;

  return Object.keys(ops).length ? ops : undefined;
}

/* ------------------------------------------------
 * Diffs ordinary non-relation fields.
 * - Compare top-level Decimal values first to avoid microdiff blind spots.
 * ------------------------------------------------ */
function diffNormalFields(original: Plain, current: Plain, relationKeys: Set<string>): Plain {
  const out: Plain = {};
  const handledTopKeys = new Set<string>();

  // Compare top-level Decimal values first.
  const allTopKeys = new Set<string>([...Object.keys(original ?? {}), ...Object.keys(current ?? {})]);
  for (const k of allTopKeys) {
    if (relationKeys.has(k)) continue;
    if (k === 'Id' || k === '__cid' || k === '__rowKey') continue;

    const oa = (original ?? {})[k];
    const ob = (current ?? {})[k];

    if (isDecimalCandidate(oa) || isDecimalCandidate(ob)) {
      if (!decimalEqual(oa, ob)) {
        out[k] = ob !== undefined ? ob : null;
      }
      handledTopKeys.add(k);
    }
  }

  // Use microdiff for the rest after excluding processed top-level keys.
  const changes = microdiff(original ?? {}, current ?? {}) as DiffChange[];
  for (const c of changes) {
    const topKey = String(c.path[0]);
    if (relationKeys.has(topKey)) continue;
    if (isArrayPath(c.path)) continue;
    if (handledTopKeys.has(topKey)) continue;

    // Collapse Decimal internals back to the top-level field as a fallback.
    if (c.path.length > 1) {
      const oa = (original ?? {})[topKey];
      const ob = (current ?? {})[topKey];
      if (isDecimalLike(oa) || isDecimalLike(ob)) {
        out[topKey] = ob !== undefined ? ob : null;
        handledTopKeys.add(topKey);
        continue;
      }
    }

    if (c.path.length === 1 && (topKey === 'Id' || topKey === '__cid' || topKey === '__rowKey')) continue;

    let target: Plain = out;
    for (let i = 0; i < c.path.length - 1; i++) {
      const key = String(c.path[i]);
      target[key] = asObjectRecord(target[key]) ?? {};
      target = target[key] as Plain;
    }
    const leaf = String(c.path[c.path.length - 1]);
    target[leaf] = 'value' in c ? (c.value !== undefined ? c.value : null) : null;
  }

  return out;
}

/* ------------------------------------------------
 * Builds the update payload.
 * ------------------------------------------------ */
export function buildUpdatePayload(original: Plain, current: Plain, fieldsMeta?: FieldsMeta): Plain {
  const payload: Plain = {};
  const allKeys = new Set<string>([...Object.keys(original ?? {}), ...Object.keys(current ?? {})]);

  const processed = new Map<string, RelationType | 'ManyToManyRef'>();

  // 1) Relation fields explicitly declared in metadata.
  for (const key of allKeys) {
    const fm = fieldsMeta?.[key];
    const rel = fm?.relation as RelationType | undefined;

    // ManyToManyRef: send ID lists and compare them after dedupe and sorting.
    if (fm?.type === 'ManyToManyRef') {
      const origIds = toIdList(original?.[key]);
      const currIds = toIdList(current?.[key]);
      const same = origIds.length === currIds.length && origIds.every(id => currIds.includes(id));
      if (!same) payload[key] = currIds;
      processed.set(key, 'ManyToManyRef');
      continue;
    }

    if (!rel) continue;

    if (rel === 'ManyToOne') {
      const oId = toId(original?.[key]);
      const cId = toId(current?.[key]);
      if (oId !== cId) payload[key] = cId ?? null;
      processed.set(key, rel);
      continue;
    }

    if (rel === 'OneToMany' || rel === 'ManyToMany') {
      const ops = diffArrayRelation(rel === 'OneToMany' ? 'o2m' : 'm2m', toArray(original?.[key]), toArray(current?.[key]));
      if (ops) payload[key] = ops;
      processed.set(key, rel);
      continue;
    }
  }

  // 2) Infer undeclared relation-like keys.
  for (const key of allKeys) {
    if (processed.has(key)) continue;
    if (fieldsMeta?.[key]?.relation) continue;

    const oVal = original?.[key];
    const cVal = current?.[key];

    if (Array.isArray(oVal) || Array.isArray(cVal)) {
      const ops = diffArrayRelation('o2m', toArray(oVal), toArray(cVal));
      if (ops) payload[key] = ops;
      processed.set(key, 'OneToMany');
      continue;
    }

    const looksLikeM2O = (!!oVal && typeof oVal === 'object' && 'Id' in oVal) || (!!cVal && typeof cVal === 'object' && 'Id' in cVal);
    if (looksLikeM2O) {
      const oId = toId(oVal);
      const cId = toId(cVal);
      if (oId !== cId) payload[key] = cId ?? null;
      processed.set(key, 'ManyToOne');
      continue;
    }
  }

  // 3) Ordinary fields.
  const relationKeys = new Set<string>([...processed.keys()]);
  Object.assign(payload, diffNormalFields(original, current, relationKeys));

  return payload;
}

/* =========================================================
 * Semantic diff support for onchange path collection.
 * ========================================================= */

export function normalizeForDiff(value: unknown, depth = 0, seen = new WeakSet<object>()): unknown {
  if (value === null || typeof value !== 'object') {
    if (typeof value === 'bigint') return `__bigint:${value.toString()}`;
    return value;
  }
  if (value instanceof Date) {
    return isNaN(value.getTime()) ? '__date:invalid' : `__date:${value.toISOString()}`;
  }
  const valueRecord = asObjectRecord(value);
  if (valueRecord?.sd && valueRecord?.decimalPlaces && typeof valueRecord.toString === 'function') {
    try {
      return `__decimal:${valueRecord.toString()}`;
    } catch {
      return '__decimal:err';
    }
  }
  if (seen.has(value as object) || depth > 6) return '__cycle';
  seen.add(value as object);

  if (Array.isArray(value)) {
    return value.map(v => normalizeForDiff(v, depth + 1, seen));
  }

  const out: ObjectRecord = {};
  for (const k of Object.keys(valueRecord ?? {}).sort()) {
    out[k] = normalizeForDiff((valueRecord as ObjectRecord)[k], depth + 1, seen);
  }
  return out;
}

export interface ChangedPathOptions {
  includeTopLevel?: boolean;
  includeFullPath?: boolean;
  normalizeArrayIndex?: boolean;
  pruneRelationChildren?: boolean;
  fieldsMeta?: FieldsMeta;
  collapseFinal?: boolean;
}

function buildRelationSet(meta?: FieldsMeta): Set<string> {
  const set = new Set<string>();
  if (!meta) return set;
  for (const [k, v] of Object.entries(meta)) {
    if (v?.relation) set.add(k);
  }
  return set;
}

function collapseParentPaths(paths: Iterable<string>): string[] {
  const arr = Array.from(new Set(paths)).filter(Boolean).sort();
  const keep: string[] = [];
  for (const p of arr) {
    // Skip when a parent path is already present.
    if (keep.some(parent => p !== parent && p.startsWith(parent + '.'))) continue;
    // Remove existing children when this path becomes the parent.
    for (let i = keep.length - 1; i >= 0; i--) {
      if (keep[i] !== p && keep[i].startsWith(p + '.')) keep.splice(i, 1);
    }
    keep.push(p);
  }
  return keep;
}

/**
 * Collects changed paths with relation-aware pruning.
 */
export function collectChangedPaths(original: Plain, current: Plain, opts?: ChangedPathOptions): Set<string> {
  const {
    includeTopLevel = true,
    includeFullPath = true,
    normalizeArrayIndex = true,
    pruneRelationChildren = true,
    fieldsMeta,
    collapseFinal = true,
  } = opts || {};

  const relSet = pruneRelationChildren ? buildRelationSet(fieldsMeta) : new Set<string>();

  const normOrig = normalizeForDiff(original ?? {});
  const normCurr = normalizeForDiff(current ?? {});
  const left = (Array.isArray(normOrig) ? normOrig : (asObjectRecord(normOrig) ?? {})) as Array<unknown> | ObjectRecord;
  const right = (Array.isArray(normCurr) ? normCurr : (asObjectRecord(normCurr) ?? {})) as Array<unknown> | ObjectRecord;
  const diffs = microdiff(left, right) as DiffChange[];
  const out = new Set<string>();

  for (const d of diffs) {
    if (!d?.path?.length) continue;
    const segs = d.path.map(p => String(p));
    if (!segs.length) continue;

    const top = segs[0];

    const isRelationTop = relSet.has(top);
    if (isRelationTop && pruneRelationChildren) {
      // Relation fields keep only the top-level path so the change is never dropped.
      out.add(top);
      continue;
    }

    if (includeTopLevel && top) out.add(top);
    if (includeFullPath) out.add(segs.join('.'));

    if (normalizeArrayIndex) {
      if (segs.length >= 3 && /^\d+$/.test(segs[1])) {
        const cloned = [...segs];
        cloned.splice(1, 1);
        out.add(cloned.join('.'));
        out.add(top);
      } else if (segs.length >= 2 && /^\d+$/.test(segs[1])) {
        out.add(top);
      }
    }
  }

  if (!collapseFinal) {
    return out;
  }
  // Collapse parent and child paths once more as a final safety net.
  const collapsed = collapseParentPaths(out);
  return new Set(collapsed);
}
