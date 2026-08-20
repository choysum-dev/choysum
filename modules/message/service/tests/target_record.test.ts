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

async function withSeams<T>(fn: () => Promise<T>): Promise<T> {
  resetSeams();
  try {
    return await fn();
  } finally {
    resetSeams();
  }
}

test('message target_record: denies when the test override is null', async () => {
  await withSeams(async () => {
    __setMessageTargetRecordAuthForTest(null);
    let err: unknown;
    try {
      await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  });
});

test('message target_record: allows when the test override succeeds', async () => {
  await withSeams(async () => {
    __setMessageTargetRecordAuthForTest(async () => undefined);
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  });
});

test('message target_record: denies when dial Search is missing', async () => {
  await withSeams(async () => {
    __setMessageTargetRecordDialForTest(() => ({}));
    let err: unknown;
    try {
      await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect(isMessageError(err)).toBe(true);
    expect((err as any).code).toBe(MessageErrCode.PERMISSION_DENIED);
  });
});

test('message target_record: allows when dial Search finds the record', async () => {
  await withSeams(async () => {
    __setMessageTargetRecordDialForTest(
      () =>
        ({
          Search: async () => [{ Id: 'r1' }],
        }) as any
    );
    await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
  });
});

test('message target_record: denies when dial Search returns no rows', async () => {
  await withSeams(async () => {
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
  });
});

test('message target_record: rethrows permission denied errors from dial Search', async () => {
  await withSeams(async () => {
    const denied = { code: MessageErrCode.PERMISSION_DENIED, message: 'blocked' };
    __setMessageTargetRecordDialForTest(
      () =>
        ({
          Search: async () => {
            throw denied;
          },
        }) as any
    );
    let err: unknown;
    try {
      await assertTargetRecordReadable('partner.Partner', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect(err).toBe(denied);
  });
});

test('message target_record: maps other dial failures to permission denied', async () => {
  await withSeams(async () => {
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
  });
});
