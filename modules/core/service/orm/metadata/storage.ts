// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, ModelCtor } from './field';
import { ModelMetadata } from './model';
import BaseModel from '../model/model';
import type { ConstraintMeta, EffectiveConstraintMeta } from './constraint';
import type { OnchangeHandlerMeta, EffectiveOnchangeMeta } from './model';
import type { ObjectRecord } from '../../../utils/types';

export class MetadataStorage {
  private static _instance: MetadataStorage;
  private models = new Map<ModelCtor, ModelMetadata>();

  // Add a merged-metadata cache. WeakMap avoids memory leaks.
  private mergedMetadataCache = new WeakMap<ModelCtor, ModelMetadata>();

  static get instance() {
    if (!this._instance) {
      this._instance = new MetadataStorage();
    }
    return this._instance;
  }

  private asRecord(value: unknown): ObjectRecord | undefined {
    return value && (typeof value === 'object' || typeof value === 'function') ? (value as ObjectRecord) : undefined;
  }

  private ensureStringArray(value: unknown): string[] {
    return Array.isArray(value) ? value.filter(item => item != null).map(item => String(item)) : [];
  }

  private getMethodName(value: unknown): string | undefined {
    const record = this.asRecord(value);
    if (!record || !record.method) {
      return undefined;
    }
    return String(record.method);
  }

  private normalizeConstraintHandler(value: unknown, existing?: ConstraintMeta): ConstraintMeta | undefined {
    const method = this.getMethodName(value);
    if (!method) {
      return undefined;
    }

    const record = this.asRecord(value)!;
    return {
      method,
      fields: [...new Set(this.ensureStringArray(record.fields ?? existing?.fields))],
      preview: typeof record.preview === 'boolean' ? record.preview : (existing?.preview ?? false),
      alwaysOnCreate: typeof record.alwaysOnCreate === 'boolean' ? record.alwaysOnCreate : (existing?.alwaysOnCreate ?? false),
      priority: typeof record.priority === 'number' ? record.priority : (existing?.priority ?? 100),
      isStatic: typeof record.isStatic === 'boolean' ? record.isStatic : (existing?.isStatic ?? false),
    };
  }

  private normalizeOnchangeHandler(value: unknown, existing?: OnchangeHandlerMeta): OnchangeHandlerMeta | undefined {
    const method = this.getMethodName(value);
    if (!method) {
      return undefined;
    }

    const record = this.asRecord(value)!;
    const rawTriggers = Array.isArray(record.triggers) ? record.triggers : existing?.triggers || [];
    const rawReads = Array.isArray(record.reads) ? record.reads : existing?.reads;

    const handler: OnchangeHandlerMeta = {
      method,
      triggers: [...new Set(this.ensureStringArray(rawTriggers))],
      reads: rawReads ? [...new Set(this.ensureStringArray(rawReads))] : undefined,
      priority: typeof record.priority === 'number' ? record.priority : (existing?.priority ?? 100),
    };

    // Only attach signature when a valid value is present so the raw metadata
    // shape stays unchanged for handlers that never declared one.
    const sig =
      typeof record.signature === 'string' && (record.signature === 'legacyCtx' || record.signature === 'instanceNoArgs')
        ? record.signature
        : existing?.signature;
    if (typeof sig === 'string') {
      handler.signature = sig;
    }

    return handler;
  }

  private clearStaticMetadataCache(value: unknown): void {
    const record = this.asRecord(value);
    if (record?.metadata !== undefined) {
      delete record.metadata;
    }
  }

  private createInitialMetadata<T extends BaseModel>(target: ModelCtor<T>): ModelMetadata {
    return {
      name: target.name,
      modelName: '',
      fullModelName: '',
      className: target.name,
      tableName: () => '',
      type: target,
      fields: new Map(),
      services: new Map(),
    };
  }

  private mergeConstraintHandlers(existing: ConstraintMeta[] | undefined, incoming: unknown): ConstraintMeta[] | undefined {
    if (!Array.isArray(incoming)) {
      return existing;
    }

    const merged: Record<string, ConstraintMeta> = {};
    (existing || []).forEach(handler => {
      const normalized = this.normalizeConstraintHandler(handler);
      if (normalized) {
        merged[normalized.method] = normalized;
      }
    });

    incoming.forEach(handler => {
      const method = this.getMethodName(handler);
      if (!method) return;
      const normalized = this.normalizeConstraintHandler(handler, merged[method]);
      if (normalized) {
        merged[method] = normalized;
      }
    });

    return Object.values(merged);
  }

  private mergeOnchangeHandlers(existing: OnchangeHandlerMeta[] | undefined, incoming: unknown): OnchangeHandlerMeta[] | undefined {
    if (!Array.isArray(incoming)) {
      return existing;
    }

    const merged: Record<string, OnchangeHandlerMeta> = {};
    (existing || []).forEach(handler => {
      const normalized = this.normalizeOnchangeHandler(handler);
      if (normalized) {
        merged[normalized.method] = normalized;
      }
    });

    incoming.forEach(handler => {
      const method = this.getMethodName(handler);
      if (!method) return;
      const normalized = this.normalizeOnchangeHandler(handler, merged[method]);
      if (normalized) {
        merged[method] = normalized;
      }
    });

    return Object.values(merged);
  }

  // Use explicit builder-style merging for stable ModelMetadata fields only.
  // This avoids implicit shape expansion from dynamic deep merges.
  private mergeModelMetadata(target: ModelMetadata, patch: Partial<ModelMetadata>): void {
    if (!patch) return;

    if (patch.name !== undefined) target.name = patch.name;
    if (patch.modelName !== undefined) target.modelName = patch.modelName;
    if (patch.fullModelName !== undefined) target.fullModelName = patch.fullModelName;
    if (patch.application !== undefined) target.application = patch.application;
    if (patch.className !== undefined) target.className = patch.className;
    if (patch.tableName !== undefined) target.tableName = patch.tableName;
    if (patch.type !== undefined) target.type = patch.type;
    if (patch.orderBy !== undefined) target.orderBy = patch.orderBy;
    if (patch.softDelete !== undefined) target.softDelete = patch.softDelete;
    if (patch.companyScoped !== undefined) target.companyScoped = patch.companyScoped;
    if (patch.autoMigrate !== undefined) target.autoMigrate = patch.autoMigrate;
    if (patch.readonly !== undefined) target.readonly = patch.readonly;
    if (patch.parentField !== undefined) target.parentField = patch.parentField;
    if (patch.computeGraph !== undefined) target.computeGraph = patch.computeGraph;

    if (patch.fields instanceof Map) {
      patch.fields.forEach((fieldMeta, fieldName) => {
        target.fields.set(fieldName, fieldMeta);
      });
    }

    if (patch.services instanceof Map) {
      patch.services.forEach((serviceMeta, serviceName) => {
        target.services.set(serviceName, serviceMeta);
      });
    }

    const mergedConstraintHandlers = this.mergeConstraintHandlers(target.constraintHandlers, patch.constraintHandlers);
    if (mergedConstraintHandlers) {
      target.constraintHandlers = mergedConstraintHandlers;
    }

    const mergedOnchangeHandlers = this.mergeOnchangeHandlers(target.onchangeHandlers, patch.onchangeHandlers);
    if (mergedOnchangeHandlers) {
      target.onchangeHandlers = mergedOnchangeHandlers;
    }
  }

  public setModelMetadata<T extends BaseModel>(target: ModelCtor<T>, metadata: Partial<ModelMetadata>): void {
    // Ensure the target already has a metadata record.
    if (!this.models.has(target)) {
      const initial = this.createInitialMetadata(target);
      this.mergeModelMetadata(initial, metadata);
      this.models.set(target, initial);
    } else {
      const existingMetadata = this.models.get(target)!;
      this.mergeModelMetadata(existingMetadata, metadata);
    }

    // Important: clear caches for this class and all subclasses.
    // Updating base-class metadata affects every inheriting subclass.
    this.clearCacheForClassAndSubclasses(target);
  }

  /**
   * Clear metadata caches for the specified class and its subclasses.
   *
   * Update notes:
   * - Also clear the BaseModel.metadata static property.
   * - Ensure the reverse index (computeGraph.reverseComputeIndex) is invalidated with the metadata.
   * - Recursively clear caches for all subclasses, including static properties.
   *
   * @private
   */
  private clearCacheForClassAndSubclasses<T extends BaseModel>(target: ModelCtor<T>): void {
    // 1. Clear the merged metadata cache for the current class.
    this.mergedMetadataCache.delete(target);

    // NEW: Clear the static metadata cache, including computeGraph.reverseComputeIndex.
    // This is the static property cache used by BaseModel.getMetadata().
    this.clearStaticMetadataCache(target);

    // 2. Find and clear caches for every subclass.
    // Note: this is an O(n) operation, but model counts are usually small, so the impact is limited.
    for (const [cls] of this.models.entries()) {
      // Check whether cls is a subclass of target.
      let proto = Object.getPrototypeOf(cls);
      while (proto && proto !== Object.prototype) {
        if (proto === target) {
          // This is a subclass, so clear its cache.
          this.mergedMetadataCache.delete(cls);

          // NEW: Clear the subclass static metadata property as well.
          this.clearStaticMetadataCache(cls);

          break;
        }
        proto = Object.getPrototypeOf(proto);
      }
    }
  }

  public getModelMetadata<T extends BaseModel>(target: ModelCtor<T>): ModelMetadata {
    // 1. Check the cache first.
    if (this.mergedMetadataCache.has(target)) {
      return this.mergedMetadataCache.get(target)!;
    }

    // 2. If nothing is cached, run the normal merge flow.

    // Ensure metadata exists for the current class first.
    if (!this.models.has(target)) {
      this.models.set(target, {
        name: target.name,
        modelName: '',
        fullModelName: '',
        className: target.name,
        tableName: () => '',
        type: target,
        fields: new Map(),
        services: new Map(),
      });
    }

    // Get the current class metadata.
    const metadata = this.models.get(target)!;

    // Copy the field map to avoid mutating the original data.
    const mergedFields = new Map(metadata.fields);
    const mergedOnchangeHandlers = [...(metadata.onchangeHandlers || [])];
    const mergedConstraintHandlers = [...(metadata.constraintHandlers || [])];

    function mergeByMethod<T extends { method: string }>(targetList: T[], parentList: T[] | undefined): void {
      if (!Array.isArray(parentList) || parentList.length === 0) return;
      const existing = new Set(targetList.map(item => item.method));
      for (const item of parentList) {
        if (!existing.has(item.method)) {
          targetList.push({ ...item });
          existing.add(item.method);
        }
      }
    }

    // Walk the prototype chain.
    let currentPrototype = Object.getPrototypeOf(target);

    // Keep prototype traversal simple to avoid stopping too early.
    while (currentPrototype && currentPrototype !== Object.prototype) {
      const parentCtor = typeof currentPrototype === 'function' ? (currentPrototype as ModelCtor) : undefined;

      // Only process prototypes with constructors and registered metadata.
      if (parentCtor && this.models.has(parentCtor)) {
        const parentMetadata = this.models.get(parentCtor)!;

        // Merge fields without overriding fields already defined by the subclass.
        if (parentMetadata.fields instanceof Map) {
          parentMetadata.fields.forEach((fieldMeta, fieldName) => {
            if (!mergedFields.has(fieldName)) {
              mergedFields.set(fieldName, { ...fieldMeta });
            }
          });
        }

        mergeByMethod(mergedOnchangeHandlers, parentMetadata.onchangeHandlers);
        mergeByMethod(mergedConstraintHandlers, parentMetadata.constraintHandlers);
      }

      // Continue walking up the prototype chain.
      currentPrototype = Object.getPrototypeOf(currentPrototype);
    }

    // Cache the merged result.
    const result: ModelMetadata = {
      ...metadata,
      fields: mergedFields,
      onchangeHandlers: mergedOnchangeHandlers,
      constraintHandlers: mergedConstraintHandlers,
    };

    // Inject ParentPath field metadata, including compute and dependency wiring.
    const pf = result.parentField;
    if (pf && !result.fields.has('ParentPath')) {
      const pathFieldMeta: FieldMetadata = {
        name: 'ParentPath',
        type: 'varchar',
        column: {
          size: 1000,
          index: true,
          compute: {
            expr: (self: unknown) => {
              const selfRecord = this.asRecord(self);
              const id = String(selfRecord?.Id ?? '');
              // Id is expected to exist before create; if it is unexpectedly missing, return an empty string instead of writing '/'.
              if (!id) return '';

              const parentRef = selfRecord?.[pf];
              const parentPathRaw = this.asRecord(parentRef)?.ParentPath;
              const parentPath = typeof parentPathRaw === 'string' ? parentPathRaw : undefined;
              // Cycle detection: parent paths must not contain the current Id segment.
              if (parentPath && parentPath.includes(`${id}/`)) {
                throw new Error(`Cycle detected: ${id} cannot be assigned as a child of its own descendant`);
              }

              // Materialized path: <parentPath><selfId>/.
              return `${parentPath ?? ''}${id}/`;
            },
            deps: ['Id', pf, `${pf}.ParentPath`] as never[], // Include Id as a dependency.
          },
        } as unknown as FieldMetadata['column'],
      };

      result.fields.set('ParentPath', pathFieldMeta);
    }

    this.mergedMetadataCache.set(target, result);
    return result;
  }

  /**
   * Resolves the effective constraint handlers for a model by walking the
   * prototype chain from child to root:
   *
   * - Child classes take precedence for the **same method name** (override).
   * - New method names introduced by a child are added alongside inherited ones (extend).
   * - If a child does **not** re-decorate a parent constraint, the parent's
   *   handler is inherited as-is (reuse).
   *
   * Handlers are returned sorted by priority then by method name.
   */
  public getEffectiveConstraints<T extends BaseModel>(target: ModelCtor<T>): EffectiveConstraintMeta[] {
    // Ensure metadata is initialized for the target class.
    this.getModelMetadata(target);

    const methodMap = new Map<string, EffectiveConstraintMeta>();
    let current: unknown = target;

    while (current && current !== Object.prototype) {
      if (typeof current === 'function') {
        const currentCtor = current as ModelCtor;
        if (!this.models.has(currentCtor)) {
          current = Object.getPrototypeOf(current);
          continue;
        }

        const ownMetadata = this.models.get(currentCtor)!;
        const source =
          String(ownMetadata.fullModelName || '').trim() ||
          String(ownMetadata.modelName || '').trim() ||
          String(ownMetadata.name || '').trim() ||
          String(currentCtor.name || '').trim() ||
          'unknown';

        const ownHandlers = Array.isArray(ownMetadata.constraintHandlers) ? ownMetadata.constraintHandlers : [];
        for (const handler of ownHandlers) {
          const method = String(handler?.method || '').trim();
          if (!method || methodMap.has(method)) {
            // Child class takes precedence for same method name.
            continue;
          }

          methodMap.set(method, {
            method,
            fields: Array.isArray(handler.fields) ? [...new Set(handler.fields.map(v => String(v || '').trim()).filter(Boolean))] : [],
            preview: !!handler.preview,
            alwaysOnCreate: !!handler.alwaysOnCreate,
            priority: typeof handler.priority === 'number' ? handler.priority : 100,
            isStatic: !!handler.isStatic,
            source,
          });
        }
      }

      current = Object.getPrototypeOf(current);
    }

    return Array.from(methodMap.values()).sort((left, right) => left.priority - right.priority || left.method.localeCompare(right.method));
  }

  public getEffectiveOnchange<T extends BaseModel>(target: ModelCtor<T>): EffectiveOnchangeMeta[] {
    this.getModelMetadata(target);

    const methodMap = new Map<string, EffectiveOnchangeMeta>();
    let current: unknown = target;

    while (current && current !== Object.prototype) {
      if (typeof current === 'function') {
        const currentCtor = current as ModelCtor;
        if (!this.models.has(currentCtor)) {
          current = Object.getPrototypeOf(current);
          continue;
        }

        const ownMetadata = this.models.get(currentCtor)!;
        const source =
          String(ownMetadata.fullModelName || '').trim() ||
          String(ownMetadata.modelName || '').trim() ||
          String(ownMetadata.name || '').trim() ||
          String(currentCtor.name || '').trim() ||
          'unknown';

        const ownHandlers = Array.isArray(ownMetadata.onchangeHandlers) ? ownMetadata.onchangeHandlers : [];
        for (const handler of ownHandlers) {
          const method = String(handler?.method || '').trim();
          if (!method || methodMap.has(method)) {
            continue;
          }

          methodMap.set(method, {
            method,
            triggers: Array.isArray(handler.triggers) ? [...new Set(handler.triggers.map(v => String(v || '').trim()).filter(Boolean))] : [],
            priority: typeof handler.priority === 'number' ? handler.priority : 100,
            reads: Array.isArray(handler.reads) ? [...new Set(handler.reads.map(v => String(v || '').trim()).filter(Boolean))] : undefined,
            signature:
              typeof handler.signature === 'string' && (handler.signature === 'legacyCtx' || handler.signature === 'instanceNoArgs')
                ? handler.signature
                : 'legacyCtx',
            source,
          });
        }
      }

      current = Object.getPrototypeOf(current);
    }

    return Array.from(methodMap.values()).sort((left, right) => left.priority - right.priority || left.method.localeCompare(right.method));
  }

  /**
   * Clear all metadata caches.
   * Primarily used for tests and special cases.
   */
  public clearCache(): void {
    this.mergedMetadataCache = new WeakMap();
  }
}

export function getEffectiveConstraints<T extends BaseModel>(target: ModelCtor<T>): EffectiveConstraintMeta[] {
  const storage = MetadataStorage.instance;
  if (typeof storage.getEffectiveConstraints === 'function') {
    return storage.getEffectiveConstraints(target);
  }
  return MetadataStorage.prototype.getEffectiveConstraints.call(storage, target);
}

export function getEffectiveOnchange<T extends BaseModel>(target: ModelCtor<T>): EffectiveOnchangeMeta[] {
  const storage = MetadataStorage.instance;
  if (typeof storage.getEffectiveOnchange === 'function') {
    return storage.getEffectiveOnchange(target);
  }
  return MetadataStorage.prototype.getEffectiveOnchange.call(storage, target);
}
