// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../orm/metadata/model';
import type BaseModel from '../../orm/model/model';
import { asObjectRecord } from '../../../utils/object';
import type { UnknownRecord } from '../../../utils/types';

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

export function resolveInstanceHandler(meta: ModelMetadata, field: string, methodName: string, decoratorName: string): (this: unknown) => unknown {
  const method = String(methodName || '').trim();
  if (!method) {
    throw new Error(`${decoratorName} handler is missing method name: ${modelLabel(meta)}.${field}`);
  }

  const prototypeRecord = asObjectRecord(meta.type?.prototype);
  const fn = prototypeRecord?.[method];
  if (typeof fn !== 'function') {
    throw new Error(`${decoratorName} handler not found: ${modelLabel(meta)}.${field} -> ${method}`);
  }

  return fn as (this: unknown) => unknown;
}

export function createEntityBackedModelInstance(meta: ModelMetadata, entity: UnknownRecord): BaseModel {
  const entityRecord = asObjectRecord(entity) || {};
  const prototype = (meta.type && typeof meta.type === 'function' ? meta.type.prototype : undefined) || Object.prototype;
  const target = Object.create(prototype) as object;

  return new Proxy(target, {
    get(_target, key, receiver) {
      if (typeof key === 'string' && Object.prototype.hasOwnProperty.call(entityRecord, key)) {
        return entityRecord[key];
      }
      return Reflect.get(target, key, receiver);
    },

    set(_target, key, value, receiver) {
      if (typeof key === 'string') {
        entityRecord[key] = value;
        return true;
      }
      return Reflect.set(target, key, value, receiver);
    },

    deleteProperty(_target, key) {
      if (typeof key === 'string') {
        return delete entityRecord[key];
      }
      return Reflect.deleteProperty(target, key);
    },

    has(_target, key) {
      if (typeof key === 'string' && Object.prototype.hasOwnProperty.call(entityRecord, key)) {
        return true;
      }
      return Reflect.has(target, key);
    },

    ownKeys(_target) {
      const entityKeys = Reflect.ownKeys(entityRecord);
      const targetKeys = Reflect.ownKeys(target);
      return [...new Set([...entityKeys, ...targetKeys])];
    },

    getOwnPropertyDescriptor(_target, key) {
      if (typeof key === 'string' && Object.prototype.hasOwnProperty.call(entityRecord, key)) {
        return {
          configurable: true,
          enumerable: true,
          writable: true,
          value: entityRecord[key],
        };
      }
      return Reflect.getOwnPropertyDescriptor(target, key);
    },

    defineProperty(_target, key, descriptor) {
      if (typeof key === 'string') {
        if ('value' in descriptor) {
          entityRecord[key] = descriptor.value;
          return true;
        }
        return Reflect.defineProperty(target, key, descriptor);
      }
      return Reflect.defineProperty(target, key, descriptor);
    },
  }) as unknown as BaseModel;
}
