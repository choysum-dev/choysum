// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';

export type LoginFormFields = {
  username: string;
  password: string;
};

export type LoginFieldErrors = {
  username: string;
  password: string;
};

/**
 * Resolve the post-login destination from the route redirect query.
 * Only same-origin relative paths are accepted; absolute and protocol-relative
 * URLs fall back to `/` so `?redirect=` cannot bounce users off-site.
 */
export function resolveLoginRedirect(redirectQuery: string | undefined | null): string {
  const redirect = String(redirectQuery ?? '').trim();
  if (redirect.startsWith('/') && !redirect.startsWith('//')) {
    return redirect;
  }
  return '/';
}

/**
 * Validate required login fields and write field-level error messages.
 */
export function validateLoginForm(
  form: LoginFormFields,
  fieldErrors: LoginFieldErrors,
  t: (msg: string) => string,
): boolean {
  fieldErrors.username = '';
  fieldErrors.password = '';
  let ok = true;
  if (!form.username.trim()) {
    fieldErrors.username = t('Enter username');
    ok = false;
  }
  if (!form.password) {
    fieldErrors.password = t('Enter password');
    ok = false;
  }
  return ok;
}

/**
 * Map a login failure to a user-visible error message.
 * `fallback` should already be translated by the caller (keeps msgid extractable).
 */
export function formatLoginError(err: unknown, fallback: string): string {
  if (err instanceof ChoysumError) {
    return err.message;
  }
  console.error('Login flow failed:', err);
  return fallback;
}

/**
 * Run the login store call and return whether redirect should follow.
 * Returns false when validation fails or a submit is already in flight.
 */
export async function runLoginSubmit(opts: {
  loading: boolean;
  form: LoginFormFields;
  fieldErrors: LoginFieldErrors;
  t: (msg: string) => string;
  loginFailedMessage: string;
  login: (username: string, password: string, csrf: string, device: string, rememberMe: boolean) => Promise<void>;
  rememberMe: boolean;
  setError: (message: string) => void;
}): Promise<boolean> {
  if (opts.loading) return false;
  if (!validateLoginForm(opts.form, opts.fieldErrors, opts.t)) return false;
  try {
    opts.setError('');
    await opts.login(opts.form.username, opts.form.password, '', '', opts.rememberMe);
    return true;
  } catch (err) {
    opts.setError(formatLoginError(err, opts.loginFailedMessage));
    return false;
  }
}
