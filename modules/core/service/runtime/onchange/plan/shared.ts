// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Re-exported runtime model constructor type used by onchange plan helpers.
 */
export type { RuntimeModelCtor as ModelCtor } from '../../../orm/model/types';
import type { ObjectRecord } from '../../../../utils/types';

/**
 * Returns whether a value is an object record.
 */
export function isObject(v: unknown): v is ObjectRecord {
  return v !== null && typeof v === 'object';
}

/**
 * Deduplicates array values while preserving first-seen order.
 */
export function uniq<T>(arr: T[]): T[] {
  return Array.from(new Set(arr));
}

/**
 * Splits an array into chunks of the requested size.
 */
export function chunk<T>(arr: T[], size: number): T[][] {
  if (!Array.isArray(arr) || arr.length === 0) return [];
  if (!size || size <= 0) return [arr.slice()];
  const out: T[][] = [];
  for (let i = 0; i < arr.length; i += size) {
    out.push(arr.slice(i, i + size));
  }
  return out;
}
