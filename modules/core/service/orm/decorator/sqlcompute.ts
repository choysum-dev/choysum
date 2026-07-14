// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ComputeDep, ModelCtor } from '../metadata/field';

export type SqlComputeOptions<TModel extends BaseModel = BaseModel> = {
  deps?: Array<ComputeDep<TModel>>;
};

export function SqlCompute<TModel extends BaseModel>(field: Extract<keyof TModel, string>, options?: SqlComputeOptions<TModel>): MethodDecorator {
  return function (target: Object, propertyKey: string | symbol, descriptor: PropertyDescriptor) {
    const method = String(propertyKey || '').trim();
    if (!method) {
      throw new Error('@SqlCompute requires a method name');
    }
    if (!descriptor || typeof descriptor.value !== 'function') {
      throw new Error(`@SqlCompute(${String(field)}) must decorate an instance method`);
    }
    if (descriptor.value.length !== 0) {
      throw new Error(`@SqlCompute(${String(field)}) method must be parameterless`);
    }

    const fieldName = String(field || '').trim();
    if (!fieldName) {
      throw new Error('@SqlCompute requires a target field name');
    }

    const deps = Array.isArray(options?.deps) ? [...new Set(options.deps.map(dep => String(dep || '').trim()).filter(Boolean))] : undefined;

    const ctor = target.constructor as ModelCtor<BaseModel>;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const sqlComputeHandlers = new Map(prev.sqlComputeHandlers || []);
    const fields = new Map(prev.fields || []);

    // SqlCompute fields are always virtual — strip any column metadata.
    const existing = fields.get(fieldName);
    if (existing && existing.column != null) {
      fields.set(fieldName, { ...existing, column: undefined });
    }

    sqlComputeHandlers.set(fieldName, {
      field: fieldName,
      method,
      deps,
    });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
      sqlComputeHandlers,
    });
  };
}
