// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { ValidationPipelineError } from '../metadata';
import type { ObjectRecord } from '../../../utils/types';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { loadEffectivePropertySchema } from './properties_resolve';
import { isPlainPropertiesMap, normalizePropertiesMap, propertyValueMatchesType } from './properties_types';

function fail(fieldName: string, code: string, message: string): never {
  throw new ValidationPipelineError(message, [
    {
      scope: 'platform',
      field: fieldName,
      code,
      message,
      severity: 'error',
    },
  ]);
}

/**
 * Validate and normalize a properties field write payload (PP3 / PP4).
 * - `undefined` → skip (caller must not assign)
 * - `null` → clear column
 * - Rejects arrays / non-maps
 * - Empty effective schema + non-empty map → fail (PP2)
 * - Unknown names → fail (not strip)
 * - Readonly keys from submission ignored; existing readonly values preserved from `currentMap`
 * - Returns the replace map (writable submitted keys ⊕ preserved readonly)
 *
 * Callers should assign the returned map onto the write payload (whole-column replace).
 */
export async function validatePropertiesWrite(
  ModelCtor: RuntimeModelCtor<BaseModel>,
  fieldName: string,
  value: unknown,
  rowCtx: ObjectRecord,
  currentMap: Record<string, unknown> = {}
): Promise<Record<string, unknown> | null | undefined> {
  if (value === undefined) {
    return undefined;
  }
  if (value === null) {
    return null;
  }
  if (Array.isArray(value) || !isPlainPropertiesMap(value)) {
    fail(
      fieldName,
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
      fieldName,
      'PROPERTIES_WRITE_NO_SCHEMA',
      `Field "${fieldName}" has empty effective schema; non-empty properties write is not allowed`
    );
  }

  const out: Record<string, unknown> = {};
  for (const key of keys) {
    const item = byName.get(key);
    if (!item) {
      fail(fieldName, 'PROPERTIES_WRITE_UNKNOWN_NAME', `Field "${fieldName}" properties write has unknown name "${key}"`);
    }
    if (item.readonly) {
      continue;
    }
    const v = submitted[key];
    if (!propertyValueMatchesType(item, v)) {
      fail(
        fieldName,
        'PROPERTIES_WRITE_TYPE',
        `Field "${fieldName}" property "${key}" value does not match type "${item.type}"`
      );
    }
    out[key] = v;
  }

  // Preserve readonly values from the current column (PP3 ignore on submit; do not wipe).
  for (const item of schema) {
    if (!item.readonly) continue;
    if (Object.prototype.hasOwnProperty.call(currentMap, item.name)) {
      out[item.name] = currentMap[item.name];
    }
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
    const currentMap = normalizePropertiesMap(current?.[fieldName]);
    const normalized = await validatePropertiesWrite(ModelCtor, fieldName, input[fieldName], rowCtx, currentMap);
    if (normalized === undefined) {
      delete input[fieldName];
      continue;
    }
    input[fieldName] = normalized;
  }
}
