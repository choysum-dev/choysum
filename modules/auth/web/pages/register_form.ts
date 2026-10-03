// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';

export type RegisterClientRule = {
  required?: boolean;
  message?: string;
  validator?: (rule: unknown, value: unknown, cb: (error?: Error) => void) => void;
};

export type RegisterFormFields = {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
  agreeTerms: boolean;
};

function ruleFromError(check: (value: unknown) => string): RegisterClientRule {
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

export function registerUsernameError(value: unknown, t: (msg: string) => string): string {
  const raw = String(value ?? '');
  if (!raw) return t('Enter username');
  if (raw.length < 3) return t('Username must be at least 3 characters');
  if (!/^[a-zA-Z0-9_\-\.]+$/.test(raw)) {
    return t('Username can only contain letters, numbers, underscores, hyphens, and dots');
  }
  return '';
}

export function registerEmailError(value: unknown, t: (msg: string) => string): string {
  const raw = String(value ?? '');
  if (!raw) return t('Enter email address');
  if (!/^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/.test(raw)) {
    return t('Enter a valid email address');
  }
  return '';
}

export function registerPasswordError(value: unknown, t: (msg: string) => string): string {
  const raw = String(value ?? '');
  if (!raw) return t('Enter password');
  if (raw.length < 6) return t('Password must be at least 6 characters');
  return '';
}

export function registerConfirmPasswordError(
  value: unknown,
  password: unknown,
  t: (msg: string) => string,
): string {
  const raw = String(value ?? '');
  if (!raw) return t('Re-enter password');
  if (raw !== String(password ?? '')) return t('Passwords do not match');
  return '';
}

export function registerAgreeTermsError(value: unknown, t: (msg: string) => string): string {
  if (value !== true) {
    return t('You must agree to the Terms of Service and Privacy Policy');
  }
  return '';
}

export function registerUsernameRules(t: (msg: string) => string): RegisterClientRule[] {
  return [ruleFromError(value => registerUsernameError(value, t))];
}

export function registerEmailRules(t: (msg: string) => string): RegisterClientRule[] {
  return [ruleFromError(value => registerEmailError(value, t))];
}

export function registerPasswordRules(t: (msg: string) => string): RegisterClientRule[] {
  return [ruleFromError(value => registerPasswordError(value, t))];
}

export function registerConfirmPasswordRules(
  t: (msg: string) => string,
  getPassword: () => unknown,
): RegisterClientRule[] {
  return [ruleFromError(value => registerConfirmPasswordError(value, getPassword(), t))];
}

export function registerAgreeTermsRules(t: (msg: string) => string): RegisterClientRule[] {
  return [ruleFromError(value => registerAgreeTermsError(value, t))];
}

export function validateRegisterForm(form: RegisterFormFields, t: (msg: string) => string): string[] {
  return [
    registerUsernameError(form.username, t),
    registerEmailError(form.email, t),
    registerPasswordError(form.password, t),
    registerConfirmPasswordError(form.confirmPassword, form.password, t),
    registerAgreeTermsError(form.agreeTerms, t),
  ].filter(Boolean);
}

/**
 * Create the account then log in. Field rules run in FormView before this.
 */
export async function runRegisterSubmit(opts: {
  loading: boolean;
  username: string;
  email: string;
  password: string;
  registerFailedMessage: string;
  register: (username: string, email: string, password: string) => Promise<unknown>;
  login: (username: string, password: string) => Promise<unknown>;
  setError: (message: string) => void;
}): Promise<boolean> {
  if (opts.loading) return false;
  opts.setError('');
  try {
    await opts.register(opts.username, opts.email, opts.password);
    await opts.login(opts.username, opts.password);
    return true;
  } catch (err) {
    if (err instanceof ChoysumError) {
      opts.setError(err.message || opts.registerFailedMessage);
    } else {
      console.error('Registration flow failed:', err);
      opts.setError(opts.registerFailedMessage);
    }
    return false;
  }
}
