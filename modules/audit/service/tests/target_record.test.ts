// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  __setFieldChangeTargetAuthForTest,
  __setFieldChangeTargetDialForTest,
  assertTargetRecordReadable,
} from '../target_record';
import { AuditErrCode, isAuditError } from '../error';

function resetSeams(): void {
  __setFieldChangeTargetAuthForTest(undefined);
  __setFieldChangeTargetDialForTest(undefined);
}

test('audit target_record: denies when the test override is null', async () => {
  resetSeams();
  __setFieldChangeTargetAuthForTest(null);
  let err: unknown;
  try {
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('audit target_record: allows when the test override succeeds', async () => {
  resetSeams();
  __setFieldChangeTargetAuthForTest(async () => undefined);
  await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  resetSeams();
});

test('audit target_record: denies when dial Search is missing', async () => {
  resetSeams();
  __setFieldChangeTargetDialForTest(() => ({}));
  let err: unknown;
  try {
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect(isAuditError(err)).toBe(true);
  expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('audit target_record: allows when dial Search finds the record', async () => {
  resetSeams();
  __setFieldChangeTargetDialForTest(
    () =>
      ({
        Search: async () => [{ Id: 'r1' }],
      }) as any
  );
  await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  resetSeams();
});

test('audit target_record: denies when dial Search returns no rows', async () => {
  resetSeams();
  __setFieldChangeTargetDialForTest(
    () =>
      ({
        Search: async () => [],
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('audit target_record: rethrows permission denied errors from dial Search', async () => {
  resetSeams();
  __setFieldChangeTargetDialForTest(
    () =>
      ({
        Search: async () => {
          throw { code: AuditErrCode.PERMISSION_DENIED, message: 'blocked' };
        },
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('audit target_record: maps other dial failures to permission denied', async () => {
  resetSeams();
  __setFieldChangeTargetDialForTest(
    () =>
      ({
        Search: async () => {
          throw new Error('boom');
        },
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  resetSeams();
});
