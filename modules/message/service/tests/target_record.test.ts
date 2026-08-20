// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  __setMessageTargetRecordAuthForTest,
  __setMessageTargetRecordDialForTest,
  assertTargetRecordReadable,
} from '../target_record';
import { MessageErrCode, isMessageError } from '../error';

function resetSeams(): void {
  __setMessageTargetRecordAuthForTest(undefined);
  __setMessageTargetRecordDialForTest(undefined);
}

test('message target_record: denies when the test override is null', async () => {
  resetSeams();
  __setMessageTargetRecordAuthForTest(null);
  let err: unknown;
  try {
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('message target_record: allows when the test override succeeds', async () => {
  resetSeams();
  __setMessageTargetRecordAuthForTest(async () => undefined);
  await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  resetSeams();
});

test('message target_record: denies when dial Search is missing', async () => {
  resetSeams();
  __setMessageTargetRecordDialForTest(() => ({}));
  let err: unknown;
  try {
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect(isMessageError(err)).toBe(true);
  expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('message target_record: allows when dial Search finds the record', async () => {
  resetSeams();
  __setMessageTargetRecordDialForTest(
    () =>
      ({
        Search: async () => [{ Id: 'r1' }],
      }) as any
  );
  await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  resetSeams();
});

test('message target_record: denies when dial Search returns no rows', async () => {
  resetSeams();
  __setMessageTargetRecordDialForTest(
    () =>
      ({
        Search: async () => [],
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('message target_record: rethrows permission denied errors from dial Search', async () => {
  resetSeams();
  __setMessageTargetRecordDialForTest(
    () =>
      ({
        Search: async () => {
          throw { code: MessageErrCode.PERMISSION_DENIED, message: 'blocked' };
        },
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  resetSeams();
});

test('message target_record: maps other dial failures to permission denied', async () => {
  resetSeams();
  __setMessageTargetRecordDialForTest(
    () =>
      ({
        Search: async () => {
          throw new Error('boom');
        },
      }) as any
  );
  let err: unknown;
  try {
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  } catch (e) {
    err = e;
  }
  expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  resetSeams();
});
