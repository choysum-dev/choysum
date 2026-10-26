// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../orm/metadata/model';
import type BaseModel from '../../orm/model/model';
import { buildRelationAliasCandidates } from '../../orm/relation/relation_alias';
import { asObjectRecord } from '../../../utils/object';
import type { UnknownRecord } from '../../../utils/types';

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

function isOrmRelationFieldMeta(fm: unknown): boolean {
  const fieldMeta = asObjectRecord(fm);
  if (!fieldMeta?.relation) return false;
  const type = fieldMeta.type;
  return type === 'ManyToOne' || type === 'OneToMany' || type === 'ManyToMany';
}

/** Parse JSON-string relation payloads the same way EntityConverter does for $rel$ aliases. */
function parseJsonRelationValue(value: unknown): unknown {
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

/**
 * Read a preloaded relation from `$rel$…` aliases when the plain field key is absent.
 * Browse/Search leave collection relations under `$rel$Lines` until transport rematerializes them;
 * virtual @Compute handlers still read `this.Lines` via this proxy.
 */
function readPreloadedRelation(meta: ModelMetadata, entityRecord: UnknownRecord, fieldName: string): unknown {
  const fieldMeta = meta.fields?.get?.(fieldName);
  if (!isOrmRelationFieldMeta(fieldMeta)) return undefined;

  for (const alias of buildRelationAliasCandidates(fieldName)) {
    if (!Object.prototype.hasOwnProperty.call(entityRecord, alias)) continue;
    const raw = entityRecord[alias];
    if (raw === undefined) continue;
    return parseJsonRelationValue(raw);
  }
  return undefined;
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
      if (typeof key === 'string') {
        const preloaded = readPreloadedRelation(meta, entityRecord, key);
        const hasDirect = Object.prototype.hasOwnProperty.call(entityRecord, key);
        if (hasDirect) {
          const direct = entityRecord[key];
          // ManyToOne rows keep the FK scalar on the field key and the object under `$rel$`.
          // Prefer the preloaded object so related/compute paths like CategoryId.Name resolve.
          const directIsFkScalar =
            typeof direct === 'string' || typeof direct === 'number' || typeof direct === 'bigint';
          if (preloaded != null && directIsFkScalar) {
            return preloaded;
          }
          return direct;
        }
        if (preloaded !== undefined) return preloaded;
      }
      return Reflect.get(target, key, receiver);
    },

    set(_target, key, value, receiver) {
      if (typeof key === 'string') {
        let proto: unknown = target;
        while (proto && (typeof proto === 'object' || typeof proto === 'function')) {
          const desc = Object.getOwnPropertyDescriptor(proto as object, key);
          if (desc?.set) {
            return Reflect.set(target, key, value, receiver);
          }
          proto = Object.getPrototypeOf(proto as object);
        }

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
      if (typeof key === 'string') {
        if (Object.prototype.hasOwnProperty.call(entityRecord, key)) {
          return true;
        }
        if (readPreloadedRelation(meta, entityRecord, key) !== undefined) {
          return true;
        }
      }
      return Reflect.has(target, key);
    },

    ownKeys(_target) {
      const entityKeys = Reflect.ownKeys(entityRecord);
      const targetKeys = Reflect.ownKeys(target);
      return [...new Set([...entityKeys, ...targetKeys])];
    },

    getOwnPropertyDescriptor(_target, key) {
      if (typeof key === 'string') {
        if (Object.prototype.hasOwnProperty.call(entityRecord, key)) {
          return {
            configurable: true,
            enumerable: true,
            writable: true,
            value: entityRecord[key],
          };
        }
        const preloaded = readPreloadedRelation(meta, entityRecord, key);
        if (preloaded !== undefined) {
          return {
            configurable: true,
            enumerable: true,
            writable: true,
            value: preloaded,
          };
        }
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
