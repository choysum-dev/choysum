// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata/model';
import type { FieldMetadata } from '../../metadata/field';
import { asObjectRecord } from '../../../../utils/object';

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

function resolveSqlComputeMethod(meta: ModelMetadata, field: string): ((this: unknown) => unknown) | undefined {
  const sqlHandler = meta.sqlComputeHandlers?.get(field);
  if (!sqlHandler) return;

  const methodName = String(sqlHandler.method || '').trim();
  if (!methodName) {
    throw new Error(`@SqlCompute handler is missing method name: ${modelLabel(meta)}.${field}`);
  }

  const prototypeRecord = asObjectRecord(meta.type?.prototype);
  const method = prototypeRecord?.[methodName];
  if (typeof method !== 'function') {
    throw new Error(`@SqlCompute handler not found: ${modelLabel(meta)}.${field} -> ${methodName}`);
  }

  return method as (this: unknown) => unknown;
}

export function hasRepositorySqlComputeExpression(meta: ModelMetadata, field: string): boolean {
  return Boolean(meta.sqlComputeHandlers?.has(field));
}

export function isRepositorySelectableScalarField(meta: ModelMetadata, field: string, fieldMeta: FieldMetadata | undefined): boolean {
  if (fieldMeta?.column) return true;
  return hasRepositorySqlComputeExpression(meta, field);
}

export function resolveRepositorySqlComputeExpression(meta: ModelMetadata, field: string, sqlCtx: unknown): unknown | undefined {
  const method = resolveSqlComputeMethod(meta, field);
  if (!method) return;

  const host = Object.create(meta.type?.prototype || Object.prototype);
  Object.defineProperty(host, '$sql', {
    configurable: true,
    enumerable: false,
    get: () => sqlCtx,
  });

  return method.call(host);
}
