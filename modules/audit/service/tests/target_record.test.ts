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

async function withSeams<T>(fn: () => Promise<T>): Promise<T> {
  resetSeams();
  try {
    return await fn();
  } finally {
    resetSeams();
  }
}

test('audit target_record: denies when the test override is null', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetAuthForTest(null);
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  });
});

test('audit target_record: allows when the test override succeeds', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetAuthForTest(async () => undefined);
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  });
});

test('audit target_record: denies when dial Search is missing', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetDialForTest(() => ({}));
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect(isAuditError(err)).toBe(true);
    expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  });
});

test('audit target_record: allows when dial Search finds the record', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetDialForTest(
      () =>
        ({
          Search: async () => [{ Id: 'r1' }],
        }) as any
    );
    await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
  });
});

test('audit target_record: denies when dial Search returns no rows', async () => {
  await withSeams(async () => {
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
  });
});

test('audit target_record: rethrows permission denied errors from dial Search', async () => {
  await withSeams(async () => {
    const denied = { code: AuditErrCode.PERMISSION_DENIED, message: 'blocked' };
    __setFieldChangeTargetDialForTest(
      () =>
        ({
          Search: async () => {
            throw denied;
          },
        }) as any
    );
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect(err).toBe(denied);
  });
});

test('audit target_record: maps other dial failures to permission denied', async () => {
  await withSeams(async () => {
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
  });
});

test('audit target_record: denies when dial Search returns a non-array payload', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetDialForTest(
      () =>
        ({
          Search: async () => ({ Id: 'r1' }),
        }) as any
    );
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  });
});

test('audit target_record: denies when dial returns a null service', async () => {
  await withSeams(async () => {
    __setFieldChangeTargetDialForTest(() => null as any);
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  });
});

test('audit target_record: falls back to live dial when dial override is unset', async () => {
  await withSeams(async () => {
    // Both seams stay undefined so dialFn resolves via `targetRecordDialOverride || dial`.
    let err: unknown;
    try {
      await assertTargetRecordReadable('base.UoM', 'r1', 'denied');
    } catch (e) {
      err = e;
    }
    expect((err as any).code).toBe(AuditErrCode.PERMISSION_DENIED);
  });
});
