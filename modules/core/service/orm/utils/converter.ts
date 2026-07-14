// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Entity } from '../repository';
import { MetadataStorage } from '../metadata/storage';
import { FieldMetadata, ManyToOneMetadata, OneToManyMetadata, ManyToManyMetadata } from '../metadata';
import BaseModel from '../model/model';
import type { InstantiableModelCtor } from '../model/types';
import { hydrateModel } from '../model/model_hydration';
import { buildRelationAliasCandidates, REL_ALIAS_PREFIX } from '../relation/relation_alias';
import { FieldSelection } from '../repository';
import { serialize } from '../../../utils/decimal';
import type { ObjectRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';

type InstantiableBaseModelCtor = InstantiableModelCtor<BaseModel>;
type RelationMetadataLike = ManyToOneMetadata<BaseModel> | OneToManyMetadata<BaseModel> | ManyToManyMetadata<BaseModel, BaseModel>;
type OrmRelationFieldMetadata = FieldMetadata & {
  type: 'ManyToOne' | 'OneToMany' | 'ManyToMany';
  relation: NonNullable<FieldMetadata['relation']>;
};

// Cache the set of public non-relation fields to speed up calls without an explicit fields list.
const NON_REL_PUBLIC_FIELD_CACHE = new WeakMap<InstantiableBaseModelCtor, string[]>();

// Compiled-plan cache: ModelCtor -> signature -> plan.
const PLAN_CACHE = new WeakMap<InstantiableBaseModelCtor, Map<string, CompiledPlan>>();

type ScalarConv = 'none' | 'date' | 'decimal' | 'bigint' | 'serialize';
type FieldOp =
  | {
      kind: 'scalar';
      key: string;
      conv: ScalarConv;
    }
  | {
      kind: 'relation';
      key: string;
      cardinality: 'one' | 'many';
      // Relation target.
      childCtor?: InstantiableBaseModelCtor;
      childPlan?: CompiledPlan;
    };

type CompiledPlan = {
  ctor: InstantiableBaseModelCtor;
  ops: FieldOp[];
  // Whether the plan is JSON-safe and avoids fields that require normalization.
  jsonSafe: boolean;
  // Whether the plan contains relations.
  hasRelations: boolean;
};

// Reuse the shared serializer so Decimal and BigInt encoding stays consistent.
// Fast path: return primitives directly, special-case Date and BigInt, and send objects through serialize.
function normalizeValueForTransport(value: unknown): unknown {
  if (value == null) return value;
  const t = typeof value;
  if (t === 'string' || t === 'number' || t === 'boolean') return value;
  if (t === 'bigint') return { $bigint: value.toString() };
  if (value instanceof Date) return value.toISOString();
  // Delegate arrays, Decimal values, and other objects to the shared serializer.
  return serialize(value);
}

function getRelationTargetCtor(relation: unknown): InstantiableBaseModelCtor | undefined {
  const relationRecord = asObjectRecord(relation);
  const targetModel = relationRecord?.targetModel;
  if (typeof targetModel !== 'function') return undefined;
  const resolved = targetModel();
  return typeof resolved === 'function' ? (resolved as InstantiableBaseModelCtor) : undefined;
}

function isOrmRelationFieldMeta(fm: unknown): fm is OrmRelationFieldMetadata {
  const fieldMeta = asObjectRecord(fm) as FieldMetadata | undefined;
  if (!fieldMeta?.relation) return false;
  return fieldMeta.type === 'ManyToOne' || fieldMeta.type === 'OneToMany' || fieldMeta.type === 'ManyToMany';
}

function assignField(instance: BaseModel, key: string, value: unknown): void {
  (instance as unknown as ObjectRecord)[key] = value;
}

export class EntityConverter {
  private static isPublicKey = (k: string) => /^[A-Z]/.test(k);

  private static hydrateRelationTarget<T extends BaseModel>(targetCtor: InstantiableModelCtor<T> | undefined, value: unknown): T | undefined {
    if (!targetCtor) return undefined;
    return hydrateModel(targetCtor, value as Entity);
  }

  private static parseJsonRelationValue(value: unknown): unknown {
    if (typeof value !== 'string') return value;
    const s = value.trim();
    if (!s) return value;
    const first = s[0];
    if (first !== '[' && first !== '{') return value;
    try {
      return JSON.parse(s);
    } catch {
      return value;
    }
  }

  private static getNonRelPublicFields<T extends BaseModel>(ctor: InstantiableModelCtor<T>): string[] {
    const cached = NON_REL_PUBLIC_FIELD_CACHE.get(ctor);
    if (cached) return cached;

    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const list: string[] = [];
    for (const [k, fm] of meta.fields.entries()) {
      if (!this.isPublicKey(k)) continue;
      if (isOrmRelationFieldMeta(fm)) continue;
      list.push(k);
    }
    NON_REL_PUBLIC_FIELD_CACHE.set(ctor, list);
    return list;
  }

  static entityToModel<T extends BaseModel>(instance: T, entity: Entity): void {
    const meta = MetadataStorage.instance.getModelMetadata(instance.constructor as InstantiableModelCtor<BaseModel>);

    // 1. Walk entity keys and handle directly matching fields, including relations.
    for (const [key, value] of Object.entries(entity)) {
      // Skip $rel$ preloaded keys here and handle them later while iterating metadata.
      if (typeof key === 'string' && key.startsWith(REL_ALIAS_PREFIX)) continue;

      const fm = meta.fields.get(key);
      if (!fm) continue;

      // Handle relation fields that are present directly on the entity, such as 'Roles': [...].
      if (isOrmRelationFieldMeta(fm)) {
        if (value === undefined) continue;
        const relValue = this.parseJsonRelationValue(value);

        if (relValue === null) {
          assignField(instance, key, null);
          continue;
        }
        if (fm.type === 'ManyToOne') {
          const targetCtor = getRelationTargetCtor(fm.relation);
          if (targetCtor) {
            assignField(instance, key, this.hydrateRelationTarget(targetCtor, relValue));
          }
        } else if (fm.type === 'OneToMany' || fm.type === 'ManyToMany') {
          const targetCtor = getRelationTargetCtor(fm.relation);
          if (targetCtor && Array.isArray(relValue)) {
            assignField(
              instance,
              key,
              relValue.map(v => this.hydrateRelationTarget(targetCtor, v) ?? v)
            );
          }
        }
        continue;
      }

      assignField(instance, key, value);
    }

    // 2. Walk relation fields from metadata and look for preloaded $rel$ aliases such as $rel$_roles.
    for (const [key, fm] of meta.fields) {
      if (!isOrmRelationFieldMeta(fm)) continue;

      // If the entity already contains the field directly, step 1 handled it.
      if (Object.prototype.hasOwnProperty.call(entity, key)) continue;

      const preloaded = this.getPreloadedRelationValue(entity, key);
      if (preloaded !== undefined) {
        const relValue = this.parseJsonRelationValue(preloaded);

        if (relValue === null) {
          assignField(instance, key, null);
          continue;
        }
        if (fm.type === 'ManyToOne') {
          const targetCtor = getRelationTargetCtor(fm.relation);
          if (targetCtor) {
            assignField(instance, key, this.hydrateRelationTarget(targetCtor, relValue));
          }
        } else if (fm.type === 'OneToMany' || fm.type === 'ManyToMany') {
          const targetCtor = getRelationTargetCtor(fm.relation);
          if (targetCtor && Array.isArray(relValue)) {
            assignField(
              instance,
              key,
              relValue.map(v => this.hydrateRelationTarget(targetCtor, v) ?? v)
            );
          }
        }
      }
    }
  }

  private static getPreloadedRelationValue(row: ObjectRecord, fieldName: string): unknown {
    const keys = buildRelationAliasCandidates(fieldName);
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i]!;
      const v = row[k];
      if (v !== undefined) return v;
    }
    return undefined;
  }

  // Some computed to-many fields are selected as direct columns (e.g. Childs with select.expr)
  // and may not carry a $rel$ alias. Normalize them to array payloads for transport.
  private static normalizeToManyRelationValue(raw: unknown): unknown[] | undefined {
    if (raw === undefined) return undefined;
    if (raw === null) return [];

    const parsed = this.parseJsonRelationValue(raw);
    if (Array.isArray(parsed)) return parsed;
    if (parsed == null) return [];

    if (typeof parsed === 'object') {
      const obj = parsed as ObjectRecord;

      for (const key of ['value', 'values', 'items']) {
        const arr = obj[key];
        if (Array.isArray(arr)) return arr;
      }

      const numericKeys = Object.keys(obj)
        .filter(k => /^\d+$/.test(k))
        .sort((a, b) => Number(a) - Number(b));
      if (numericKeys.length) {
        return numericKeys.map(k => obj[k]);
      }
    }

    return [];
  }

  // Normalize and stabilize FieldSelection so it can be used as a plan-cache key.
  private static canonicalizeFields(sel?: unknown[]): string {
    if (!sel || sel.length === 0) return '__ALL_PUBLIC__';
    const norm = (arr: unknown[]): unknown[] => {
      const out: unknown[] = [];
      for (const it of arr) {
        if (!it) continue;
        if (typeof it === 'string') {
          out.push(it);
        } else if (typeof it === 'object') {
          const record = asObjectRecord(it);
          if (!record) continue;
          const keys = Object.keys(record).sort();
          for (const k of keys) {
            const v = record[k];
            out.push({ [k]: Array.isArray(v) ? norm(v) : [] });
          }
        }
      }
      // Sort by key so equivalent selections share the same signature.
      return out.sort((a, b) => {
        const ka =
          typeof a === 'string'
            ? a
            : (() => {
                const record = asObjectRecord(a);
                return record ? Object.keys(record)[0] : '';
              })();
        const kb =
          typeof b === 'string'
            ? b
            : (() => {
                const record = asObjectRecord(b);
                return record ? Object.keys(record)[0] : '';
              })();
        return String(ka).localeCompare(String(kb));
      });
    };
    try {
      return JSON.stringify(norm(sel));
    } catch {
      // Fallback: keep the original shape, even if cache hit rate drops slightly.
      return JSON.stringify(sel);
    }
  }

  // Infer the conversion strategy for scalar fields.
  private static inferScalarConvByType(type: string | undefined): ScalarConv {
    switch (type) {
      case 'datetime':
        return 'date';
      case 'decimal':
        return 'decimal';
      // Handle bigint model types here; otherwise runtime BigInt values are normalized later.
      case 'bigint':
        return 'bigint';
      default:
        return 'none';
    }
  }

  // Compile the conversion plan recursively.
  private static compilePlan<T extends BaseModel>(ctor: InstantiableModelCtor<T>, fields?: FieldSelection<T> | unknown[]): CompiledPlan {
    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const ops: FieldOp[] = [];
    let jsonSafe = true;
    let hasRelations = false;

    const pushScalarOp = (key: string) => {
      const fm = meta.fields.get(key);
      if (!fm || isOrmRelationFieldMeta(fm)) return;
      const conv = this.inferScalarConvByType(fm.type);
      if (conv !== 'none') jsonSafe = false;

      ops.push({ kind: 'scalar', key, conv });
    };

    const pushRelationOp = (key: string, subSel?: unknown[]) => {
      const fm = meta.fields.get(key);
      if (!fm || !isOrmRelationFieldMeta(fm)) return;

      hasRelations = true;
      if (fm.type === 'ManyToOne') {
        const childCtor = getRelationTargetCtor(fm.relation);
        if (childCtor) {
          const childPlan = this.getOrBuildPlan(childCtor, subSel); // Recurse by default.
          if (!childPlan.jsonSafe) jsonSafe = false;
          ops.push({ kind: 'relation', key, cardinality: 'one', childCtor, childPlan });
        } else {
          // No target model is available, so fall back to serialize.
          jsonSafe = false;
          ops.push({ kind: 'relation', key, cardinality: 'one' });
        }
      } else if (fm.type === 'OneToMany' || fm.type === 'ManyToMany') {
        const childCtor = getRelationTargetCtor(fm.relation);
        if (childCtor) {
          const childPlan = this.getOrBuildPlan(childCtor, subSel);
          if (!childPlan.jsonSafe) jsonSafe = false;
          ops.push({ kind: 'relation', key, cardinality: 'many', childCtor, childPlan });
        } else {
          jsonSafe = false;
          ops.push({ kind: 'relation', key, cardinality: 'many' });
        }
      }
    };

    // Without an explicit fields list, use public non-relation fields by default.
    if (!fields || fields.length === 0) {
      const keys = this.getNonRelPublicFields(ctor);
      for (let i = 0; i < keys.length; i++) {
        const k = keys[i]!;
        pushScalarOp(k);
      }
      return { ctor, ops, jsonSafe, hasRelations };
    }

    // With an explicit fields list, preserve the caller's declared order.
    for (const item of fields as unknown[]) {
      if (!item) continue;
      if (typeof item === 'string') {
        if (!this.isPublicKey(item)) continue;
        const fm = meta.fields.get(item);
        if (!fm) continue;

        if (!isOrmRelationFieldMeta(fm)) {
          pushScalarOp(item);
        } else {
          // A relation name without a nested selection uses the child model default selection.
          pushRelationOp(item, undefined);
        }
        continue;
      }

      if (typeof item === 'object') {
        const record = asObjectRecord(item);
        if (!record) continue;
        const relName = Object.keys(record)[0];
        if (!relName) continue;
        if (!this.isPublicKey(relName)) continue;
        const subSel = record[relName];
        pushRelationOp(relName, Array.isArray(subSel) ? subSel : undefined);
      }
    }

    return { ctor, ops, jsonSafe, hasRelations };
  }

  private static getOrBuildPlan<T extends BaseModel>(ctor: InstantiableModelCtor<T>, fields?: FieldSelection<T> | unknown[]): CompiledPlan {
    let bySig = PLAN_CACHE.get(ctor);
    if (!bySig) {
      bySig = new Map<string, CompiledPlan>();
      PLAN_CACHE.set(ctor, bySig);
    }
    const sig = this.canonicalizeFields(fields as unknown[]);
    const cached = bySig.get(sig);
    if (cached) return cached;

    const plan = this.compilePlan(ctor, fields);
    bySig.set(sig, plan);
    return plan;
  }

  // Convert scalar values according to the compiled-plan conv mode.
  private static applyScalarConv(val: unknown, conv: ScalarConv): unknown {
    if (val === undefined) return val;
    switch (conv) {
      case 'none':
        // Still handle BigInt as a fallback.
        return typeof val === 'bigint' ? { $bigint: val.toString() } : val;
      case 'date':
        return val instanceof Date ? val.toISOString() : val;
      case 'decimal':
        return serialize(val);
      case 'bigint':
        return typeof val === 'bigint' ? { $bigint: val.toString() } : val;
      case 'serialize':
      default:
        return normalizeValueForTransport(val);
    }
  }

  // Execute the compiled plan for a single object.
  private static executePlan(plan: CompiledPlan, row: ObjectRecord): ObjectRecord {
    const out: ObjectRecord = {};

    // ManyToOne relations may sometimes carry only the scalar foreign key on the transport layer,
    // such as ParentId='xxx', while $rel$ParentId is empty. Fall back to the scalar value so null does not overwrite it.
    const scalarRelationFallback = (op: Extract<FieldOp, { kind: 'relation' }>): unknown => {
      if (op.cardinality !== 'one') return undefined;
      const scalar = row[op.key];
      if (scalar === undefined || scalar === null) return undefined;
      return op.childPlan ? { Id: scalar } : scalar;
    };

    // JSON-safe fast path: copy only scalar values and recursively execute JSON-safe child plans without normalization.
    if (plan.jsonSafe) {
      for (let i = 0; i < plan.ops.length; i++) {
        const op = plan.ops[i]!;
        if (op.kind === 'scalar') {
          const v = row[op.key];
          if (v !== undefined) out[op.key] = v;
        } else {
          // Relations: read from the $rel$ alias first, then fall back to the field name itself.
          let pre = this.getPreloadedRelationValue(row, op.key);
          if (op.cardinality === 'one') {
            const fallback = scalarRelationFallback(op);
            if (pre === undefined) {
              if (fallback !== undefined) out[op.key] = fallback;
              continue;
            }

            if (!op.childPlan || pre == null || typeof pre !== 'object') {
              out[op.key] = pre == null && fallback !== undefined ? fallback : pre;
            } else {
              out[op.key] = this.executePlan(op.childPlan, pre as unknown as ObjectRecord);
            }
            continue;
          }

          const manyRaw = this.normalizeToManyRelationValue(pre !== undefined ? pre : row[op.key]);
          if (manyRaw === undefined) continue;

          if (op.cardinality === 'many') {
            if (!op.childPlan) {
              out[op.key] = manyRaw.slice(); // Shallow copy is enough here.
            } else {
              const arr: unknown[] = new Array(manyRaw.length);
              for (let j = 0; j < manyRaw.length; j++) {
                const ch = manyRaw[j];
                arr[j] = ch && typeof ch === 'object' ? this.executePlan(op.childPlan, ch as ObjectRecord) : ch;
              }
              out[op.key] = arr;
            }
          }
        }
      }
      return out;
    }

    // General path: apply only the minimum normalization required by conv.
    for (let i = 0; i < plan.ops.length; i++) {
      const op = plan.ops[i]!;
      if (op.kind === 'scalar') {
        const raw = row[op.key];
        if (raw !== undefined) out[op.key] = this.applyScalarConv(raw, op.conv);
        continue;
      }

      const pre = this.getPreloadedRelationValue(row, op.key);
      if (op.cardinality === 'one') {
        const fallback = scalarRelationFallback(op);
        if (pre === undefined) {
          if (fallback !== undefined) out[op.key] = fallback;
          continue;
        }

        if (!op.childPlan) {
          out[op.key] = pre == null && fallback !== undefined ? fallback : normalizeValueForTransport(pre);
        } else if (pre == null || typeof pre !== 'object') {
          out[op.key] = pre == null && fallback !== undefined ? fallback : pre;
        } else {
          out[op.key] = this.executePlan(op.childPlan, pre as unknown as ObjectRecord);
        }
      } else {
        const manyRaw = this.normalizeToManyRelationValue(pre !== undefined ? pre : row[op.key]);
        if (manyRaw === undefined) continue;
        if (!op.childPlan) {
          out[op.key] = manyRaw.map(v => normalizeValueForTransport(v));
        } else {
          const arr: unknown[] = new Array(manyRaw.length);
          for (let j = 0; j < manyRaw.length; j++) {
            const ch = manyRaw[j];
            arr[j] = ch && typeof ch === 'object' ? this.executePlan(op.childPlan, ch as ObjectRecord) : ch;
          }
          out[op.key] = arr;
        }
      }
    }

    return out;
  }

  // Convert a query row into a client-facing plain object without building models or proxies.
  static entityToPlainObject<T extends BaseModel>(ctor: InstantiableModelCtor<T>, row: ObjectRecord, fields?: FieldSelection<T>): ObjectRecord {
    // Plan cache hit rates are high because the same list usually reuses the same plan.
    const plan = this.getOrBuildPlan(ctor, fields);
    return this.executePlan(plan, row);
  }

  static entityArrayToPlainObject<T extends BaseModel>(ctor: InstantiableModelCtor<T>, rows: ObjectRecord[], fields?: FieldSelection<T>): ObjectRecord[] {
    try {
      const plan = this.getOrBuildPlan(ctor, fields);
      const out = new Array(rows.length);
      for (let i = 0; i < rows.length; i++) {
        out[i] = this.executePlan(plan, rows[i]!);
      }
      return out;
    } catch (e) {
      throw e;
    }
  }

  /**
   * Convert a model instance into a plain object.
   * Rules:
   * - Exclude function-valued properties.
   * - Keep only properties whose names start with an uppercase letter.
   */
  static modelToPlainObject<T extends BaseModel>(model: T, fields?: FieldSelection<T>): ObjectRecord {
    const isPublicKey = (k: string) => /^[A-Z]/.test(k);
    const modelState = model as unknown as ObjectRecord;

    const serializeField = (val: unknown, subFields?: FieldSelection<BaseModel>): unknown => {
      if (val instanceof BaseModel) {
        return this.modelToPlainObject(val, subFields);
      }
      if (Array.isArray(val) && val.length && val[0] instanceof BaseModel) {
        return val.map(v => this.modelToPlainObject(v, subFields));
      }
      return val;
    };

    // When fields are not specified, output every uppercase-leading non-function property, recursively handling models and model arrays.
    if (!fields || fields.length === 0) {
      const out: ObjectRecord = {};
      for (const k of Object.getOwnPropertyNames(model)) {
        if (!isPublicKey(k)) continue;
        const v = modelState[k];
        if (typeof v === 'function') continue;
        out[k] = serializeField(v);
      }
      return out;
    }

    const out: ObjectRecord = {};

    for (const item of fields as unknown[]) {
      if (!item) continue;

      // Simple field or relation name.
      if (typeof item === 'string') {
        if (!isPublicKey(item)) continue;
        const v = modelState[item];
        if (v === undefined || typeof v === 'function') continue;
        out[item] = serializeField(v);
        continue;
      }

      // Nested relation-selection object: { RelationName: [ 'FieldA', { NestedRel: [...] } ] }
      if (typeof item === 'object') {
        const record = asObjectRecord(item);
        if (!record) continue;
        for (const relName of Object.keys(record)) {
          if (!isPublicKey(relName)) continue;
          const subSel = record[relName] as FieldSelection<BaseModel> | undefined;
          const v = modelState[relName];
          if (v === undefined || typeof v === 'function') continue;
          out[relName] = serializeField(v, subSel);
        }
      }
    }

    return out;
  }
}
