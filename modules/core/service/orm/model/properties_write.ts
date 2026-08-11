// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { raiseDomainError } from '@/core/service/error';
import type { ObjectRecord } from '../../../utils/types';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { loadEffectivePropertySchema } from './properties_resolve';
import { isPlainPropertiesMap, type PropertyItemDefinition } from './properties_types';

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function coarseTypeOk(item: PropertyItemDefinition, value: unknown): boolean {
  if (value === null || value === undefined) return true;
  switch (item.type) {
    case 'boolean':
      return typeof value === 'boolean';
    case 'integer':
      return typeof value === 'number' && Number.isFinite(value) && Number.isInteger(value);
    case 'float':
      return typeof value === 'number' && Number.isFinite(value);
    case 'char':
    case 'text':
    case 'date':
    case 'datetime':
    case 'selection':
      return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean';
    default:
      return true;
  }
}

/**
 * Validate and normalize a properties field write payload (PP3 / PP4).
 * - Rejects arrays / non-maps
 * - Empty effective schema + non-empty map → fail (PP2)
 * - Unknown names → fail (not strip)
 * - Readonly keys ignored
 * - Returns the replace map
 *
 * Callers should assign the returned map onto the write payload (whole-column replace).
 */
export async function validatePropertiesWrite(
  ModelCtor: RuntimeModelCtor<BaseModel>,
  fieldName: string,
  value: unknown,
  rowCtx: ObjectRecord
): Promise<Record<string, unknown>> {
  if (value === undefined) {
    return {};
  }
  if (Array.isArray(value) || !isPlainPropertiesMap(value)) {
    fail(
      'PROPERTIES_WRITE_SHAPE',
      `Field "${fieldName}" properties write must be a plain object map (arrays are rejected)`
    );
  }

  const schema = await loadEffectivePropertySchema(ModelCtor, fieldName, rowCtx);
  const byName = new Map(schema.map(item => [item.name, item]));
  const submitted = value as Record<string, unknown>;
  const keys = Object.keys(submitted);

  if (schema.length === 0) {
    if (keys.length === 0) return {};
    fail(
      'PROPERTIES_WRITE_NO_SCHEMA',
      `Field "${fieldName}" has empty effective schema; non-empty properties write is not allowed`
    );
  }

  const out: Record<string, unknown> = {};
  for (const key of keys) {
    const item = byName.get(key);
    if (!item) {
      fail('PROPERTIES_WRITE_UNKNOWN_NAME', `Field "${fieldName}" properties write has unknown name "${key}"`);
    }
    if (item.readonly) {
      continue;
    }
    const v = submitted[key];
    if (!coarseTypeOk(item, v)) {
      fail(
        'PROPERTIES_WRITE_TYPE',
        `Field "${fieldName}" property "${key}" value does not match type "${item.type}"`
      );
    }
    out[key] = v;
  }
  return out;
}

/**
 * Validate all `properties` fields present on a create/update payload and rewrite maps in place.
 */
export async function validatePropertiesFieldsOnWrite(params: {
  ModelCtor: RuntimeModelCtor<BaseModel>;
  input: ObjectRecord;
  current?: ObjectRecord;
  mode: 'create' | 'update' | string;
}): Promise<void> {
  const { ModelCtor, input, current, mode } = params;
  if (mode !== 'create' && mode !== 'update') return;

  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  const rowCtx: ObjectRecord = { ...(current || {}), ...(input || {}) };

  for (const [fieldName, fm] of meta.fields) {
    if (!fm || fm.type !== 'properties') continue;
    if (!Object.prototype.hasOwnProperty.call(input, fieldName)) continue;
    const normalized = await validatePropertiesWrite(ModelCtor, fieldName, input[fieldName], rowCtx);
    input[fieldName] = normalized;
  }
}
