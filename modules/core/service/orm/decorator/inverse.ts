// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import type { ModelCtor } from '../metadata/field';

export function Inverse<TModel extends BaseModel>(field: Extract<keyof TModel, string>): MethodDecorator {
  return function (target: Object, propertyKey: string | symbol, descriptor: PropertyDescriptor) {
    const method = String(propertyKey || '').trim();
    if (!method) {
      throw new Error('@Inverse requires a method name');
    }
    if (!descriptor || typeof descriptor.value !== 'function') {
      throw new Error(`@Inverse(${String(field)}) must decorate an instance method`);
    }
    if (descriptor.value.length !== 0) {
      throw new Error(`@Inverse(${String(field)}) method must be parameterless`);
    }

    const fieldName = String(field || '').trim();
    if (!fieldName) {
      throw new Error('@Inverse requires a target field name');
    }

    const ctor = target.constructor as ModelCtor<BaseModel>;
    const prev = MetadataStorage.instance.getModelMetadata(ctor);
    const inverseHandlers = new Map(prev.inverseHandlers || []);
    inverseHandlers.set(fieldName, {
      field: fieldName,
      method,
    });

    MetadataStorage.instance.setModelMetadata(ctor, {
      ...prev,
      type: ctor,
      inverseHandlers,
    });
  };
}
