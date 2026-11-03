// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import { runSetupSubmit, setupUsernameError, setupPasswordError, setupConfirmPasswordError } from './setup_form';

test('setupUsernameError rejects short or empty names', () => {
  const t = (s: string) => s;
  expect(setupUsernameError('', t)).toBe('Enter username');
  expect(setupUsernameError('ab', t)).toBe('Username must be at least 3 characters');
  expect(setupUsernameError('admin', t)).toBe('');
});

test('setupPasswordError requires min length', () => {
  const t = (s: string) => s;
  expect(setupPasswordError('', t)).toBe('Enter password');
  expect(setupPasswordError('12345', t)).toBe('Password must be at least 6 characters');
  expect(setupPasswordError('secret1', t)).toBe('');
});

test('setupConfirmPasswordError requires match', () => {
  const t = (s: string) => s;
  expect(setupConfirmPasswordError('', 'secret1', t)).toBe('Re-enter password');
  expect(setupConfirmPasswordError('other', 'secret1', t)).toBe('Passwords do not match');
  expect(setupConfirmPasswordError('secret1', 'secret1', t)).toBe('');
});

test('runSetupSubmit completes then logs in', async () => {
  const calls: string[] = [];
  const ok = await runSetupSubmit({
    loading: false,
    username: ' admin ',
    password: 'secret1',
    setupFailedMessage: 'setup failed',
    loginFailedMessage: 'login failed',
    completeInitialSetup: async (u, p) => {
      calls.push(`setup:${u}:${p}`);
    },
    login: async (u, p) => {
      calls.push(`login:${u}:${p}`);
    },
    setError: () => {},
  });
  expect(ok).toBe(true);
  expect(calls).toEqual(['setup:admin:secret1', 'login:admin:secret1']);
});

test('runSetupSubmit surfaces CompleteInitialSetup errors', async () => {
  let message = '';
  const ok = await runSetupSubmit({
    loading: false,
    username: 'admin',
    password: 'secret1',
    setupFailedMessage: 'setup failed',
    loginFailedMessage: 'login failed',
    completeInitialSetup: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'VALIDATION_FAILED', message: 'already done' });
    },
    login: async () => {
      throw new Error('should not login');
    },
    setError: m => {
      message = m;
    },
  });
  expect(ok).toBe(false);
  expect(message).toBe('already done');
});

test('runSetupSubmit surfaces login errors after successful setup', async () => {
  let message = '';
  const ok = await runSetupSubmit({
    loading: false,
    username: 'admin',
    password: 'secret1',
    setupFailedMessage: 'setup failed',
    loginFailedMessage: 'login failed',
    completeInitialSetup: async () => {},
    login: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: 'bad login' });
    },
    setError: m => {
      message = m;
    },
  });
  expect(ok).toBe(false);
  expect(message).toBe('bad login');
});

test('runSetupSubmit skips while loading or when credentials are empty', async () => {
  expect(
    await runSetupSubmit({
      loading: true,
      username: 'admin',
      password: 'secret1',
      setupFailedMessage: 'setup failed',
      loginFailedMessage: 'login failed',
      completeInitialSetup: async () => {
        throw new Error('no');
      },
      login: async () => {
        throw new Error('no');
      },
      setError: () => {},
    }),
  ).toBe(false);

  expect(
    await runSetupSubmit({
      loading: false,
      username: '  ',
      password: 'secret1',
      setupFailedMessage: 'setup failed',
      loginFailedMessage: 'login failed',
      completeInitialSetup: async () => {
        throw new Error('no');
      },
      login: async () => {
        throw new Error('no');
      },
      setError: () => {},
    }),
  ).toBe(false);
});
