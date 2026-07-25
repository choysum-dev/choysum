// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ComputeDep, ModelCtor } from '../metadata/field';

export type ComputeOptions<TModel extends BaseModel = BaseModel> = {
  deps: Array<ComputeDep<TModel>>;
  store?: boolean;
  searchable?: boolean;
};

export function Compute<TModel extends BaseModel>(field: Extract<keyof TModel, string>, options: ComputeOptions<TModel>): MethodDecorator {
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

    const ctor = target.constructor as ModelCtor<BaseModel>;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const computeHandlers = new Map(prev.computeHandlers || []);
    const fields = new Map(prev.fields || []);

    if (options?.store === false) {
      const existing = fields.get(fieldName);
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
      store: options?.store !== false,
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
