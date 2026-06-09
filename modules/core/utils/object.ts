// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ObjectRecord } from './types';

export function isObjectRecord(value: unknown): value is ObjectRecord {
  if (!value || typeof value !== 'object') return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

function isObjectLike(value: unknown): value is ObjectRecord {
  return value !== null && typeof value === 'object';
}

export function asObjectRecord(value: unknown): ObjectRecord | undefined {
  return isObjectLike(value) ? value : undefined;
}

export function asRuntimeCarrier(value: unknown): ObjectRecord | undefined {
  if (value === null) return undefined;
  if (typeof value === 'object') return value as ObjectRecord;
  if (typeof value === 'function') return value as unknown as ObjectRecord;
  return undefined;
}

export function hasOwnKey(record: ObjectRecord, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(record, key);
}

export function getOwnValue(record: ObjectRecord, key: string): unknown {
  return hasOwnKey(record, key) ? record[key] : undefined;
}

export function isStringNumberEnvelope<TKey extends string>(value: unknown, key: TKey): value is Record<TKey, string | number> {
  const record = asObjectRecord(value);
  if (!record || !hasOwnKey(record, key)) return false;
  const wrapped = record[key];
  return typeof wrapped === 'string' || typeof wrapped === 'number';
}
