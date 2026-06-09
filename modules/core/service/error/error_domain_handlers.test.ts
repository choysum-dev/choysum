// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError, createDomainErrorHandlers, errorAs, isErrorOf } from '@/core/service/error';

test('core error isErrorOf matches nested cause chain by domain and code', () => {
  const inner = new ChoysumError({
    domain: 'core.error',
    code: 'E_MATCH',
    message: 'inner-error',
  });
  const wrapped = new Error('outer-error');
  (wrapped as any).cause = inner;

  expect(isErrorOf(wrapped, 'core.error')).toBe(true);
  expect(isErrorOf(wrapped, 'core.error', 'E_MATCH')).toBe(true);
  expect(isErrorOf(wrapped, 'core.error', 'E_OTHER')).toBe(false);
  expect(isErrorOf(wrapped, 'other.domain')).toBe(false);
  expect(isErrorOf(new Error('plain'), 'core.error')).toBe(false);
});

test('core error domain handlers isError delegates to domain checker', () => {
  const handlers = createDomainErrorHandlers<'E_CREATE' | 'E_WRAP'>('core.domain');

  const created = handlers.newError({ code: 'E_CREATE', message: 'created' });
  expect(handlers.isError(created)).toBe(true);
  expect(handlers.isError(created, 'E_CREATE')).toBe(true);
  expect(handlers.isError(created, 'E_WRAP')).toBe(false);

  const wrapped = handlers.wrapError(new Error('boom'), { code: 'E_WRAP', message: 'wrapped' });
  expect(handlers.isError(wrapped)).toBe(true);
  expect(handlers.isError(wrapped, 'E_WRAP')).toBe(true);
  expect(handlers.isError(wrapped, 'E_CREATE')).toBe(false);
});

test('core error ChoysumError supports fromErrorInfo toErrorInfo and toString roundtrip', () => {
  const source = new ChoysumError({
    domain: 'core.roundtrip',
    code: 'E_RT',
    message: 'roundtrip',
  })
    .withGrpcCode(GrpcCode.Internal)
    .withMetadata({ a: '1', b: '2' });

  const info = source.toErrorInfo();
  const cloned = ChoysumError.fromErrorInfo(info as any);

  expect(cloned.domain).toBe('core.roundtrip');
  expect(cloned.code).toBe('E_RT');
  expect(cloned.message).toBe('roundtrip');
  expect(cloned.metadata).toEqual({ a: '1', b: '2' });

  const text = cloned.toString();
  expect(text.includes('[core.roundtrip]')).toBe(true);
  expect(text.includes('E_RT')).toBe(true);
  expect(text.includes('a=1')).toBe(true);
});

test('core error ChoysumError withCause and errorAs resolve nested error types', () => {
  class CustomError extends Error {}

  const leaf = new CustomError('leaf');
  const middle = new ChoysumError({ domain: 'core.chain', code: 'E_MID', message: 'middle' }).withCause(leaf);
  const top = new ChoysumError({ domain: 'core.chain', code: 'E_TOP', message: 'top' }).withCause(middle);

  const asCustom = top.as(CustomError);
  expect(asCustom instanceof CustomError).toBe(true);
  expect(asCustom?.message).toBe('leaf');

  const viaGlobal = errorAs(top, CustomError);
  expect(viaGlobal instanceof CustomError).toBe(true);
  expect(viaGlobal?.message).toBe('leaf');

  const text = top.toString();
  expect(text.includes('Caused by')).toBe(true);
  expect(text.includes('leaf')).toBe(true);
});
