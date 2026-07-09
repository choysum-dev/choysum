// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getBackendEnv, getBackendEnvPositiveInt, getBackendEnvText, isTruthyFlag } from '@/core/service/runtime/env/backend_env';

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

test('getBackendEnvPositiveInt parses positive values and falls back for invalid values', () => {
  const envKey = 'CHOYSUM_TEST_POSITIVE_INT';
  const globalAny = globalThis as any;
  const bucketKey = '__choysumBackendEnv';
  if (!globalAny[bucketKey]) globalAny[bucketKey] = {};

  const prev = globalAny[bucketKey][envKey];
  try {
    globalAny[bucketKey][envKey] = '12';
    expect(getBackendEnvPositiveInt(envKey, 7)).toBe(12);

    globalAny[bucketKey][envKey] = 9.9;
    expect(getBackendEnvPositiveInt(envKey, 7)).toBe(9);

    globalAny[bucketKey][envKey] = 'bad';
    expect(getBackendEnvPositiveInt(envKey, 7)).toBe(7);

    globalAny[bucketKey][envKey] = 0;
    expect(getBackendEnvPositiveInt(envKey, 7)).toBe(7);
  } finally {
    if (prev === undefined) {
      delete globalAny[bucketKey][envKey];
    } else {
      globalAny[bucketKey][envKey] = prev;
    }
  }
});
