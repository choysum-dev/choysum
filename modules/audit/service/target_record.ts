// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { dial } from '@/core/service/orm/model/model_pool';
import { AuditErrCode, GrpcCode, newAuditError } from './error';

export type TargetRecordAuthFn = (model: string, resId: string) => Promise<void>;
export type TargetRecordDialFn = <T = Record<string, (...args: unknown[]) => unknown>>(fullModelName: string) => T;

let targetRecordAuthOverride: TargetRecordAuthFn | null | undefined;
let targetRecordDialOverride: TargetRecordDialFn | undefined;

/**
 * Test-only override for FieldChange target-record read checks.
 * undefined = live dial Search; null = force deny; function = stub.
 */
export function __setFieldChangeTargetAuthForTest(fn: TargetRecordAuthFn | null | undefined): void {
  targetRecordAuthOverride = fn;
}

/**
 * Test-only override for the live target dial used when auth override is unset.
 */
export function __setFieldChangeTargetDialForTest(fn: TargetRecordDialFn | undefined): void {
  targetRecordDialOverride = fn;
}

function permissionDenied(message: string) {
  return newAuditError({ code: AuditErrCode.PERMISSION_DENIED, message }).withGrpcCode(GrpcCode.PermissionDenied);
}

/**
 * Confirms the caller can Search the underlying business record identified by Model/ResId.
 */
export async function assertTargetRecordReadable(model: string, resId: string, deniedMessage: string): Promise<void> {
  if (targetRecordAuthOverride !== undefined) {
    if (targetRecordAuthOverride === null) {
      throw permissionDenied(deniedMessage);
    }
    await targetRecordAuthOverride(model, resId);
    return;
  }

  try {
    const dialFn = targetRecordDialOverride || dial;
    const svc = dialFn<{ Search?: (condition: unknown, options?: unknown) => Promise<unknown> }>(model);
    if (typeof svc?.Search !== 'function') {
      throw permissionDenied(deniedMessage);
    }
    const rows = await svc.Search(['Id', '=', resId], { fields: ['Id'], limit: 1 });
    if (Array.isArray(rows) && rows.length > 0) return;
  } catch (err) {
    if (err && typeof err === 'object' && (err as { code?: string }).code === AuditErrCode.PERMISSION_DENIED) {
      throw err;
    }
  }
  throw permissionDenied(deniedMessage);
}
