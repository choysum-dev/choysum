// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from './storage';

/**
 * Result of resolving a model identifier to its constructor and display name.
 */
export type ResolvedEffectiveModel = {
  ctor: typeof BaseModel;
  model: string;
};

/**
 * Resolve a model constructor from an identifier string and derive its
 * canonical display name from metadata.
 *
 * Throws when the identifier is empty or no matching constructor is found.
 */
export function resolveEffectiveModel(identifier: string): ResolvedEffectiveModel {
  const key = String(identifier || '').trim();
  if (!key) {
    throw new Error('modelIdentifier cannot be empty');
  }

  const ctor = BaseModel.resolveModelConstructor(key);
  if (!ctor) {
    throw new Error(`model not found: ${key}`);
  }

  const meta = MetadataStorage.instance.getModelMetadata(ctor);
  const model =
    String(meta.fullModelName || '').trim() || String(meta.modelName || '').trim() || String(meta.name || '').trim() || String(ctor.name || '').trim() || key;

  return { ctor, model };
}

/**
 * Normalize a priority range filter from loose user input.
 */
export type NormalizedPriorityRange = {
  min: number | undefined;
  max: number | undefined;
};

export function normalizePriorityRange(options?: { minPriority?: number; maxPriority?: number }): NormalizedPriorityRange {
  const hasMin = typeof options?.minPriority === 'number' && Number.isFinite(options.minPriority);
  const hasMax = typeof options?.maxPriority === 'number' && Number.isFinite(options.maxPriority);
  return {
    min: hasMin ? Number(options?.minPriority) : undefined,
    max: hasMax ? Number(options?.maxPriority) : undefined,
  };
}

/**
 * Return whether an item's priority falls within the given range.
 * Items without a numeric priority default to 0.
 */
export function priorityInRange(item: { priority?: number }, range: NormalizedPriorityRange): boolean {
  const priority = typeof item.priority === 'number' && Number.isFinite(item.priority) ? item.priority : 0;
  if (range.min !== undefined && priority < range.min) return false;
  if (range.max !== undefined && priority > range.max) return false;
  return true;
}

/**
 * Return whether an item's method name starts with the given prefix (case-insensitive).
 * An empty prefix matches everything.
 */
export function matchesMethodPrefix(item: { method?: string }, normalizedPrefix: string): boolean {
  if (!normalizedPrefix) return true;
  return String(item.method || '')
    .toLowerCase()
    .startsWith(normalizedPrefix);
}
