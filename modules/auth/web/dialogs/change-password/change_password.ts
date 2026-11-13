// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';

const MIN_PASSWORD_LENGTH = 6;

// Literal `_t` anchors keep RuleItem msgids in the auth catalog.
const { _t: _tChangePasswordRule } = createTranslate('auth', {
  scope: 'web/dialogs/change-password/ChangePasswordDialog',
});
void [
  _tChangePasswordRule('Current password is required'),
  _tChangePasswordRule('New password is required'),
  _tChangePasswordRule('Confirm new password is required'),
  _tChangePasswordRule('Password must be at least 6 characters'),
  _tChangePasswordRule('Passwords do not match'),
  _tChangePasswordRule('New password must differ from the current password'),
];

export type ChangePasswordForm = {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
};

export type ChangePasswordClientRule = {
  required?: boolean;
  message?: string;
  validator?: (rule: unknown, value: unknown, cb: (error?: Error) => void) => void;
};

function ruleFromError(check: (value: unknown) => string): ChangePasswordClientRule {
  return {
    validator: (_rule: unknown, value: unknown, cb: (error?: Error) => void) => {
      const message = check(value);
      if (message) {
        cb(new Error(message));
        return;
      }
      cb();
    },
  };
}

export function changePasswordCurrentError(value: unknown, t: (msg: string) => string): string {
  if (value == null || value === '') return t('Current password is required');
  return '';
}

export function changePasswordNewError(
  value: unknown,
  currentPassword: unknown,
  t: (msg: string) => string,
): string {
  const raw = String(value ?? '');
  if (!raw) return t('New password is required');
  if (raw.length < MIN_PASSWORD_LENGTH) return t('Password must be at least 6 characters');
  if (raw === String(currentPassword ?? '')) {
    return t('New password must differ from the current password');
  }
  return '';
}

export function changePasswordConfirmError(
  value: unknown,
  newPassword: unknown,
  t: (msg: string) => string,
): string {
  const raw = String(value ?? '');
  if (!raw) return t('Confirm new password is required');
  if (raw !== String(newPassword ?? '')) return t('Passwords do not match');
  return '';
}

export function changePasswordCurrentRules(t: (msg: string) => string): ChangePasswordClientRule[] {
  return [ruleFromError(value => changePasswordCurrentError(value, t))];
}

export function changePasswordNewRules(
  t: (msg: string) => string,
  getCurrentPassword: () => unknown,
): ChangePasswordClientRule[] {
  return [ruleFromError(value => changePasswordNewError(value, getCurrentPassword(), t))];
}

export function changePasswordConfirmRules(
  t: (msg: string) => string,
  getNewPassword: () => unknown,
): ChangePasswordClientRule[] {
  return [ruleFromError(value => changePasswordConfirmError(value, getNewPassword(), t))];
}

/**
 * Aggregate client validation (tests / non-field callers).
 * Returns the first error message, or null when valid.
 */
export function validateChangePasswordForm(
  form: ChangePasswordForm,
  messages: {
    currentRequired: string;
    newRequired: string;
    confirmRequired: string;
    tooShort: string;
    mismatch: string;
    sameAsCurrent: string;
  },
): string | null {
  const t = (key: string) => {
    switch (key) {
      case 'Current password is required':
        return messages.currentRequired;
      case 'New password is required':
        return messages.newRequired;
      case 'Confirm new password is required':
        return messages.confirmRequired;
      case 'Password must be at least 6 characters':
        return messages.tooShort;
      case 'Passwords do not match':
        return messages.mismatch;
      case 'New password must differ from the current password':
        return messages.sameAsCurrent;
      default:
        return key;
    }
  };
  return (
    changePasswordCurrentError(form.currentPassword, t) ||
    changePasswordNewError(form.newPassword, form.currentPassword, t) ||
    changePasswordConfirmError(form.confirmPassword, form.newPassword, t) ||
    null
  );
}
