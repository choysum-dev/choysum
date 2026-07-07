// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getBackendEnv, getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';

test('getBackendEnv returns a non-null object', () => {
  const env = getBackendEnv();
  expect(typeof env).toBe('object');
  expect(env).not.toBe(null);
});

test('getBackendEnvText returns empty string for nonexistent keys', () => {
  expect(getBackendEnvText('NONEXISTENT_KEY_12345')).toBe('');
});

test('isTruthyFlag recognizes truthy and falsy values', () => {
  expect(isTruthyFlag('1')).toBe(true);
  expect(isTruthyFlag('true')).toBe(true);
  expect(isTruthyFlag('TRUE')).toBe(true);
  expect(isTruthyFlag('yes')).toBe(true);
  expect(isTruthyFlag('YES')).toBe(true);
  expect(isTruthyFlag('on')).toBe(true);
  expect(isTruthyFlag('ON')).toBe(true);

  expect(isTruthyFlag('0')).toBe(false);
  expect(isTruthyFlag('false')).toBe(false);
  expect(isTruthyFlag('no')).toBe(false);
  expect(isTruthyFlag('off')).toBe(false);
  expect(isTruthyFlag('')).toBe(false);
  expect(isTruthyFlag('  random  ')).toBe(false);
});
