// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ComputeDep, ModelCtor } from '../metadata/field';
import type { ComputeRunAs } from '../metadata/compute';

export type ComputeOptions<TModel extends BaseModel = BaseModel> = {
  deps: Array<ComputeDep<TModel>>;
  store?: boolean;
  searchable?: boolean;
  runAs?: ComputeRunAs;
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

    const runAs = options?.runAs;
    if (runAs != null && runAs !== 'user' && runAs !== 'sudo') {
      throw new Error(`@Compute(${fieldName}) runAs must be user or sudo`);
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
      runAs,
    });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      fields,
      computeHandlers,
    });
  };
}
