// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import {
  registerAgreeTermsError,
  registerConfirmPasswordError,
  registerEmailError,
  registerPasswordError,
  registerUsernameError,
  runRegisterSubmit,
  validateRegisterForm,
} from './register_form';

const t = (msg: string) => msg;

test('registerUsernameError: required, length, and charset', () => {
  expect(registerUsernameError('', t)).toBe('Enter username');
  expect(registerUsernameError('ab', t)).toBe('Username must be at least 3 characters');
  expect(registerUsernameError('bad name', t)).toBe(
    'Username can only contain letters, numbers, underscores, hyphens, and dots',
  );
  expect(registerUsernameError('ok_user-1.x', t)).toBe('');
});

test('registerEmailError: required and format', () => {
  expect(registerEmailError('', t)).toBe('Enter email address');
  expect(registerEmailError('not-an-email', t)).toBe('Enter a valid email address');
  expect(registerEmailError('user@example.com', t)).toBe('');
});

test('registerPasswordError: required and min length', () => {
  expect(registerPasswordError('', t)).toBe('Enter password');
  expect(registerPasswordError('12345', t)).toBe('Password must be at least 6 characters');
  expect(registerPasswordError('123456', t)).toBe('');
});

test('registerConfirmPasswordError: required and match', () => {
  expect(registerConfirmPasswordError('', 'secret', t)).toBe('Re-enter password');
  expect(registerConfirmPasswordError('other', 'secret', t)).toBe('Passwords do not match');
  expect(registerConfirmPasswordError('secret', 'secret', t)).toBe('');
});

test('registerAgreeTermsError: must be true', () => {
  expect(registerAgreeTermsError(false, t)).toBe(
    'You must agree to the Terms of Service and Privacy Policy',
  );
  expect(registerAgreeTermsError(true, t)).toBe('');
});

test('validateRegisterForm: collects all field errors', () => {
  const errors = validateRegisterForm(
    { username: '', email: '', password: '', confirmPassword: '', agreeTerms: false },
    t,
  );
  expect(errors.length).toBe(5);
  expect(
    validateRegisterForm(
      {
        username: 'alice',
        email: 'alice@example.com',
        password: 'secret1',
        confirmPassword: 'secret1',
        agreeTerms: true,
      },
      t,
    ),
  ).toEqual([]);
});

test('runRegisterSubmit: skips while loading', async () => {
  let calls = 0;
  const ok = await runRegisterSubmit({
    loading: true,
    username: 'alice',
    email: 'a@b.co',
    password: 'secret1',
    registerFailedMessage: 'Registration failed. Please try again later.',
    register: async () => {
      calls += 1;
    },
    login: async () => {
      calls += 1;
    },
    setError: () => undefined,
  });
  expect(ok).toBe(false);
  expect(calls).toBe(0);
});

test('runRegisterSubmit: skips empty credentials', async () => {
  let calls = 0;
  const ok = await runRegisterSubmit({
    loading: false,
    username: '  ',
    email: '',
    password: '',
    registerFailedMessage: 'Registration failed. Please try again later.',
    register: async () => {
      calls += 1;
    },
    login: async () => {
      calls += 1;
    },
    setError: () => undefined,
  });
  expect(ok).toBe(false);
  expect(calls).toBe(0);
});

test('runRegisterSubmit: register then login on success', async () => {
  const seen: string[] = [];
  const ok = await runRegisterSubmit({
    loading: false,
    username: 'alice',
    email: 'a@b.co',
    password: 'secret1',
    registerFailedMessage: 'Registration failed. Please try again later.',
    register: async (username, email, password) => {
      seen.push('reg', username, email, password);
    },
    login: async (username, password) => {
      seen.push('login', username, password);
    },
    setError: () => undefined,
  });
  expect(ok).toBe(true);
  expect(seen).toEqual(['reg', 'alice', 'a@b.co', 'secret1', 'login', 'alice', 'secret1']);
});

test('runRegisterSubmit: maps ChoysumError to setError', async () => {
  const errors: string[] = [];
  const ok = await runRegisterSubmit({
    loading: false,
    username: 'alice',
    email: 'a@b.co',
    password: 'secret1',
    registerFailedMessage: 'Registration failed. Please try again later.',
    register: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'REGISTRATION_FAILED', message: 'taken' });
    },
    login: async () => undefined,
    setError: m => errors.push(m),
  });
  expect(ok).toBe(false);
  expect(errors).toEqual(['', 'taken']);
});

test('runRegisterSubmit: unknown errors use fallback', async () => {
  const errors: string[] = [];
  const ok = await runRegisterSubmit({
    loading: false,
    username: 'alice',
    email: 'a@b.co',
    password: 'secret1',
    registerFailedMessage: 'Registration failed. Please try again later.',
    register: async () => {
      throw new Error('boom');
    },
    login: async () => undefined,
    setError: m => errors.push(m),
  });
  expect(ok).toBe(false);
  expect(errors).toEqual(['', 'Registration failed. Please try again later.']);
});
