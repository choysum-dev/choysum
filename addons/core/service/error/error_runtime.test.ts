// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError, errorAs, isErrorOf } from '@/core/service/error';

test('runtime error wrap handles Error, non-Error and null-like causes', () => {
  const fromError = ChoysumError.wrap(
    new Error('plain'),
    {
      domain: 'core',
      code: 'E_WRAP_ERR',
      message: 'wrapped error',
    },
    true
  );
  expect(String((fromError as any).cause?.message || '')).toBe('plain');

  const fromPrimitive = ChoysumError.wrap(
    0,
    {
      domain: 'core',
      code: 'E_WRAP_NUM',
      message: 'wrapped primitive',
    },
    true
  );
  expect((fromPrimitive as any).cause instanceof Error).toBe(true);
  expect(String((fromPrimitive as any).cause?.message || '')).toBe('0');

  const fromNull = ChoysumError.wrap(
    null,
    {
      domain: 'core',
      code: 'E_WRAP_NULL',
      message: 'wrapped null',
    },
    true
  );
  expect((fromNull as any).cause).toBeUndefined();
});

test('runtime error as/is traverses nested ChoysumError chains', () => {
  class CustomError extends Error {}

  const leaf = new ChoysumError({ domain: 'auth', code: 'E_LEAF', message: 'leaf' }).withCause(new CustomError('custom-leaf'));
  const root = new ChoysumError({ domain: 'core', code: 'E_ROOT', message: 'root' }).withCause(leaf).withGrpcCode(GrpcCode.Internal);

  expect(root.as(ChoysumError)).toBe(root);
  expect(root.as(CustomError) instanceof CustomError).toBe(true);
  expect(root.is('auth', 'E_LEAF')).toBe(true);
  expect(root.is('missing')).toBe(false);
});

test('runtime errorAs and isErrorOf handle direct type matches and generic Error causes', () => {
  const direct = new TypeError('direct');
  expect(errorAs(direct, TypeError)).toBe(direct);
  expect(errorAs(new Error('plain'), TypeError)).toBe(null);

  const chained = new Error('outer') as any;
  chained.cause = new TypeError('inner');
  expect(errorAs(chained, TypeError) instanceof TypeError).toBe(true);

  const coreErr = new ChoysumError({ domain: 'core', code: 'E_CORE', message: 'core' });
  const wrapped = new Error('wrapper') as any;
  wrapped.cause = coreErr;

  expect(isErrorOf(wrapped, 'core', 'E_CORE')).toBe(true);
  expect(isErrorOf(new Error('no-cause'), 'core')).toBe(false);
});
