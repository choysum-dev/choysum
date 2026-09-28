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

export type ResolveLoginRedirectDeps = {
  /** Override page origin (tests / non-browser hosts). */
  origin?: string;
  /** Override URL construction so failure paths are testable. */
  createUrl?: (input: string, base: string) => { origin: string; pathname: string; search: string; hash: string };
};

/**
 * Resolve the origin used when parsing relative login redirects.
 * Empty / missing window origins fall back to `http://localhost`.
 */
export function resolveLoginRedirectOrigin(opts?: { windowOrigin?: string | null }): string {
  const fromWindow =
    opts && 'windowOrigin' in opts
      ? opts.windowOrigin
      : typeof window !== 'undefined'
        ? window.location?.origin
        : undefined;
  const origin = String(fromWindow ?? '').trim();
  return origin || 'http://localhost';
}

/**
 * Resolve the post-login destination from the route redirect query.
 * Only same-origin relative paths are accepted; absolute, protocol-relative,
 * backslash, and normalization-induced `//host` payloads fall back to `/`.
 */
export function resolveLoginRedirect(
  redirectQuery: string | undefined | null,
  deps?: ResolveLoginRedirectDeps,
): string {
  const redirect = String(redirectQuery ?? '').trim();
  if (!redirect.startsWith('/') || redirect.startsWith('//') || redirect.includes('\\')) {
    return '/';
  }
  try {
    const base = deps?.origin || resolveLoginRedirectOrigin();
    // Compare against a normalized origin so a caller-supplied base with a
    // trailing slash or path cannot fail closed for a valid redirect.
    const expectedOrigin = normalizeLoginRedirectOrigin(base);
    const createUrl = deps?.createUrl || ((input: string, origin: string) => new URL(input, origin));
    const url = createUrl(redirect, base);
    const pathOnly = String(url.pathname || '');
    // Percent-encoded separators (`%2f`, `%5c`) survive URL parsing; re-check the
    // decoded path so a downstream consumer cannot re-interpret it as `//host`.
    let decodedPath = pathOnly;
    try {
      decodedPath = decodeURIComponent(pathOnly);
    } catch {
      decodedPath = pathOnly;
    }
    if (
      url.origin !== expectedOrigin ||
      isUnsafeLoginRedirectPath(pathOnly) ||
      isUnsafeLoginRedirectPath(decodedPath)
    ) {
      return '/';
    }
    return `${pathOnly}${url.search}${url.hash}`;
  } catch {
    return '/';
  }
}

/**
 * True when a path is not a safe same-document relative redirect target.
 * Covers protocol-relative (`//`), empty segments, backslash, and `..` traversal.
 */
export function isUnsafeLoginRedirectPath(path: string): boolean {
  return (
    !path.startsWith('/') ||
    path.startsWith('//') ||
    path.includes('//') ||
    path.includes('\\') ||
    /(^|\/)\.\.(\/|$)/.test(path)
  );
}

/** Normalize a base URL/origin string to `url.origin` for same-origin checks. */
export function normalizeLoginRedirectOrigin(base: string): string {
  try {
    const origin = new URL(base).origin;
    // Some engines yield an empty origin instead of throwing for bare strings.
    return origin || String(base);
  } catch {
    return String(base);
  }
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
    return err.message || fallback;
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
    // Trim username only; password whitespace can be intentional.
    await opts.login(opts.form.username.trim(), opts.form.password, '', '', opts.rememberMe);
    return true;
  } catch (err) {
    opts.setError(formatLoginError(err, opts.loginFailedMessage));
    return false;
  }
}
