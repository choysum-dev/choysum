// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import {
  formatLoginError,
  isUnsafeLoginRedirectPath,
  normalizeLoginRedirectOrigin,
  resolveLoginRedirect,
  resolveLoginRedirectOrigin,
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

test('resolveLoginRedirect: rejects normalization that yields protocol-relative paths', () => {
  expect(resolveLoginRedirect('/..//evil.example')).toBe('/');
  expect(resolveLoginRedirect('/a/..//evil.com')).toBe('/');
  expect(resolveLoginRedirect('/%2e%2e/%2e%2e//evil.com')).toBe('/');
});

test('resolveLoginRedirect: rejects percent-encoded separators that decode to // or \\', () => {
  expect(resolveLoginRedirect('/%2f%2fevil.com')).toBe('/');
  expect(resolveLoginRedirect('/%2F%2Fevil.com')).toBe('/');
  expect(resolveLoginRedirect('/ok%2f%2fevil')).toBe('/');
  expect(resolveLoginRedirect('/ok%5cevil')).toBe('/');
});

test('resolveLoginRedirect: keeps path when decodeURIComponent throws', () => {
  // Lone `%` is invalid percent-encoding; fail open to the encoded pathname checks.
  expect(
    resolveLoginRedirect('/ok', {
      origin: 'http://localhost',
      createUrl: () => ({
        origin: 'http://localhost',
        pathname: '/%E0%A4%A',
        search: '',
        hash: '',
      }),
    }),
  ).toBe('/%E0%A4%A');
});

test('isUnsafeLoginRedirectPath: flags protocol-relative, empty segments, slash, and ..', () => {
  expect(isUnsafeLoginRedirectPath('/ok')).toBe(false);
  expect(isUnsafeLoginRedirectPath('//evil')).toBe(true);
  expect(isUnsafeLoginRedirectPath('/a//b')).toBe(true);
  expect(isUnsafeLoginRedirectPath('/a\\b')).toBe(true);
  expect(isUnsafeLoginRedirectPath('/../x')).toBe(true);
  expect(isUnsafeLoginRedirectPath('relative')).toBe(true);
});

test('resolveLoginRedirect: rejects off-origin URL results', () => {
  expect(
    resolveLoginRedirect('/ok', {
      origin: 'http://localhost',
      createUrl: () => ({
        origin: 'https://evil.example',
        pathname: '/ok',
        search: '',
        hash: '',
      }),
    }),
  ).toBe('/');
});

test('resolveLoginRedirect: fails closed when URL construction throws', () => {
  expect(
    resolveLoginRedirect('/ok', {
      createUrl: () => {
        throw new Error('bad url');
      },
    }),
  ).toBe('/');
});

test('resolveLoginRedirect: honors explicit origin override', () => {
  expect(resolveLoginRedirect('/home', { origin: 'https://app.example' })).toBe('/home');
  // Trailing slash / path on the override must not fail closed for same-origin paths.
  expect(resolveLoginRedirect('/home', { origin: 'https://app.example/' })).toBe('/home');
  expect(resolveLoginRedirect('/home', { origin: 'https://app.example/app' })).toBe('/home');
});

test('normalizeLoginRedirectOrigin: strips path and trailing slash', () => {
  expect(normalizeLoginRedirectOrigin('https://app.example/')).toBe('https://app.example');
  expect(normalizeLoginRedirectOrigin('https://app.example/app')).toBe('https://app.example');
  expect(normalizeLoginRedirectOrigin('not a url')).toBe('not a url');
});

test('resolveLoginRedirectOrigin: uses window origin or localhost fallback', () => {
  expect(resolveLoginRedirectOrigin({ windowOrigin: 'https://app.example' })).toBe('https://app.example');
  expect(resolveLoginRedirectOrigin({ windowOrigin: '  ' })).toBe('http://localhost');
  expect(resolveLoginRedirectOrigin({ windowOrigin: null })).toBe('http://localhost');
  expect(resolveLoginRedirectOrigin()).toMatch(/^https?:\/\//);
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

test('formatLoginError: falls back when ChoysumError message is empty', () => {
  const err = new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: '' });
  expect(formatLoginError(err, 'fallback')).toBe('fallback');
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

test('runLoginSubmit: returns true on success and trims username', async () => {
  const errors: string[] = [];
  const seen: string[] = [];
  const ok = await runLoginSubmit({
    loading: false,
    form: { username: '  admin  ', password: 'secret' },
    fieldErrors: emptyErrors(),
    t,
    loginFailedMessage: 'Login failed. Please try again later.',
    login: async (username, password) => {
      seen.push(username, password);
    },
    rememberMe: false,
    setError: m => errors.push(m),
  });
  expect(ok).toBe(true);
  expect(errors).toEqual(['']);
  expect(seen).toEqual(['admin', 'secret']);
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
