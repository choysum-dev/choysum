// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../orm/metadata/model';
import type { UnknownRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';
import { withBridgeFrame } from './bridge';
import { createEntityBackedModelInstance, resolveInstanceHandler } from './handler_runtime';
import { validateAutoInverseRelatedPath } from './parser';

function isPromiseLike<T = unknown>(value: unknown): value is Promise<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

function normalizeFieldName(value: unknown): string {
  return String(value || '').trim();
}

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

function mergeDeep(target: UnknownRecord, patch: UnknownRecord): void {
  for (const [key, nextVal] of Object.entries(patch)) {
    const current = asObjectRecord(target[key]);
    const nextRecord = asObjectRecord(nextVal);
    if (current && nextRecord) {
      mergeDeep(current, nextRecord);
      target[key] = current;
      continue;
    }
    target[key] = nextVal;
  }
}

function setPath(target: UnknownRecord, path: string, value: unknown): void {
  const segments = String(path || '')
    .split('.')
    .map(seg => seg.trim())
    .filter(Boolean);
  if (!segments.length) {
    throw new Error('inverse write path is empty');
  }

  let cursor: UnknownRecord = target;
  for (let i = 0; i < segments.length - 1; i++) {
    const seg = segments[i];
    const bucket = asObjectRecord(cursor[seg]) || {};
    cursor[seg] = bucket;
    cursor = bucket;
  }

  cursor[segments[segments.length - 1]] = value;
}

function readPath(source: UnknownRecord, path: string): unknown {
  const segments = String(path || '')
    .split('.')
    .map(seg => seg.trim())
    .filter(Boolean);
  if (!segments.length) return undefined;

  let cursor: unknown = source;
  for (const seg of segments) {
    const record = asObjectRecord(cursor);
    if (!record) return undefined;
    cursor = record[seg];
  }
  return cursor;
}

function getExplicitInverseMethod(meta: ModelMetadata, fieldName: string): string {
  const handlerMethod = normalizeFieldName(meta.inverseHandlers?.get(fieldName)?.method);
  if (handlerMethod) return handlerMethod;

  const fieldMeta = meta.fields.get(fieldName);
  const legacyMethod = normalizeFieldName(fieldMeta?.column?.compute?.inverse);
  return legacyMethod;
}

export function tryAutoInverseWriteback(meta: ModelMetadata, fieldName: string, value: unknown): UnknownRecord | undefined {
  const fieldMeta = meta.fields.get(fieldName);
  const related = fieldMeta?.related;
  if (!related || related.store !== true) return;

  const relatedPath = String(related.path || '').trim();
  const validated = validateAutoInverseRelatedPath(meta, fieldName, relatedPath);
  return {
    [validated.root]: {
      [validated.leaf]: value,
    },
  };
}

export async function applyInverseWriteback(meta: ModelMetadata, values: UnknownRecord): Promise<UnknownRecord> {
  const model = modelLabel(meta);
  const nextValues: UnknownRecord = { ...values };
  const fields = Object.keys(nextValues);

  for (const fieldName of fields) {
    const fieldMeta = meta.fields.get(fieldName);
    if (!fieldMeta) continue;

    const hasRelated = !!fieldMeta.related?.path;
    const hasComputeBehavior = !!meta.computeHandlers?.has(fieldName) || !!meta.sqlComputeHandlers?.has(fieldName) || !!fieldMeta.column?.compute;
    if (!hasRelated && !hasComputeBehavior) continue;

    const explicitMethod = getExplicitInverseMethod(meta, fieldName);
    const inputValue = nextValues[fieldName];

    if (!explicitMethod) {
      if (hasRelated && fieldMeta.related?.store === false) {
        throw new Error(`INVERSE_HANDLER_REQUIRED: ${model}.${fieldName} is related.store=false and cannot be written`);
      }

      try {
        const autoPatch = tryAutoInverseWriteback(meta, fieldName, inputValue);
        if (autoPatch) {
          mergeDeep(nextValues, autoPatch);
          delete nextValues[fieldName];
          continue;
        }
      } catch (error) {
        const message = String((error as Error)?.message || error);
        throw new Error(`INVERSE_HANDLER_REQUIRED: ${model}.${fieldName} cannot use auto inverse writeback (${message})`);
      }

      if (hasRelated || hasComputeBehavior) {
        if (hasComputeBehavior && !hasRelated) {
          // Keep compute-field payload untouched and let platform validation
          // return canonical write-protection errors.
          continue;
        }
        throw new Error(`INVERSE_HANDLER_REQUIRED: ${model}.${fieldName} requires an explicit @Inverse handler`);
      }
      continue;
    }

    const inverseMethod = resolveInstanceHandler(meta, fieldName, explicitMethod, '@Inverse');
    const modelInstance = createEntityBackedModelInstance(meta, nextValues);
    const handlerWrites: UnknownRecord = {};

    const inverseCtx = {
      value<T = unknown>() {
        return inputValue as T;
      },
      writePath(path: string, next: unknown) {
        setPath(handlerWrites, path, next);
      },
      readPath(path: string) {
        return readPath(nextValues, path);
      },
      record: modelInstance,
    };

    const result = withBridgeFrame(modelInstance as object, 'inverse', inverseCtx, () => inverseMethod.call(modelInstance));
    const settledResult = isPromiseLike(result) ? await result : result;

    if (Object.keys(handlerWrites).length) {
      mergeDeep(nextValues, handlerWrites);
    }

    const patchResult = asObjectRecord(settledResult);
    if (patchResult) {
      mergeDeep(nextValues, patchResult);
    }

    delete nextValues[fieldName];
  }

  return nextValues;
}
