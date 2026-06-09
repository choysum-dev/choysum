// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend change-tracking proxy utilities.
 *
 * Tracks model object changes and produces update data in Updateable format.
 * Supports automatic conversion of relation array mutations into RelationOperations payloads.
 */

import type { Tracked, TrackedAPI } from './types';
import { isObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../utils/types';

function cloneValue<T>(value: T): T {
  if (value instanceof Date) {
    return new Date(value.getTime()) as T;
  }

  if (Array.isArray(value)) {
    return value.map(item => cloneValue(item)) as T;
  }

  if (isObjectRecord(value)) {
    const out: ObjectRecord = {};
    for (const [key, item] of Object.entries(value)) {
      out[key] = cloneValue(item);
    }
    return out as T;
  }

  return value;
}

function isEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (a == null || b == null) return false;

  if (a instanceof Date && b instanceof Date) {
    return a.getTime() === b.getTime();
  }

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    for (let index = 0; index < a.length; index += 1) {
      if (!isEqual(a[index], b[index])) return false;
    }
    return true;
  }

  if (isObjectRecord(a) && isObjectRecord(b)) {
    const recordA = a as ObjectRecord;
    const recordB = b as ObjectRecord;
    const keysA = Object.keys(recordA);
    const keysB = Object.keys(recordB);
    if (keysA.length !== keysB.length) return false;
    for (const key of keysA) {
      if (!Object.prototype.hasOwnProperty.call(recordB, key)) return false;
      if (!isEqual(recordA[key], recordB[key])) return false;
    }
    return true;
  }

  return false;
}

/**
 * Change-tracking proxy helper class.
 */
export class TrackedModel<T extends ObjectRecord> implements TrackedAPI {
  private _original: T;
  private _draft: T;
  private _changes: ObjectRecord = {};
  private _touchedRelations: Set<string> = new Set();

  constructor(data: T) {
    this._original = cloneValue(data);
    this._draft = cloneValue(data);

    return this.createProxy();
  }

  /**
   * Creates the main proxy object.
   */
  private createProxy(): T & TrackedModel<T> {
    const self = this;

    const handler: ProxyHandler<TrackedModel<T>> = {
      get(target: TrackedModel<T>, prop: string | symbol): unknown {
        if (prop === 'getChanges' || prop === 'popChanges' || prop === 'hasChanges' || prop === 'resetChanges') {
          return Reflect.get(target, prop).bind(target);
        }

        if (typeof prop === 'string' && prop in target._draft) {
          const value = target._draft[prop];

          if (Array.isArray(value)) {
            return target.createArrayProxy(prop, value, () => target.syncRelationChange(prop));
          }

          if (isObjectRecord(value)) {
            return target.createObjectProxy(prop, value, () => target.syncFieldChange(prop));
          }

          return value;
        }

        return Reflect.get(target, prop);
      },

      set(target: TrackedModel<T>, prop: string | symbol, value: unknown): boolean {
        if (typeof prop === 'string' && !prop.startsWith('_')) {
          const draft = target._draft as ObjectRecord;
          draft[prop] = cloneValue(value);
          if (Array.isArray(draft[prop])) {
            target.syncRelationChange(prop);
          } else {
            target.syncFieldChange(prop);
          }
          return true;
        }

        return Reflect.set(target, prop, value);
      },
    };

    return new Proxy(this, handler) as T & TrackedModel<T>;
  }

  private createObjectProxy(fieldKey: string, value: ObjectRecord, onMutate: () => void): ObjectRecord {
    const handler: ProxyHandler<ObjectRecord> = {
      get: (obj: ObjectRecord, prop: string | symbol): unknown => {
        const next = Reflect.get(obj, prop);
        if (Array.isArray(next)) {
          return this.createArrayProxy(fieldKey, next, onMutate);
        }
        if (isObjectRecord(next)) {
          return this.createObjectProxy(fieldKey, next, onMutate);
        }
        return next;
      },

      set: (obj: ObjectRecord, prop: string | symbol, nextValue: unknown): boolean => {
        const result = Reflect.set(obj, prop, cloneValue(nextValue));
        onMutate();
        return result;
      },

      deleteProperty: (obj: ObjectRecord, prop: string | symbol): boolean => {
        const result = Reflect.deleteProperty(obj, prop);
        onMutate();
        return result;
      },
    };

    return new Proxy(value, handler);
  }

  private createArrayProxy(fieldKey: string, value: unknown[], onMutate: () => void): unknown[] {
    const mutatingMethods = new Set(['push', 'pop', 'shift', 'unshift', 'splice', 'sort', 'reverse']);
    const handler: ProxyHandler<unknown[]> = {
      get: (array: unknown[], prop: string | symbol): unknown => {
        const next = Reflect.get(array, prop);

        if (typeof prop === 'string' && mutatingMethods.has(prop) && typeof next === 'function') {
          return (...args: unknown[]) => {
            const result = next.apply(
              array,
              args.map(arg => cloneValue(arg))
            );
            onMutate();
            return result;
          };
        }

        if (typeof prop === 'string' && /^\d+$/.test(prop) && isObjectRecord(next)) {
          return this.createObjectProxy(fieldKey, next, onMutate);
        }

        return next;
      },

      set: (array: unknown[], prop: string | symbol, nextValue: unknown): boolean => {
        const result = Reflect.set(array, prop, cloneValue(nextValue));
        onMutate();
        return result;
      },

      deleteProperty: (array: unknown[], prop: string | symbol): boolean => {
        const result = Reflect.deleteProperty(array, prop);
        onMutate();
        return result;
      },
    };

    return new Proxy(value, handler);
  }

  private syncFieldChange(fieldKey: string): void {
    const current = this._draft[fieldKey as keyof T];
    if (isEqual(this._original[fieldKey as keyof T], current)) {
      delete this._changes[fieldKey];
      return;
    }
    this._changes[fieldKey] = cloneValue(current);
  }

  private syncRelationChange(relationKey: string): void {
    const original = Array.isArray(this._original[relationKey as keyof T]) ? (this._original[relationKey as keyof T] as unknown[]) : [];
    const current = Array.isArray(this._draft[relationKey as keyof T]) ? (this._draft[relationKey as keyof T] as unknown[]) : [];
    if (isEqual(original, current)) {
      this._touchedRelations.delete(relationKey);
      return;
    }
    this._touchedRelations.add(relationKey);
  }

  /**
   * Returns the collected changes without resetting the current change state.
   * @returns The accumulated change object.
   */
  getChanges(): Partial<ObjectRecord> {
    const updateData: ObjectRecord = cloneValue(this._changes);

    for (const relationKey of this._touchedRelations.values()) {
      const relationOps = this.processRelationChanges(relationKey);
      if (Object.keys(relationOps).length > 0) {
        updateData[relationKey] = relationOps;
      }
    }

    return updateData;
  }

  /**
   * Returns the collected changes and then resets the current change state.
   * @returns The accumulated change object.
   */
  popChanges(): Partial<ObjectRecord> {
    const changes = this.getChanges();
    this.resetChanges();
    return changes;
  }

  private processRelationChanges(relationKey: string): ObjectRecord {
    const originalRaw = this._original[relationKey as keyof T];
    const currentRaw = this._draft[relationKey as keyof T];
    const original = Array.isArray(originalRaw) ? originalRaw : [];
    const current = Array.isArray(currentRaw) ? currentRaw : [];

    const originalIds = new Set(this.extractIds(original));
    const currentIds = new Set(this.extractIds(current));

    const ops: ObjectRecord = {};

    const createItems = current.filter((item: unknown) => {
      const id = this.extractId(item);
      return !id || !originalIds.has(id);
    });

    if (createItems.length > 0) {
      ops.create = cloneValue(createItems);
    }

    const deleteIds = Array.from(originalIds).filter(id => !currentIds.has(id));
    if (deleteIds.length > 0) {
      ops.delete = deleteIds.map(id => ({ Id: id }));
    }

    const updateItems = current
      .filter((item: unknown) => {
        const id = this.extractId(item);
        return id && originalIds.has(id);
      })
      .filter((item: unknown) => {
        const id = this.extractId(item);
        const originalItem = original.find((o: unknown) => this.extractId(o) === id);
        return !isEqual(item, originalItem);
      });

    if (updateItems.length > 0) {
      ops.update = cloneValue(updateItems);
    }

    return ops;
  }

  /**
   * Extracts an ID from an object-like item.
   */
  private extractId(item: unknown): string | null {
    if (!item) return null;
    if (typeof item === 'string') return item;
    if (typeof item === 'object' && 'Id' in item) {
      const id = (item as { Id?: unknown }).Id;
      return typeof id === 'string' ? id : null;
    }
    return null;
  }

  /**
   * Extracts all IDs from an array.
   */
  private extractIds(items: unknown[]): string[] {
    return items.map(item => this.extractId(item)).filter((id): id is string => id !== null);
  }

  /**
   * Checks whether any unsaved changes are present.
   * @returns Whether unsaved changes exist.
   */
  hasChanges(): boolean {
    return Object.keys(this._changes).length > 0 || this._touchedRelations.size > 0;
  }

  resetChanges(): this {
    this._changes = {};
    this._touchedRelations.clear();
    this._draft = cloneValue(this._original);
    return this;
  }
}

/**
 * Creates a trackable object.
 * @param data Object to track.
 * @returns Proxy-wrapped trackable object.
 */
export function track<T extends ObjectRecord>(data: T): T & TrackedModel<T> {
  return new TrackedModel(data) as T & TrackedModel<T>;
}
