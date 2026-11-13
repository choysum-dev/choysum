// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { hashPassword } from './_authz_shared';
import { buildChangedPasswordHash, normalizeChangePasswordInput } from './_change_password';

test('normalizeChangePasswordInput: requires both passwords', () => {
  expect(() => normalizeChangePasswordInput({})).toThrow(/Current password/);
  expect(() => normalizeChangePasswordInput({ CurrentPassword: 'a' })).toThrow(/New password/);
});

test('normalizeChangePasswordInput: enforces plaintext min length and difference', () => {
  expect(() =>
    normalizeChangePasswordInput({ CurrentPassword: 'oldpass', NewPassword: 'short' }),
  ).toThrow(/at least 6/);
  expect(() =>
    normalizeChangePasswordInput({ CurrentPassword: 'same123', NewPassword: 'same123' }),
  ).toThrow(/differ/);
  expect(
    normalizeChangePasswordInput({ CurrentPassword: 'oldpass', NewPassword: 'newpass' }),
  ).toEqual({ currentPassword: 'oldpass', newPassword: 'newpass' });
});

test('normalizeChangePasswordInput: rejects malformed $CH$ client hashes', () => {
  expect(() =>
    normalizeChangePasswordInput({ CurrentPassword: 'oldpass', NewPassword: '$CH$a' }),
  ).toThrow(/client hash/);
  const ok =
    '$CH$' +
    '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef';
  expect(
    normalizeChangePasswordInput({
      CurrentPassword: 'oldpass',
      NewPassword: ok,
    }),
  ).toEqual({ currentPassword: 'oldpass', newPassword: ok });
});

test('buildChangedPasswordHash: rejects wrong current password', () => {
  const passwordHash = hashPassword('correct');
  expect(() =>
    buildChangedPasswordHash({
      currentPassword: 'wrong',
      newPassword: 'newpass',
      passwordHash,
    }),
  ).toThrow(/incorrect/);
});

test('buildChangedPasswordHash: rejects empty stored hash and marker-equivalent new password', () => {
  expect(() =>
    buildChangedPasswordHash({
      currentPassword: 'correct',
      newPassword: 'newpass',
      passwordHash: '',
    }),
  ).toThrow(/incorrect/);

  const passwordHash = hashPassword('correct');
  expect(() =>
    buildChangedPasswordHash({
      currentPassword: 'correct',
      newPassword: 'correct',
      passwordHash,
    }),
  ).toThrow(/differ/);
});

test('buildChangedPasswordHash: returns a new hash for a valid current password', () => {
  const passwordHash = hashPassword('correct');
  const next = buildChangedPasswordHash({
    currentPassword: 'correct',
    newPassword: 'newpass',
    passwordHash,
  });
  expect(typeof next).toBe('string');
  expect(next.length).toBeGreaterThan(0);
  expect(next).not.toBe(passwordHash);
});
