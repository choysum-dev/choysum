// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Watcher } from './watcher';
import { Dep } from './dep';
import { Entity } from '../../orm/repository';
import BaseModel from '../../orm/model/model';
import type { InstantiableModelCtor } from '../../orm/model/types';
import { MetadataStorage, ModelMetadata, FieldMetadata, ManyToOneMetadata, OneToManyMetadata, ManyToManyMetadata } from '../../orm/metadata';
import { buildRelationAliasCandidates, REL_ALIAS_PREFIX } from '../../orm/relation/relation_alias';
import { MODEL_SYMBOLS } from './symbols';
import { FieldSelection } from '../../orm/repository';

// Track relation-array mutations.
import { RelationArrayMethod, RelationChangeOperation, RelationChangesCollection } from '../../orm/relation/types';

// Unified Decimal normalization entrypoint.
import { normalizeDecimalByMeta } from '@/core/utils/decimal';
import { asObjectRecord, hasOwnKey } from '@/core/utils/object';
import type { UnknownRecord } from '@/core/utils/types';

/**
 * Creates proxy-wrapped model instances.
 */
export interface ProxyFactory {
  /**
   * Builds the proxy instance.
   */
  create(): BaseModel;
}

// Model-level summary cache for relation and computed fields.
const MODEL_SUMMARY_CACHE = new WeakMap<Function, { relationKeys: Set<string>; computedKeys: Set<string> }>();

type OrmRelationFieldMetadata = FieldMetadata & {
  type: 'ManyToOne' | 'OneToMany' | 'ManyToMany';
  relation: NonNullable<FieldMetadata['relation']>;
};

function isOrmRelationFieldMeta(f: FieldMetadata | undefined): f is OrmRelationFieldMetadata {
  if (!f?.relation) return false;
  return f.type === 'ManyToOne' || f.type === 'OneToMany' || f.type === 'ManyToMany';
}

function getModelSummary(meta: ModelMetadata, ctor: Function) {
  let summary = MODEL_SUMMARY_CACHE.get(ctor);
  if (summary) return summary;

  const relationKeys = new Set<string>();
  const computedKeys = new Set<string>();

  for (const [k, f] of meta.fields as Map<string, FieldMetadata>) {
    if (isOrmRelationFieldMeta(f)) relationKeys.add(k);
    // Compute metadata is now object-shaped, so look for f.column.compute.expr.
    if (f.column?.compute?.expr) computedKeys.add(k);
  }

  summary = { relationKeys, computedKeys };
  MODEL_SUMMARY_CACHE.set(ctor, summary);
  return summary;
}

function hydrateRelatedModel<T extends BaseModel>(
  ModelCtor: InstantiableModelCtor<T> | undefined,
  entity: UnknownRecord,
  fields?: FieldSelection<T>
): T | undefined {
  if (!ModelCtor) return undefined;
  const factoryToken = (ModelCtor as unknown as { FACTORY_TOKEN?: symbol }).FACTORY_TOKEN as symbol;
  const instance = new ModelCtor(factoryToken, entity as Entity, fields) as T;
  return new ModelProxyFactory<T>(instance, entity as Entity, fields).create();
}

/**
 * ModelProxyFactory wraps model instances with relation loading, computed fields,
 * Decimal normalization, and change tracking behavior.
 */
export class ModelProxyFactory<T extends BaseModel> implements ProxyFactory {
  private computedWatchers: Map<string, Watcher<T>> = new Map();
  private deps: Map<string, Dep> = new Map();
  private originalValues: WeakMap<T, Map<string, unknown>> = new WeakMap();
  private fieldCache: Map<string, FieldMetadata> = new Map();
  private relationCache: Map<string, unknown> = new Map();

  private readonly relationArrayMethods = new Set<string>(Object.values(RelationArrayMethod) as string[]);
  private relationChanges: WeakMap<T, Map<string, RelationChangeOperation[]>> = new WeakMap();
  private symbolHandlers?: Map<symbol, unknown>;
  private summary: { relationKeys: Set<string>; computedKeys: Set<string> };
  private proxyRef?: T;

  private target: T;
  private entity: Entity;
  private meta: ModelMetadata;
  private fields: FieldSelection<T> | undefined;

  constructor(target: T, entity: Entity, fields?: FieldSelection<T>) {
    this.target = target;
    this.entity = entity;
    this.meta = MetadataStorage.instance.getModelMetadata(target.constructor as InstantiableModelCtor<BaseModel>);
    this.fields = fields;

    this.originalValues.set(this.target, new Map());
    this.summary = getModelSummary(this.meta, target.constructor as Function);
  }

  /**
   * Creates the proxy-wrapped model instance.
   */
  create(): T {
    const proxyHandler: ProxyHandler<T> = {
      get: (target: T, key: string | symbol) => this.handlePropertyGet(target, key),
      set: (target: T, key: string | symbol, value: unknown) => {
        if (typeof key === 'string') {
          return this.handlePropertySet(target, key, value);
        }
        return Reflect.set(target, key, value);
      },
    };

    const proxy = new Proxy(this.target, proxyHandler) as T;
    this.proxyRef = proxy;
    return proxy;
  }

  /**
   * Clears proxy-local caches and watcher state.
   */
  public cleanup(): void {
    this.computedWatchers.clear();
    this.deps.clear();
    this.originalValues.delete(this.target);
    this.fieldCache.clear();
    this.relationCache.clear();
    this.relationChanges.delete(this.target);
  }

  private handlePropertyGet(target: T, key: string | symbol): unknown {
    // Symbol-backed helper methods.
    if (typeof key === 'symbol') {
      const fn = this.methodMap[key];
      if (fn) {
        if (!this.symbolHandlers) this.symbolHandlers = new Map();
        let bound = this.symbolHandlers.get(key);
        if (!bound) {
          bound = fn.bind(null, target);
          this.symbolHandlers.set(key, bound);
        }
        return bound;
      }
      return Reflect.get(target, key);
    }

    // Relation fields, computed fields, and dependency subscriptions.
    if (typeof key === 'string') {
      if (this.summary.relationKeys.has(key)) {
        const relationResult = this.handleRelation(target, key);
        if (relationResult !== undefined) return relationResult;
      }
      if (this.summary.computedKeys.has(key)) {
        const computedResult = this.handleComputed(target, key);
        if (computedResult !== undefined) return computedResult;
      }
      if (Dep.target) {
        this.ensureDep(key).depend();
      }

      // Lazily normalize scalar decimal reads without writing back, so read-only access stays side-effect free.
      const fm = this.getFieldMeta(key);
      if (fm?.type === 'decimal') {
        const raw = Reflect.get(target, key);
        if (raw == null) return raw;
        const normalized = normalizeDecimalByMeta(fm, raw);
        return normalized ?? raw;
      }
    }

    return Reflect.get(target, key);
  }

  private handlePropertySet(target: T, key: string, value: unknown): boolean {
    const fieldMeta = this.getFieldMeta(key);

    // Disallow setting computed fields directly.
    if (fieldMeta?.column?.compute?.expr) {
      throw new Error(`Cannot set computed property "${key}". It is read-only.`);
    }

    // Quantize decimal writes consistently with metadata scale and rounding rules.
    const valueToSet = fieldMeta?.type === 'decimal' ? (normalizeDecimalByMeta(fieldMeta, value) ?? value) : value;

    this.trackValueChange(target, key, valueToSet);

    const result = Reflect.set(target, key, valueToSet);

    // Clear relation cache after a field changes to avoid returning stale objects.
    this.relationCache.clear();

    this.notifyChange(key);
    return result;
  }

  /**
   * Handle relation-property access.
   */
  private handleRelation(target: T, key: string): unknown {
    const fieldMeta = this.getFieldMeta(key);
    if (!isOrmRelationFieldMeta(fieldMeta)) return undefined;

    if (this.relationCache.has(key)) {
      const cachedResult = this.relationCache.get(key);
      const cachedRecord = asObjectRecord(cachedResult);
      if (Array.isArray(cachedResult) && cachedRecord?.__isRelationProxy !== true) {
        return this.createArrayProxy(target, key, cachedResult);
      }
      return cachedResult;
    }

    // Support multiple preloaded $rel$ aliases such as $rel$UserId, $rel$userId, $rel$user_id, and $rel$_user_id.
    const preloadedKeys = buildRelationAliasCandidates(key);

    let preloaded: unknown = undefined;
    const entityRecord = asObjectRecord(this.entity) ?? {};
    for (const k of preloadedKeys) {
      if (hasOwnKey(entityRecord, k)) {
        preloaded = entityRecord[k];
        break;
      }
    }

    const fieldType = fieldMeta.type;

    // Prefer preloaded JSON when available so no database access is triggered.
    if (preloaded !== undefined) {
      if (fieldType === 'ManyToOne') {
        const rel = fieldMeta.relation as ManyToOneMetadata<BaseModel>;
        const targetModel = rel?.targetModel?.();
        const preloadedRecord = asObjectRecord(preloaded);
        const hydrated =
          preloaded === null
            ? null
            : preloadedRecord
              ? (hydrateRelatedModel(targetModel as InstantiableModelCtor<BaseModel>, preloadedRecord, this.getFieldFields(key)) ?? preloaded)
              : preloaded;
        this.relationCache.set(key, hydrated);
        return hydrated;
      }

      if (fieldType === 'OneToMany' || fieldType === 'ManyToMany') {
        const rel =
          fieldType === 'OneToMany' ? (fieldMeta.relation as OneToManyMetadata<BaseModel>) : (fieldMeta.relation as ManyToManyMetadata<BaseModel, BaseModel>);
        const targetModel = rel?.targetModel?.();
        const fields = this.getFieldFields(key);
        const list =
          targetModel && Array.isArray(preloaded)
            ? preloaded.map(item => {
                const itemRecord = asObjectRecord(item);
                return itemRecord ? (hydrateRelatedModel(targetModel as InstantiableModelCtor<BaseModel>, itemRecord, fields) ?? item) : item;
              })
            : Array.isArray(preloaded)
              ? preloaded
              : [];
        const proxied = this.createArrayProxy(target, key, list);
        this.relationCache.set(key, proxied);
        return proxied;
      }
    }

    // Check whether the relation has already been assigned manually.
    const rawValue = Reflect.get(target, key);

    // Decide whether this is a valid loaded or manually assigned relation value.
    let isValid = false;
    if (fieldType === 'ManyToOne') {
      // Allow null or a BaseModel instance.
      isValid = rawValue === null || rawValue instanceof BaseModel;
    } else {
      // To-many values must be arrays.
      isValid = Array.isArray(rawValue);
    }

    if (isValid) {
      return undefined; // Return undefined so handlePropertyGet falls through to Reflect.get(target, key).
    }

    // No cache, no preload, and no manual assignment means access should fail.
    throw new Error(
      `Accessing unloaded relation "${key}" on model "${this.meta.type.name}". ` +
        `Use "await model.load('${key}')" or include it in "fields" (e.g. Browse/Search options) before access.`
    );
  }

  private createArrayProxy(target: T, relationKey: string, originalArray: unknown[]): unknown[] {
    const handler: ProxyHandler<unknown[]> = {
      get: (arr, prop) => {
        const value = Reflect.get(arr, prop);

        if (typeof prop !== 'string' || !this.relationArrayMethods.has(prop)) {
          return value;
        }

        return (...args: unknown[]) => {
          let snapshot: unknown[] | undefined;
          if (prop === 'splice' || prop === 'sort' || prop === 'reverse') {
            snapshot = [...arr];
          }

          this.trackRelationChange(target, relationKey, prop as RelationArrayMethod, args, snapshot);
          const ret = (value as (...innerArgs: unknown[]) => unknown).apply(arr, args);
          // Notify dependencies after relation-array mutations so local computes invalidate and collectRelationChanges stays up to date.
          this.notifyChange(relationKey);
          return ret;
        };
      },
      set: (arr, prop, value) => {
        if (prop === '__isRelationProxy') return true;

        if (typeof prop === 'string' && !isNaN(parseInt(prop, 10))) {
          this.trackRelationChange(target, relationKey, RelationArrayMethod.SET, [parseInt(prop, 10), value]);
          // Indexed assignment should also notify dependents.
          this.notifyChange(relationKey);
        }
        return Reflect.set(arr, prop, value);
      },
    };

    const proxy = new Proxy(originalArray, handler);
    Object.defineProperty(proxy, '__isRelationProxy', { value: true, enumerable: false, configurable: false });
    return proxy;
  }

  private trackRelationChange(target: T, relationKey: string, method: RelationArrayMethod, args: unknown[], snapshot?: unknown[]): void {
    if (!this.relationChanges.has(target)) {
      this.relationChanges.set(target, new Map());
    }
    const changes = this.relationChanges.get(target)!;

    if (!changes.has(relationKey)) {
      changes.set(relationKey, []);
    }
    const operations = changes.get(relationKey)!;

    const operation: RelationChangeOperation = {
      method,
      args,
      timestamp: Date.now(),
      snapshot,
    };

    operations.push(operation);
    // Do not notify here to avoid double notifications; array methods already notify after execution.
  }

  private collectRelationChanges(target: T): RelationChangesCollection {
    const changes = this.relationChanges.get(target);
    if (!changes) return {};

    const result: RelationChangesCollection = {};
    changes.forEach((operations, key) => {
      result[key] = [...operations];
    });

    return result;
  }

  private resetRelationChanges(target: T): void {
    const changes = this.relationChanges.get(target);
    if (changes) {
      changes.clear();
    }
  }

  private handleComputed(target: T, key: string): unknown {
    if (!this.summary.computedKeys.has(key)) return undefined;

    const fieldMeta = this.getFieldMeta(key);
    // Only proceed when compute.expr exists.
    if (!fieldMeta?.column?.compute?.expr) return undefined;

    // Optimization: for pristine objects, return the stored computed value directly.
    // This avoids recomputation failures in serialization-like flows when dependencies are not loaded.
    if (fieldMeta.column) {
      const isPristine = this.getChangedFields(target).length === 0 && Object.keys(this.collectRelationChanges(target)).length === 0;

      if (isPristine) {
        const storedValue = Reflect.get(target, key);
        if (storedValue !== undefined) {
          return storedValue;
        }
      }
    }

    try {
      if (!this.deps.get(key)) {
        this.deps.set(key, new Dep());
      }
      const dep = this.deps.get(key)!;
      dep.depend();

      if (!this.computedWatchers.get(key)) {
        const self = this.proxyRef ?? target;
        // Invoke column.compute.expr directly.
        this.computedWatchers.set(key, new Watcher<T>(self, options => fieldMeta.column!.compute!.expr(options.self), key));
      }

      return this.computedWatchers.get(key)!.get();
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : String(e);
      throw new Error(`Failed to compute property ${key}: ${message}`);
    }
  }

  private getFieldMeta(key: string): FieldMetadata | undefined {
    if (!this.fieldCache.has(key)) {
      const field = this.meta.fields.get(key);
      if (field) {
        this.fieldCache.set(key, field);
      }
    }
    return this.fieldCache.get(key);
  }

  private getFieldFields(key: string): string[] | undefined {
    if (!this.fields) {
      return undefined;
    }

    for (const field of this.fields) {
      if (typeof field === 'object' && field !== null && key in field) {
        const fieldRecord = asObjectRecord(field);
        const nested = fieldRecord?.[key];
        if (Array.isArray(nested)) {
          return nested.filter((item): item is string => typeof item === 'string');
        }
        return undefined;
      }
    }

    return undefined;
  }

  private notifyChange(key: string): void {
    this.deps.get(key)?.notify();
  }

  private trackValueChange(target: T, key: string, newValue: unknown): void {
    const originalValues = this.originalValues.get(target);
    if (!originalValues) return;

    const currentValue = Reflect.get(target, key);
    if (!originalValues.has(key) && currentValue !== newValue) {
      originalValues.set(key, currentValue);
    }
  }

  private getChangedFields(target: T): string[] {
    return Array.from(this.originalValues.get(target)?.keys() || []);
  }

  private getOriginalValue(target: T, field: string): unknown {
    return this.originalValues.get(target)?.get(field);
  }

  private hasChanged(target: T, field: string): boolean {
    return this.originalValues.get(target)?.has(field) || false;
  }

  private resetChanges(target: T): void {
    this.originalValues.get(target)?.clear();
  }

  private ensureDep(key: string): Dep {
    let dep = this.deps.get(key);
    if (!dep) {
      dep = new Dep();
      this.deps.set(key, dep);
    }
    return dep;
  }

  // Bind Symbol-backed helper methods.
  private readonly methodMap: Record<symbol, (target: T, ...args: unknown[]) => unknown> = {
    [MODEL_SYMBOLS.getChangedFields]: this.getChangedFields.bind(this) as (target: T, ...args: unknown[]) => unknown,
    [MODEL_SYMBOLS.getOriginalValue]: this.getOriginalValue.bind(this) as (target: T, ...args: unknown[]) => unknown,
    [MODEL_SYMBOLS.hasChanged]: this.hasChanged.bind(this) as (target: T, ...args: unknown[]) => unknown,
    [MODEL_SYMBOLS.resetChanges]: this.resetChanges.bind(this) as (target: T, ...args: unknown[]) => unknown,
    [MODEL_SYMBOLS.collectRelationChanges]: this.collectRelationChanges.bind(this) as (target: T, ...args: unknown[]) => unknown,
    [MODEL_SYMBOLS.resetRelationChanges]: this.resetRelationChanges.bind(this) as (target: T, ...args: unknown[]) => unknown,
  };
}
