// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '@/core/service/error';
import { AuthUserService, isAuthServiceNotPresent, isAuthServiceUnavailable } from '..';

test('auth user service methods delegate via AuthUserService object', async () => {
  const originalGetRecord = AuthUserService.GetRecordRuleCondition;
  const originalGetField = AuthUserService.GetFieldRuleSpec;

  (AuthUserService as any).GetRecordRuleCondition = async (model: string, op: string) => ({ model, op, kind: 'record' });
  (AuthUserService as any).GetFieldRuleSpec = async (model: string) => ({ model, kind: 'field' });

  try {
    expect(await AuthUserService.GetRecordRuleCondition('demo.Model', 'read' as any)).toEqual({
      model: 'demo.Model',
      op: 'read',
      kind: 'record',
    });
    expect(await AuthUserService.GetFieldRuleSpec('demo.Model')).toEqual({
      model: 'demo.Model',
      kind: 'field',
    });
  } finally {
    (AuthUserService as any).GetRecordRuleCondition = originalGetRecord;
    (AuthUserService as any).GetFieldRuleSpec = originalGetField;
  }
});

test('auth user service unavailable checker matches grpc code from plain objects', () => {
  expect(isAuthServiceUnavailable({ code: GrpcCode.Unimplemented })).toBe(true);
  expect(isAuthServiceUnavailable({ grpcCode: GrpcCode.NotFound })).toBe(true);
  expect(isAuthServiceUnavailable({ code: GrpcCode.Unavailable })).toBe(true);
  expect(isAuthServiceUnavailable({ code: GrpcCode.PermissionDenied })).toBe(false);
});

test('auth user service unavailable checker matches ChoysumError grpc code', () => {
  const notFound = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'nf' }).withGrpcCode(GrpcCode.NotFound);
  expect(isAuthServiceUnavailable(notFound)).toBe(true);

  const unavailable = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'na' }).withGrpcCode(GrpcCode.Unavailable);
  expect(isAuthServiceUnavailable(unavailable)).toBe(true);

  const denied = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'pd' }).withGrpcCode(GrpcCode.PermissionDenied);
  expect(isAuthServiceUnavailable(denied)).toBe(false);
});

test('auth user service unavailable checker matches type and message fallbacks', () => {
  expect(isAuthServiceUnavailable(new TypeError('grpc unary unavailable'))).toBe(true);
  expect(isAuthServiceUnavailable(new TypeError('random type error'))).toBe(false);

  const missingMessage = new TypeError('placeholder');
  (missingMessage as any).message = undefined;
  expect(isAuthServiceUnavailable(missingMessage)).toBe(false);

  expect(isAuthServiceUnavailable(new Error('no registered proto files for app auth'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('failed to load method descriptor: auth.User/GetRecordRuleCondition'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('rpc error: code = Unimplemented desc = unknown service auth.User'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('rpc error: code = Unimplemented desc = unknown method auth.User/GetFieldRuleSpec'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('rpc error: code = Unimplemented desc = target method does not exist'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('rpc error: code = Unimplemented desc = \u76ee\u6807\u65b9\u6cd5\u4e0d\u5b58\u5728'))).toBe(true);
  expect(isAuthServiceUnavailable(new Error('other runtime error'))).toBe(false);
});

test('auth user service not-present checker distinguishes missing auth from transient unavailable', () => {
  expect(isAuthServiceNotPresent({ code: GrpcCode.Unimplemented })).toBe(true);
  expect(isAuthServiceNotPresent({ grpcCode: GrpcCode.NotFound })).toBe(true);
  expect(isAuthServiceNotPresent({ code: GrpcCode.Unavailable })).toBe(false);
  expect(isAuthServiceNotPresent(new TypeError('grpc unary unavailable'))).toBe(false);

  // ChoysumError carries grpcCode as an own property — same object-code path as plain objects.
  const unimplemented = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'ni' }).withGrpcCode(GrpcCode.Unimplemented);
  expect(isAuthServiceNotPresent(unimplemented)).toBe(true);
  const notFound = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'nf' }).withGrpcCode(GrpcCode.NotFound);
  expect(isAuthServiceNotPresent(notFound)).toBe(true);
  const unavailable = new ChoysumError({ domain: 'core.auth', code: 'x', message: 'na' }).withGrpcCode(GrpcCode.Unavailable);
  expect(isAuthServiceNotPresent(unavailable)).toBe(false);

  expect(isAuthServiceNotPresent(new Error('no registered proto files for app auth'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('failed to load method descriptor: auth.User/GetRecordRuleCondition'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('rpc error: code = Unimplemented desc = unknown service auth.User'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('rpc error: code = Unimplemented desc = unknown method auth.User/GetFieldRuleSpec'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('rpc error: code = Unimplemented desc = target method does not exist'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('rpc error: code = Unimplemented desc = \u76ee\u6807\u65b9\u6cd5\u4e0d\u5b58\u5728'))).toBe(true);
  expect(isAuthServiceNotPresent(new Error('other runtime error'))).toBe(false);
  expect(isAuthServiceNotPresent('plain-string-error')).toBe(false);
  expect(isAuthServiceNotPresent(null)).toBe(false);
  expect(isAuthServiceNotPresent(undefined)).toBe(false);
  expect(isAuthServiceNotPresent({})).toBe(false);

  // Prefer errRecord.message; fall back to Error.message / empty string when message is missing.
  expect(isAuthServiceNotPresent({ message: 'unknown method auth.User/GetFieldRuleSpec' })).toBe(true);
  const errWithUndefinedMessage = new Error('placeholder');
  Object.defineProperty(errWithUndefinedMessage, 'message', { value: undefined, configurable: true });
  expect(isAuthServiceNotPresent(errWithUndefinedMessage)).toBe(false);
});

test('auth user service methods build grpc request payloads via server bridge', async () => {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as any, key);
  const previous = (globalThis as any)[key];
  const calls: any[] = [];

  (globalThis as any)[key] = {
    grpc: {
      unary: async (service: string, method: string, request: Record<string, any>) => {
        calls.push({ service, method, request });
        return { result: { service, method, request } };
      },
    },
  };

  try {
    const record = await AuthUserService.GetRecordRuleCondition('demo.Model', 'read' as any);
    const field = await AuthUserService.GetFieldRuleSpec('demo.Model');

    expect(record).toEqual({
      service: 'auth.User',
      method: 'GetRecordRuleCondition',
      request: {
        model: 'demo.Model',
        op: 'read',
      },
    });
    expect(field).toEqual({
      service: 'auth.User',
      method: 'GetFieldRuleSpec',
      request: {
        model: 'demo.Model',
      },
    });

    expect(calls).toEqual([
      {
        service: 'auth.User',
        method: 'GetRecordRuleCondition',
        request: {
          model: 'demo.Model',
          op: 'read',
        },
      },
      {
        service: 'auth.User',
        method: 'GetFieldRuleSpec',
        request: {
          model: 'demo.Model',
        },
      },
    ]);
  } finally {
    if (hadOwn) (globalThis as any)[key] = previous;
    else delete (globalThis as any)[key];
  }
});
