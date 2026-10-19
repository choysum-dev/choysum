// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import { formatLogoutError, nextLogoutCountdown, stopLogoutRedirectTimer } from './logout_page';

test('formatLogoutError: uses ChoysumError message', () => {
  const err = new ChoysumError({ domain: 'auth', code: 'LOGOUT_FAILED', message: 'session gone' });
  expect(formatLogoutError(err, 'unknown')).toBe('session gone');
});

test('formatLogoutError: uses Error message', () => {
  expect(formatLogoutError(new Error('network'), 'unknown')).toBe('network');
});

test('formatLogoutError: falls back for unknown values', () => {
  expect(formatLogoutError('nope', 'unknown')).toBe('unknown');
});

test('nextLogoutCountdown: decrements until redirect', () => {
  expect(nextLogoutCountdown(3)).toEqual({ countdown: 2, done: false });
  expect(nextLogoutCountdown(1)).toEqual({ countdown: 0, done: true });
});

test('stopLogoutRedirectTimer: no-ops without a timer', () => {
  expect(stopLogoutRedirectTimer(undefined)).toBe(undefined);
});

test('stopLogoutRedirectTimer: clears an interval id', () => {
  const id = setInterval(() => undefined, 60_000);
  expect(stopLogoutRedirectTimer(id)).toBe(undefined);
});
