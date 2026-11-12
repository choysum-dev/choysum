// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  changePasswordConfirmError,
  changePasswordCurrentError,
  changePasswordNewError,
  validateChangePasswordForm,
} from './change_password';

const messages = {
  currentRequired: 'current',
  newRequired: 'new',
  confirmRequired: 'confirm',
  tooShort: 'short',
  mismatch: 'mismatch',
  sameAsCurrent: 'same',
};

const t = (msg: string) => msg;

test('changePassword field errors: required / length / mismatch / same-as-current', () => {
  expect(changePasswordCurrentError('', t)).toBe('Current password is required');
  expect(changePasswordNewError('', 'old', t)).toBe('New password is required');
  expect(changePasswordNewError('abc', 'old', t)).toBe('Password must be at least 6 characters');
  expect(changePasswordNewError('same12', 'same12', t)).toBe(
    'New password must differ from the current password',
  );
  expect(changePasswordConfirmError('', 'newpass', t)).toBe('Confirm new password is required');
  expect(changePasswordConfirmError('other', 'newpass', t)).toBe('Passwords do not match');
  expect(changePasswordConfirmError('newpass', 'newpass', t)).toBe('');
});

test('validateChangePasswordForm: required fields', () => {
  expect(validateChangePasswordForm({ currentPassword: '', newPassword: 'abcdef', confirmPassword: 'abcdef' }, messages)).toBe(
    'current',
  );
  expect(validateChangePasswordForm({ currentPassword: 'old', newPassword: '', confirmPassword: '' }, messages)).toBe('new');
});

test('validateChangePasswordForm: length mismatch and same-as-current', () => {
  expect(
    validateChangePasswordForm(
      { currentPassword: 'oldpass', newPassword: 'abc', confirmPassword: 'abc' },
      messages,
    ),
  ).toBe('short');
  expect(
    validateChangePasswordForm(
      { currentPassword: 'oldpass', newPassword: 'newpass', confirmPassword: 'other' },
      messages,
    ),
  ).toBe('mismatch');
  expect(
    validateChangePasswordForm(
      { currentPassword: 'same12', newPassword: 'same12', confirmPassword: 'same12' },
      messages,
    ),
  ).toBe('same');
  expect(
    validateChangePasswordForm(
      { currentPassword: 'oldpass', newPassword: 'newpass', confirmPassword: 'newpass' },
      messages,
    ),
  ).toBe(null);
});
