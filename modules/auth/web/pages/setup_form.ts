// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import { createTranslate } from '@/web/web/i18n';
import { formatLoginError } from './login_form';
import {
  registerConfirmPasswordError,
  registerConfirmPasswordRules,
  registerPasswordError,
  registerPasswordRules,
  registerUsernameError,
  registerUsernameRules,
  type RegisterClientRule,
} from './register_form';

// Literal `_t` anchors keep RuleItem msgids in the auth catalog.
const { _t: _tSetupRule } = createTranslate('auth', { scope: 'web/pages/Setup' });
void [
  _tSetupRule('Enter username'),
  _tSetupRule('Username must be at least 3 characters'),
  _tSetupRule('Username can only contain letters, numbers, underscores, hyphens, and dots'),
  _tSetupRule('Enter password'),
  _tSetupRule('Password must be at least 6 characters'),
  _tSetupRule('Re-enter password'),
  _tSetupRule('Passwords do not match'),
];

export type SetupClientRule = RegisterClientRule;

/** Login route after setup wrote the marker but auto-login failed. */
export function setupLoginFallbackTarget(
  query: Record<string, unknown> | undefined,
): { name: 'login'; query: Record<string, string> } {
  const out: Record<string, string> = {};
  const src = query || {};
  for (const key of Object.keys(src)) {
    const value = src[key];
    if (value == null) continue;
    out[key] = Array.isArray(value) ? String(value[0] ?? '') : String(value);
  }
  return { name: 'login', query: out };
}

export function setupUsernameRules(t: (msg: string) => string): SetupClientRule[] {
  return registerUsernameRules(t);
}

export function setupPasswordRules(t: (msg: string) => string): SetupClientRule[] {
  return registerPasswordRules(t);
}

export function setupConfirmPasswordRules(
  t: (msg: string) => string,
  getPassword: () => unknown,
): SetupClientRule[] {
  return registerConfirmPasswordRules(t, getPassword);
}

export function setupUsernameError(value: unknown, t: (msg: string) => string): string {
  return registerUsernameError(value, t);
}

export function setupPasswordError(value: unknown, t: (msg: string) => string): string {
  return registerPasswordError(value, t);
}

export function setupConfirmPasswordError(
  value: unknown,
  password: unknown,
  t: (msg: string) => string,
): string {
  return registerConfirmPasswordError(value, password, t);
}

/**
 * Complete first-run setup then sign in. Field rules run in FormView before this;
 * empty credentials are a last guard if submit bypasses those rules.
 */
export async function runSetupSubmit(opts: {
  loading: boolean;
  username: string;
  password: string;
  setupFailedMessage: string;
  loginFailedMessage: string;
  completeInitialSetup: (username: string, password: string) => Promise<unknown>;
  login: (username: string, password: string) => Promise<unknown>;
  setError: (message: string) => void;
  /** Fired when setup RPC succeeded but the follow-up Login failed. */
  onSetupCompleteLoginFailed?: () => void;
}): Promise<boolean> {
  if (opts.loading) return false;
  opts.setError('');
  const username = String(opts.username ?? '').trim();
  if (!username || !opts.password) {
    return false;
  }
  try {
    await opts.completeInitialSetup(username, opts.password);
  } catch (err) {
    if (err instanceof ChoysumError) {
      opts.setError(err.message || opts.setupFailedMessage);
    } else {
      console.error('Initial setup flow failed:', err);
      opts.setError(opts.setupFailedMessage);
    }
    return false;
  }
  try {
    await opts.login(username, opts.password);
    return true;
  } catch (err) {
    opts.setError(formatLoginError(err, opts.loginFailedMessage));
    opts.onSetupCompleteLoginFailed?.();
    return false;
  }
}
