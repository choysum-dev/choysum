// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type { ModelMetadata } from '../metadata';
import type BaseModel from './model';
import { asObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../../utils/types';

function ensureMetadata(ModelCtor: unknown): ModelMetadata {
  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as unknown as typeof BaseModel);
  return meta;
}

class MissingDependencyError extends Error {
  constructor(fieldName: string, dependencyName: string) {
    super(`Field ${fieldName} references missing value ${dependencyName}`);
    this.name = 'MissingDependencyError';
  }
}

/**
 * Apply `@Field({ default })` column defaults (literal + dependency-aware functions).
 * Used by `BaseModel.DefaultGet` / the DefaultGet pipeline; Create must go through `ModelCtor.DefaultGet`.
 */
export async function applyFieldColumnDefaults<T>(ModelCtor: unknown, value: Partial<T>): Promise<Partial<T>> {
  const result: ObjectRecord = { ...(value as ObjectRecord) };
  const meta = ensureMetadata(ModelCtor);

  // Fields with pending defaults.
  const pendingFields = new Set<string>();
  const processingStack = new Set<string>(); // Used to detect circular dependencies.
  const processedFields = new Set<string>();

  // Collect fields that require defaults.
  meta.fields.forEach((field, key) => {
    if (result[key] === undefined && field?.column?.default !== undefined) {
      pendingFields.add(key);
    }
  });

  const processField = (fieldName: string) => {
    // Already filled via dependency resolution while another field was processed.
    if (processedFields.has(fieldName)) return;

    const field = meta.fields.get(fieldName);
    const defaultValue = field?.column?.default;
    if (defaultValue === undefined) return;

    // Literal default.
    if (typeof defaultValue !== 'function') {
      result[fieldName] = defaultValue;
      processedFields.add(fieldName);
      return;
    }

    // Function default: read dependencies through a Proxy.
    processingStack.add(fieldName);
    try {
      const proxy = new Proxy(result, {
        get(target, prop) {
          const propName = String(prop);

          // Detect mutual references before recursing so in-progress fields fail fast.
          if (processingStack.has(propName) && propName !== fieldName) {
            const circularPath = Array.from(processingStack).join(' → ') + ' → ' + propName + ' → ' + fieldName;
            throw new Error(`Circular dependency detected: ${fieldName} depends on in-progress field ${propName}; cycle path ${circularPath}`);
          }

          // If the dependency also has a default and is still pending, resolve it first to avoid reading undefined.
          if (pendingFields.has(propName) && !processedFields.has(propName) && propName !== fieldName) {
            processField(propName);
          }

          // Accessing an unknown dependency throws MissingDependencyError so the caller can skip the field default.
          if (!(propName in target) && !pendingFields.has(propName)) {
            throw new MissingDependencyError(fieldName, propName);
          }

          return asObjectRecord(target)?.[propName];
        },
      });

      try {
        result[fieldName] = defaultValue(proxy);
        processedFields.add(fieldName);
      } catch (error) {
        if (error instanceof MissingDependencyError) {
          // Missing dependency: skip this field default to preserve the existing behavior.
          delete result[fieldName];
          processedFields.add(fieldName);
          return;
        }
        throw error;
      }
    } finally {
      processingStack.delete(fieldName);
    }
  };

  // Resolve pending fields; processField no-ops when a dependency pass already filled the field.
  pendingFields.forEach(fieldName => {
    processField(fieldName);
  });

  return result as Partial<T>;
}

/**
 * @deprecated Prefer {@link applyFieldColumnDefaults}. Kept as a thin alias for one transition release.
 */
export class DefaultOperations {
  static async DefaultGet<T>(ModelCtor: unknown, value: Partial<T>): Promise<Partial<T>> {
    return applyFieldColumnDefaults(ModelCtor, value);
  }
}
