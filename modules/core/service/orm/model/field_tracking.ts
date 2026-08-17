// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Field tracking → audit.FieldChange (AU3 / PR-P3-A2).
 *
 * Failure policy (frozen): audit Append failures **block** the business write
 * (fail-closed) so compliance history is not silently dropped. Skip quietly when
 * the model is audit.FieldChange itself (avoid recursion) or when no tracked
 * fields participate in the write.
 */

import { dial } from './model_pool';
import { MetadataStorage } from '../metadata';
import type { FieldMetadata } from '../metadata/field';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import type { ObjectRecord } from '../../../utils/types';
import { getActiveCompanyId } from '../../runtime/context';

const AUDIT_FIELD_CHANGE = 'audit.FieldChange';

type AppendFn = (req: {
  Model: string;
  ResId: string;
  Field?: string | null;
  Kind: string;
  OldValue?: string | null;
  NewValue?: string | null;
  CompanyId?: string | null;
}) => Promise<unknown>;

/** Test seam: undefined = live dial; null = force missing; function = stub Append. */
let appendOverride: AppendFn | null | undefined;

/**
 * Test-only override for audit Append resolution.
 */
export function __setFieldTrackingAppendForTest(fn: AppendFn | null | undefined): void {
  appendOverride = fn;
}

export type FieldTrackingWriteEvent = {
  childCtor: RuntimeModelCtor;
  operation: 'create' | 'update' | 'delete';
  changedFields?: string[];
  beforeEntity?: ObjectRecord;
  afterEntity?: ObjectRecord;
};

function fullModelName(ModelCtor: RuntimeModelCtor): string {
  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as any);
  const app = String(meta?.application || '').trim();
  const name = String(meta?.name || ModelCtor?.name || '').trim();
  return app && name ? `${app}.${name}` : name;
}

function isTrackableScalar(fm: FieldMetadata | undefined): boolean {
  if (!fm || fm.tracking !== true) return false;
  const t = String(fm.type || '');
  if (t === 'OneToMany' || t === 'ManyToMany' || t === 'properties') return false;
  return true;
}

function serializeTrackedValue(value: unknown): string | null {
  if (value === undefined || value === null) return null;
  if (value instanceof Date) return value.toISOString();
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function valuesEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (a == null && b == null) return true;
  if (a instanceof Date && b instanceof Date) return a.getTime() === b.getTime();
  return serializeTrackedValue(a) === serializeTrackedValue(b);
}

function resolveAppend(): AppendFn | null {
  if (appendOverride !== undefined) return appendOverride;
  try {
    const svc = dial<{ Append?: AppendFn }>(AUDIT_FIELD_CHANGE);
    if (typeof svc?.Append !== 'function') return null;
    return svc.Append.bind(svc);
  } catch {
    return null;
  }
}

/**
 * After a successful scalar write, append FieldChange rows for tracked fields.
 * Fail-closed: Append errors propagate to the caller.
 */
export async function recordFieldTrackingEvents(event: FieldTrackingWriteEvent): Promise<void> {
  const ModelCtor = event.childCtor;
  if (!ModelCtor) return;

  const trackedModel = fullModelName(ModelCtor);
  if (!trackedModel || trackedModel === AUDIT_FIELD_CHANGE) return;

  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor as any);
  const fields = meta?.fields;
  if (!fields?.size) return;

  const hasAnyTracking = Array.from(fields.values()).some(fm => isTrackableScalar(fm));
  if (!hasAnyTracking) return;

  const resId = String(event.afterEntity?.Id ?? event.beforeEntity?.Id ?? '').trim();
  if (!resId) return;

  const companyId = (() => {
    const fromRow = event.afterEntity?.CompanyId ?? event.beforeEntity?.CompanyId;
    if (fromRow != null && String(fromRow).trim()) return String(fromRow).trim();
    try {
      const active = String(getActiveCompanyId() || '').trim();
      return active || null;
    } catch {
      return null;
    }
  })();

  const append = resolveAppend();
  if (!append) {
    throw new Error(`[FieldTracking] ${AUDIT_FIELD_CHANGE} is not available but ${trackedModel} has tracking fields`);
  }

  if (event.operation === 'create') {
    await append({
      Model: trackedModel,
      ResId: resId,
      Field: null,
      Kind: 'create',
      OldValue: null,
      NewValue: null,
      CompanyId: companyId,
    });
    return;
  }

  if (event.operation === 'delete') {
    await append({
      Model: trackedModel,
      ResId: resId,
      Field: null,
      Kind: 'unlink',
      OldValue: null,
      NewValue: null,
      CompanyId: companyId,
    });
    return;
  }

  // update
  const changed = event.changedFields || [];
  for (const name of changed) {
    const fm = fields.get(name);
    if (!isTrackableScalar(fm)) continue;
    const before = event.beforeEntity?.[name];
    const after = event.afterEntity?.[name];
    if (valuesEqual(before, after)) continue;
    await append({
      Model: trackedModel,
      ResId: resId,
      Field: name,
      Kind: 'field',
      OldValue: serializeTrackedValue(before),
      NewValue: serializeTrackedValue(after),
      CompanyId: companyId,
    });
  }
}
