// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { isTransportOutage } from './transportOutage';

function connectError(code: Code, message: string): ConnectError {
  return Object.assign(Object.create(ConnectError.prototype), {
    code,
    rawMessage: message,
    message,
    metadata: new Headers(),
    findDetails: () => [],
  }) as ConnectError;
}

test('isTransportOutage: Unavailable is an outage', () => {
  expect(isTransportOutage(connectError(Code.Unavailable, 'unavailable'))).toBe(true);
});

test('isTransportOutage: Unknown with Failed to fetch is an outage', () => {
  expect(isTransportOutage(connectError(Code.Unknown, 'Failed to fetch'))).toBe(true);
  expect(isTransportOutage(connectError(Code.Unknown, 'net::ERR_CONNECTION_REFUSED'))).toBe(true);
});

test('isTransportOutage: business codes are not outages', () => {
  expect(isTransportOutage(connectError(Code.InvalidArgument, 'bad'))).toBe(false);
  expect(isTransportOutage(connectError(Code.NotFound, 'missing'))).toBe(false);
  expect(isTransportOutage(connectError(Code.PermissionDenied, 'no'))).toBe(false);
  expect(isTransportOutage(connectError(Code.Unauthenticated, 'auth'))).toBe(false);
  expect(isTransportOutage(connectError(Code.FailedPrecondition, 'pre'))).toBe(false);
});

test('isTransportOutage: Canceled and AbortError are not outages', () => {
  expect(isTransportOutage(connectError(Code.Canceled, 'canceled'))).toBe(false);
  expect(isTransportOutage({ name: 'AbortError', message: 'aborted' })).toBe(false);
});

test('isTransportOutage: ConnectError with ErrorInfo is business', () => {
  const err = Object.assign(Object.create(ConnectError.prototype), {
    code: Code.Unknown,
    message: 'Failed to fetch',
    metadata: new Headers(),
    findDetails: () => [{ domain: 'api', code: 'X' }],
  }) as ConnectError;
  expect(isTransportOutage(err)).toBe(false);
});

test('isTransportOutage: TypeError Failed to fetch is an outage', () => {
  expect(isTransportOutage(new TypeError('Failed to fetch'))).toBe(true);
});

test('isTransportOutage: DeadlineExceeded is not an outage', () => {
  expect(isTransportOutage(connectError(Code.DeadlineExceeded, 'deadline'))).toBe(false);
});

test('isTransportOutage: ambiguous Load failed alone is not an outage', () => {
  expect(isTransportOutage(new Error('Load failed'))).toBe(false);
  expect(isTransportOutage(new Error('Load failed: Invalid file format'))).toBe(false);
});

test('isTransportOutage: null and AbortError variants', () => {
  expect(isTransportOutage(null)).toBe(false);
  expect(isTransportOutage({ name: 'AbortError', code: 'ABORT_ERR', message: 'aborted' })).toBe(false);
  expect(isTransportOutage('Failed to fetch')).toBe(true);
});

test('isTransportOutage: findDetails throwing is treated as no ErrorInfo', () => {
  const err = Object.assign(Object.create(ConnectError.prototype), {
    code: Code.Unknown,
    message: 'Failed to fetch',
    metadata: new Headers(),
    findDetails: () => {
      throw new Error('detail boom');
    },
  }) as ConnectError;
  expect(isTransportOutage(err)).toBe(true);
});
