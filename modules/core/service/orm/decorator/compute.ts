// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ComputeDep, ModelCtor } from '../metadata/field';

/**
 * Property keys whose values are arrays of BaseModel (OneToMany / ManyToMany style).
 */
export type CollectionRelationKeys<TModel> = Extract<
  {
    [K in keyof TModel]-?: NonNullable<TModel[K]> extends readonly (infer E)[] ? (E extends BaseModel ? K : never) : never;
  }[keyof TModel],
  string
>;

/** Keys that are not collection-of-model relations (scalars, ManyToOne, Ref string[], …). */
export type NonCollectionRelationKeys<TModel> = Exclude<Extract<keyof TModel, string>, CollectionRelationKeys<TModel>>;

/** Shared compute options; `store` defaults to true at runtime when omitted. */
export type ComputeOptions<TModel extends BaseModel = BaseModel> = {
  deps: Array<ComputeDep<TModel>>;
  store?: boolean;
  searchable?: boolean;
};

/**
 * Compute options for OneToMany / ManyToMany targets: must be virtual (no parent column).
 */
export type VirtualCollectionComputeOptions<TModel extends BaseModel = BaseModel> = {
  deps: Array<ComputeDep<TModel>>;
  store: false;
  searchable?: boolean;
};

const COLLECTION_RELATION_TYPES = new Set(['OneToMany', 'ManyToMany']);

/**
 * Registers a JS compute handler for a model field.
 *
 * Overloads: collection-of-model fields (O2M/M2M) require `store: false`; other fields keep
 * optional `store` (default true). Runtime also rejects persisted compute on O2M/M2M metadata.
 */
export function Compute<TModel extends BaseModel>(
  field: CollectionRelationKeys<TModel>,
  options: VirtualCollectionComputeOptions<TModel>
): MethodDecorator;
export function Compute<TModel extends BaseModel>(
  field: NonCollectionRelationKeys<TModel>,
  options: ComputeOptions<TModel>
): MethodDecorator;
export function Compute<TModel extends BaseModel>(
  field: Extract<keyof TModel, string>,
  options: ComputeOptions<TModel> | VirtualCollectionComputeOptions<TModel>
): MethodDecorator {
  return function (target: Object, propertyKey: string | symbol, descriptor: PropertyDescriptor) {
    const method = String(propertyKey || '').trim();
    if (!method) {
      throw new Error('@Compute requires a method name');
    }
    if (!descriptor || typeof descriptor.value !== 'function') {
      throw new Error(`@Compute(${String(field)}) must decorate an instance method`);
    }
    if (descriptor.value.length !== 0) {
      throw new Error(`@Compute(${String(field)}) method must be parameterless`);
    }

    const fieldName = String(field || '').trim();
    if (!fieldName) {
      throw new Error('@Compute requires a target field name');
    }

    const deps = Array.isArray(options?.deps) ? [...new Set(options.deps.map(dep => String(dep || '').trim()).filter(Boolean))] : [];
    if (!deps.length) {
      throw new Error(`@Compute(${fieldName}) deps must be a non-empty array`);
    }

    const optionsRecord = options as Record<string, unknown> | undefined;
    if (optionsRecord && Object.prototype.hasOwnProperty.call(optionsRecord, 'runAs')) {
      throw new Error(`@Compute(${fieldName}) runAs is removed; call BaseModel.sudo / withUser inside the method body`);
    }

    const store = options?.store !== false;

    const ctor = target.constructor as ModelCtor<BaseModel>;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const computeHandlers = new Map(prev.computeHandlers || []);
    const fields = new Map(prev.fields || []);
    const existing = fields.get(fieldName);
    const fieldType = existing?.type;

    if (fieldType && COLLECTION_RELATION_TYPES.has(fieldType) && store) {
      throw new Error(
        `@Compute(${fieldName}) ${fieldType} targets require store: false (collection fields have no parent column; use @SqlCompute for SQL projection)`
      );
    }

    if (options?.store === false) {
      if (existing && existing.column != null) {
        fields.set(fieldName, {
          ...existing,
          column: undefined,
        });
      }
    }

    computeHandlers.set(fieldName, {
      field: fieldName,
      method,
      deps,
      store,
      searchable: typeof options?.searchable === 'boolean' ? options.searchable : undefined,
    });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
      computeHandlers,
    });
  };
}
