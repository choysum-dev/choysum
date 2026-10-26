// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { watch, type WatchSource } from 'vue';
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
    // Percent-encoded separators (`%2f`, `%5c`, multiply-encoded `%252f`) survive
    // URL parsing; re-check a fully decoded path so downstream re-decode cannot
    // turn them into `//host`.
    const decodedPath = decodeLoginRedirectPath(pathOnly);
    if (decodedPath == null) return '/';
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
 * Decode a pathname until stable (up to 3 passes). Returns null when encoding
 * is malformed so callers can fail closed.
 */
export function decodeLoginRedirectPath(path: string): string | null {
  let decoded = path;
  for (let pass = 0; pass < 3; pass++) {
    let next: string;
    try {
      next = decodeURIComponent(decoded);
    } catch {
      return null;
    }
    if (next === decoded) return decoded;
    decoded = next;
  }
  return decoded;
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
export function normalizeLoginRedirectOrigin(
  base: string,
  parseUrl: (input: string) => { origin: string } = (input) => new URL(input),
): string {
  try {
    const origin = parseUrl(base).origin;
    // Some engines yield an empty origin instead of throwing for bare strings.
    return origin || String(base);
  } catch {
    return String(base);
  }
}

export type LoginClientRule = {
  required?: boolean;
  message?: string;
  validator?: (rule: unknown, value: unknown, cb: (error?: Error) => void) => void;
};

/**
 * Username RuleItem: empty or whitespace-only is invalid (matches historical trim).
 * Use a live validator so locale switches refresh the message on the next check.
 */
export function loginUsernameRules(t: (msg: string) => string): LoginClientRule[] {
  return [
    {
      validator: (_rule: unknown, value: unknown, cb: (error?: Error) => void) => {
        if (!String(value ?? '').trim()) {
          cb(new Error(t('Enter username')));
          return;
        }
        cb();
      },
    },
  ];
}

/**
 * Password RuleItem: empty / null is invalid; whitespace-only is allowed.
 * Validator (not a frozen `message`) keeps the copy in sync with the active locale.
 */
export function loginPasswordRules(t: (msg: string) => string): LoginClientRule[] {
  return [
    {
      validator: (_rule: unknown, value: unknown, cb: (error?: Error) => void) => {
        if (value == null || value === '') {
          cb(new Error(t('Enter password')));
          return;
        }
        cb();
      },
    },
  ];
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
 * Field-level required checks belong on FormView Field rules; this only
 * skips in-flight submits and empty credentials as a last guard.
 */
export async function runLoginSubmit(opts: {
  loading: boolean;
  username: string;
  password: string;
  rememberMe: boolean;
  loginFailedMessage: string;
  login: (username: string, password: string, csrf: string, device: string, rememberMe: boolean) => Promise<void>;
  setError: (message: string) => void;
}): Promise<boolean> {
  if (opts.loading) return false;
  // Clear any previous server error before a new attempt so stale alerts
  // never sit next to freshly surfaced field errors.
  opts.setError('');
  if (!String(opts.username ?? '').trim() || !opts.password) return false;
  try {
    // Trim username only; password whitespace can be intentional.
    await opts.login(String(opts.username).trim(), opts.password, '', '', opts.rememberMe);
    return true;
  } catch (err) {
    opts.setError(formatLoginError(err, opts.loginFailedMessage));
    return false;
  }
}

/**
 * Run an auth page submit handler and always mark it handled.
 * Redirect/navigation stays outside the submit try so a rejected
 * router.replace is not reported as a credential failure.
 */
export async function runHandledAuthSubmit(opts: {
  submit: () => Promise<boolean>;
  fallbackMessage: string;
  setError: (message: string) => void;
  onSuccess: () => void;
}): Promise<{ handled: true; skipSuccessMessage: true }> {
  let ok = false;
  try {
    ok = await opts.submit();
  } catch (err) {
    opts.setError(err instanceof Error ? err.message : opts.fallbackMessage);
    return { handled: true, skipSuccessMessage: true };
  }
  if (ok) opts.onSuccess();
  return { handled: true, skipSuccessMessage: true };
}

/**
 * Clear a page-level auth error when credentials change after a failed attempt.
 * Bind the watch source to FormView draft fields (slot `formData`), not
 * createLocalFormStore getters — field edits write through form-root into draft.
 */
export function watchClearPageErrorOnCredentialChange(
  credentials: WatchSource,
  opts: { hasError: () => boolean; clearError: () => void },
) {
  return watch(credentials, () => {
    if (opts.hasError()) opts.clearError();
  });
}
