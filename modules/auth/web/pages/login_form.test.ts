// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import {
  formatLoginError,
  resolveLoginRedirect,
  runLoginSubmit,
  validateLoginForm,
  type LoginFieldErrors,
  type LoginFormFields,
} from './login_form';

const t = (msg: string) => msg;

function emptyErrors(): LoginFieldErrors {
  return { username: '', password: '' };
}

test('resolveLoginRedirect: defaults to / when query missing or blank', () => {
  expect(resolveLoginRedirect(undefined)).toBe('/');
  expect(resolveLoginRedirect(null)).toBe('/');
  expect(resolveLoginRedirect('   ')).toBe('/');
});

test('resolveLoginRedirect: keeps same-origin relative paths', () => {
  expect(resolveLoginRedirect('/auth/tokens')).toBe('/auth/tokens');
  expect(resolveLoginRedirect('/web/?tab=1')).toBe('/web/?tab=1');
  expect(resolveLoginRedirect('/web/#section')).toBe('/web/#section');
});

test('resolveLoginRedirect: rejects absolute and protocol-relative URLs', () => {
  expect(resolveLoginRedirect('https://evil.example/phish')).toBe('/');
  expect(resolveLoginRedirect('http://evil.example')).toBe('/');
  expect(resolveLoginRedirect('//evil.example/path')).toBe('/');
  expect(resolveLoginRedirect('auth/tokens')).toBe('/');
});

test('resolveLoginRedirect: rejects backslash open-redirect payloads', () => {
  expect(resolveLoginRedirect('/\\evil.example/phish')).toBe('/');
  expect(resolveLoginRedirect('/\\\\evil.example')).toBe('/');
});

test('validateLoginForm: requires username and password', () => {
  const form: LoginFormFields = { username: '', password: '' };
  const errors = emptyErrors();
  expect(validateLoginForm(form, errors, t)).toBe(false);
  expect(errors.username).toBe('Enter username');
  expect(errors.password).toBe('Enter password');
});

test('validateLoginForm: accepts trimmed non-empty credentials', () => {
  const form: LoginFormFields = { username: '  admin  ', password: 'secret' };
  const errors = emptyErrors();
  expect(validateLoginForm(form, errors, t)).toBe(true);
  expect(errors.username).toBe('');
  expect(errors.password).toBe('');
});

test('formatLoginError: uses ChoysumError message', () => {
  const err = new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: 'bad creds' });
  expect(formatLoginError(err, 'fallback')).toBe('bad creds');
});

test('formatLoginError: falls back for unknown errors', () => {
  expect(formatLoginError(new Error('boom'), 'Login failed. Please try again later.')).toBe(
    'Login failed. Please try again later.',
  );
});

test('runLoginSubmit: skips while loading', async () => {
  let calls = 0;
  const ok = await runLoginSubmit({
    loading: true,
    form: { username: 'a', password: 'b' },
    fieldErrors: emptyErrors(),
    t,
    loginFailedMessage: 'Login failed. Please try again later.',
    login: async () => {
      calls += 1;
    },
    rememberMe: true,
    setError: () => undefined,
  });
  expect(ok).toBe(false);
  expect(calls).toBe(0);
});

test('runLoginSubmit: skips invalid form', async () => {
  let calls = 0;
  const ok = await runLoginSubmit({
    loading: false,
    form: { username: '', password: '' },
    fieldErrors: emptyErrors(),
    t,
    loginFailedMessage: 'Login failed. Please try again later.',
    login: async () => {
      calls += 1;
    },
    rememberMe: true,
    setError: () => undefined,
  });
  expect(ok).toBe(false);
  expect(calls).toBe(0);
});

test('runLoginSubmit: returns true on success', async () => {
  const errors: string[] = [];
  const ok = await runLoginSubmit({
    loading: false,
    form: { username: 'admin', password: 'secret' },
    fieldErrors: emptyErrors(),
    t,
    loginFailedMessage: 'Login failed. Please try again later.',
    login: async () => undefined,
    rememberMe: false,
    setError: m => errors.push(m),
  });
  expect(ok).toBe(true);
  expect(errors).toEqual(['']);
});

test('runLoginSubmit: maps failures to setError', async () => {
  const errors: string[] = [];
  const ok = await runLoginSubmit({
    loading: false,
    form: { username: 'admin', password: 'secret' },
    fieldErrors: emptyErrors(),
    t,
    loginFailedMessage: 'Login failed. Please try again later.',
    login: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: 'nope' });
    },
    rememberMe: true,
    setError: m => errors.push(m),
  });
  expect(ok).toBe(false);
  expect(errors).toEqual(['', 'nope']);
});
