// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  assertRecordReadable,
  type RecordProbeDialFn,
} from '@/core/service/orm/model';
import { AuditErrCode, GrpcCode, newAuditError } from './error';

export type TargetRecordAuthFn = (model: string, resId: string) => Promise<void>;
export type TargetRecordDialFn = RecordProbeDialFn;

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
 * Delegates the dial Search probe to core {@link assertRecordReadable}; wraps failures as audit PERMISSION_DENIED.
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
    await assertRecordReadable(model, resId, {
      dial: targetRecordDialOverride,
      message: deniedMessage,
    });
  } catch {
    throw permissionDenied(deniedMessage);
  }
}
