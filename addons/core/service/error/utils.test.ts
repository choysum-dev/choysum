// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { generateErrorId, validateErrorCode } from '../../error/utils';

test('error utils generateErrorId uses $choysum.xid.New when available', () => {
  const originalChoysum = (globalThis as any).$choysum;

  try {
    (globalThis as any).$choysum = {
      xid: {
        New: () => 'XID_FROM_RUNTIME',
      },
    };

    expect(generateErrorId()).toBe('XID_FROM_RUNTIME');
  } finally {
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('error utils generateErrorId falls back to browser-style id when $choysum runtime is unavailable', () => {
  const originalChoysum = (globalThis as any).$choysum;

  try {
    delete (globalThis as any).$choysum;
    const id = generateErrorId();
    expect(typeof id).toBe('string');
    expect(id.startsWith('err_')).toBe(true);
    expect(id.length > 8).toBe(true);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('error utils validateErrorCode accepts uppercase underscore and rejects invalid formats', () => {
  validateErrorCode('VALID_CODE');

  let message = '';
  try {
    validateErrorCode('invalid-code' as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('Invalid error code format')).toBe(true);
});
