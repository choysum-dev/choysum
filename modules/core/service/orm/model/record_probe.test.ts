// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import {
  assertRecordReadable,
  isRecordNotReadableError,
  RECORD_NOT_READABLE,
  RECORD_PROBE_DOMAIN,
  type RecordProbeDialFn,
} from './record_probe';

function dialWithSearch(search: (() => Promise<unknown>) | undefined): RecordProbeDialFn {
  return () =>
    ({
      Search: search,
    }) as Record<string, (...args: unknown[]) => unknown>;
}

async function expectNotReadable(promise: Promise<unknown>, message?: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(isRecordNotReadableError(err)).toBe(true);
    expect(err instanceof ChoysumError).toBe(true);
    const e = err as ChoysumError;
    expect(e.domain).toBe(RECORD_PROBE_DOMAIN);
    expect(e.code).toBe(RECORD_NOT_READABLE);
    expect(e.grpcCode).toBe(GrpcCode.PermissionDenied);
    if (message !== undefined) expect(e.message).toBe(message);
  }
}

test('assertRecordReadable: allows when dial Search finds the record', async () => {
  await assertRecordReadable('partner.Partner', 'r1', {
    dial: dialWithSearch(async () => [{ Id: 'r1' }]),
  });
});

test('assertRecordReadable: denies when dial Search returns no rows', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: dialWithSearch(async () => []),
      message: 'missing',
    }),
    'missing'
  );
});

test('assertRecordReadable: denies when dial Search returns a non-array payload', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: dialWithSearch(async () => ({ Id: 'r1' })),
    })
  );
});

test('assertRecordReadable: denies when Search is missing', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: () => ({}) as Record<string, (...args: unknown[]) => unknown>,
    })
  );
});

test('assertRecordReadable: denies when dial returns a null service', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: (() => null) as unknown as RecordProbeDialFn,
    })
  );
});

test('assertRecordReadable: denies empty model or id', async () => {
  await expectNotReadable(assertRecordReadable('', 'r1', { message: 'empty-model' }), 'empty-model');
  await expectNotReadable(assertRecordReadable('partner.Partner', '  ', { message: 'empty-id' }), 'empty-id');
  await expectNotReadable(assertRecordReadable('  ', '', { dial: dialWithSearch(async () => [{ Id: 'x' }]) }));
});

test('assertRecordReadable: maps dial Search failures to RECORD_NOT_READABLE', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: dialWithSearch(async () => {
        throw new Error('boom');
      }),
      message: 'wrapped',
    }),
    'wrapped'
  );
});

test('assertRecordReadable: maps dial factory failures to RECORD_NOT_READABLE', async () => {
  await expectNotReadable(
    assertRecordReadable('partner.Partner', 'r1', {
      dial: () => {
        throw new ChoysumError({ domain: 'core', code: 'DIAL_INVALID_MODEL', message: 'bad dial' });
      },
      message: 'dial-failed',
    }),
    'dial-failed'
  );
});

test('assertRecordReadable: rethrows existing RECORD_NOT_READABLE from Search', async () => {
  const denied = new ChoysumError({
    domain: RECORD_PROBE_DOMAIN,
    code: RECORD_NOT_READABLE,
    message: 'inner',
  }).withGrpcCode(GrpcCode.PermissionDenied);

  let err: unknown;
  try {
    await assertRecordReadable('partner.Partner', 'r1', {
      dial: dialWithSearch(async () => {
        throw denied;
      }),
    });
  } catch (e) {
    err = e;
  }
  expect(err).toBe(denied);
});

test('isRecordNotReadableError: false for other errors', () => {
  expect(isRecordNotReadableError(new Error('x'))).toBe(false);
  expect(
    isRecordNotReadableError(new ChoysumError({ domain: 'core', code: 'DIAL_INVALID_MODEL', message: 'x' }))
  ).toBe(false);
  expect(
    isRecordNotReadableError(
      new ChoysumError({ domain: 'message', code: 'PERMISSION_DENIED', message: 'x' })
    )
  ).toBe(false);
});
