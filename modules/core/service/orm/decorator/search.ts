// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ModelCtor } from '../metadata/field';

export function Search<TModel extends BaseModel>(field: Extract<keyof TModel, string>): MethodDecorator {
  return function (target: Object, propertyKey: string | symbol, descriptor: PropertyDescriptor) {
    const method = String(propertyKey || '').trim();
    if (!method) {
      throw new Error('@Search requires a method name');
    }
    if (!descriptor || typeof descriptor.value !== 'function') {
      throw new Error(`@Search(${String(field)}) must decorate an instance method`);
    }
    if (descriptor.value.length !== 0) {
      throw new Error(`@Search(${String(field)}) method must be parameterless`);
    }

    const fieldName = String(field || '').trim();
    if (!fieldName) {
      throw new Error('@Search requires a target field name');
    }

    const ctor = target.constructor as ModelCtor<BaseModel>;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const searchHandlers = new Map(prev.searchHandlers || []);
    searchHandlers.set(fieldName, {
      field: fieldName,
      method,
    });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      searchHandlers,
    });
  };
}
