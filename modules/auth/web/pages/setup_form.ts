// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import { createTranslate } from '@/web/web/i18n';
import type { CompleteInitialSetupInput } from '../stores/auth';
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
  _tSetupRule('Enter a valid email address'),
  _tSetupRule('Email is too long'),
  _tSetupRule('Enter organization name'),
  _tSetupRule('Organization name must be at most 100 characters'),
  _tSetupRule('Select a base currency'),
  _tSetupRule('Select a business timezone'),
];

/**
 * Translator for setup rule messages. Step components must build rules with this
 * (not their own scope) because the msgids are registered under `web/pages/Setup`.
 */
export const setupRuleTranslate = (msg: string): string => _tSetupRule(msg);

export type SetupClientRule = RegisterClientRule;

/** Wizard step ids, in display order. */
export type SetupStep = 'admin' | 'organization' | 'review';

export const SETUP_STEPS: readonly SetupStep[] = ['admin', 'organization', 'review'];

/** Local form fields shown on each editable step. */
export const SETUP_STEP_FIELDS: Record<Exclude<SetupStep, 'review'>, readonly string[]> = {
  admin: ['Username', 'Password', 'ConfirmPassword', 'Email'],
  organization: ['OrgName', 'CountryCode', 'CurrencyCode', 'Timezone'],
};

/** Server `metadata.Field` → local form field. */
const SERVER_FIELD_TO_LOCAL: Record<string, string> = {
  'Admin.Username': 'Username',
  'Admin.Password': 'Password',
  'Admin.Email': 'Email',
  'Organization.Name': 'OrgName',
  'Organization.CountryCode': 'CountryCode',
  'Organization.CurrencyCode': 'CurrencyCode',
  'Organization.Timezone': 'Timezone',
};

/** Local form field for a server error field, or '' when it has no input on the page. */
export function setupLocalFieldForServerField(field: string): string {
  return SERVER_FIELD_TO_LOCAL[String(field || '').trim()] ?? '';
}

/** Step that owns a server error field, or null when it maps to no step. */
export function setupStepForServerField(field: string): SetupStep | null {
  const local = setupLocalFieldForServerField(field);
  if (!local) return null;
  if (SETUP_STEP_FIELDS.admin.includes(local)) return 'admin';
  return 'organization';
}

function trimmed(value: unknown): string {
  return String(value ?? '').trim();
}

/**
 * Map wizard form values to the plain-text setup request.
 * Empty optional fields (Email, CountryCode) are omitted.
 */
export function buildCompleteInitialSetupReq(
  data: Record<string, unknown>,
  adminLanguageCode: string,
): CompleteInitialSetupInput {
  const email = trimmed(data.Email);
  const country = trimmed(data.CountryCode);
  return {
    Admin: {
      Username: trimmed(data.Username),
      Password: String(data.Password ?? ''),
      LanguageCode: trimmed(adminLanguageCode),
      ...(email ? { Email: email } : {}),
    },
    Organization: {
      Name: trimmed(data.OrgName),
      ...(country ? { CountryCode: country } : {}),
      CurrencyCode: trimmed(data.CurrencyCode),
      Timezone: trimmed(data.Timezone),
    },
  };
}

/** Login route after setup; `setupDone` adds `setup=done` so Login shows the completion notice. */
export function setupLoginFallbackTarget(
  query: Record<string, unknown> | undefined,
  opts?: { setupDone?: boolean },
): { name: 'login'; query: Record<string, string> } {
  const out: Record<string, string> = {};
  const src = query || {};
  for (const key of Object.keys(src)) {
    const value = src[key];
    if (value == null) continue;
    out[key] = Array.isArray(value) ? String(value[0] ?? '') : String(value);
  }
  if (opts?.setupDone) out.setup = 'done';
  return { name: 'login', query: out };
}

function ruleFromError(check: (value: unknown) => string): SetupClientRule {
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

/** Optional email: empty passes; otherwise matches the Register format and length cap. */
export function setupEmailError(value: unknown, t: (msg: string) => string): string {
  const raw = trimmed(value);
  if (!raw) return '';
  if (raw.length > 100) return t('Email is too long');
  if (!/^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/.test(raw)) {
    return t('Enter a valid email address');
  }
  return '';
}

export function setupOrgNameError(value: unknown, t: (msg: string) => string): string {
  const raw = trimmed(value);
  if (!raw) return t('Enter organization name');
  if (raw.length > 100) return t('Organization name must be at most 100 characters');
  return '';
}

function requiredSelection(message: string): (value: unknown) => string {
  return value => (trimmed(value) ? '' : message);
}

export function setupEmailRules(t: (msg: string) => string): SetupClientRule[] {
  return [ruleFromError(value => setupEmailError(value, t))];
}

export function setupOrgNameRules(t: (msg: string) => string): SetupClientRule[] {
  return [ruleFromError(value => setupOrgNameError(value, t))];
}

export function setupCurrencyRules(t: (msg: string) => string): SetupClientRule[] {
  return [ruleFromError(requiredSelection(t('Select a base currency')))];
}

export function setupTimezoneRules(t: (msg: string) => string): SetupClientRule[] {
  return [ruleFromError(requiredSelection(t('Select a business timezone')))];
}

/** Rule that reports a server-side field error until `getMessage` returns ''. */
export function setupServerErrorRule(getMessage: () => string): SetupClientRule {
  return ruleFromError(() => getMessage());
}

/** Append a server error rule when `message` is non-empty; the result changes identity with it. */
export function withSetupServerError(rules: SetupClientRule[], message: string | undefined): SetupClientRule[] {
  const text = String(message || '');
  return text ? [...rules, setupServerErrorRule(() => text)] : rules;
}

/** True when the review-step demo checkbox is checked. */
export function setupWantsDemo(data: Record<string, unknown> | null | undefined): boolean {
  return data?.WithDemo === true;
}

export type SetupLoadDemoResp = {
  Ok?: boolean;
  ErrorMessage?: string;
};

export type SetupLoadDemoRequester = () => Promise<SetupLoadDemoResp | null | undefined>;

/** Test-only RequestLoadDemo override; cleared between mounts. */
let testSetupLoadDemoRequester: SetupLoadDemoRequester | null = null;

/** Override the post-setup RequestLoadDemo call for FE mounts without meta store. */
export function setSetupLoadDemoRequesterForTests(fn: SetupLoadDemoRequester | null): void {
  testSetupLoadDemoRequester = fn;
}

/** Resolve RequestLoadDemo: test override first, else factory. */
export function resolveSetupLoadDemoRequester(fallback: SetupLoadDemoRequester): SetupLoadDemoRequester {
  return testSetupLoadDemoRequester ?? fallback;
}

/**
 * Best-effort load-demo after setup login. Never throws; callers toast via onFailure.
 * Soft-ok when the bridge reports no pending demo modules.
 */
export async function runSetupLoadDemo(opts: {
  requestLoadDemo: SetupLoadDemoRequester;
  failedMessage: string;
  onFailure: (message: string) => void;
}): Promise<boolean> {
  try {
    const resp = await opts.requestLoadDemo();
    if (resp && resp.Ok === false) {
      const message = String(resp.ErrorMessage || '').trim() || opts.failedMessage;
      opts.onFailure(message);
      return false;
    }
    return true;
  } catch (err) {
    console.error('Load demo after initial setup failed:', err);
    const message =
      err instanceof Error && String(err.message || '').trim()
        ? String(err.message).trim()
        : opts.failedMessage;
    opts.onFailure(message);
    return false;
  }
}

/**
 * Complete initial setup then sign in with the new admin.
 * Field rules run per step before this; empty credentials are a last guard.
 */
export async function runSetupSubmit(opts: {
  loading: boolean;
  req: CompleteInitialSetupInput;
  setupFailedMessage: string;
  completeInitialSetup: (req: CompleteInitialSetupInput) => Promise<unknown>;
  login: (username: string, password: string) => Promise<unknown>;
  setError: (message: string) => void;
  /**
   * Server error carrying `metadata.Field`; return true when the page shows it
   * next to the field (the header error is then left empty).
   */
  onFieldError?: (field: string, message: string) => boolean;
  /** Fired once CompleteInitialSetup succeeds, before Login. */
  onSetupSucceeded?: () => void;
  /** Fired when setup RPC succeeded but the follow-up Login failed. */
  onSetupCompleteLoginFailed?: () => void;
  /**
   * After CompleteInitialSetup fails (e.g. already complete elsewhere),
   * re-check Ready and leave /setup when appropriate.
   */
  onSetupFailed?: () => void | Promise<void>;
}): Promise<boolean> {
  if (opts.loading) return false;
  opts.setError('');
  const username = trimmed(opts.req.Admin.Username);
  const password = opts.req.Admin.Password;
  if (!username || !password) {
    return false;
  }
  try {
    await opts.completeInitialSetup(opts.req);
  } catch (err) {
    if (err instanceof ChoysumError) {
      const message = err.message || opts.setupFailedMessage;
      const field = String(err.metadata?.Field ?? '');
      const handled = !!field && !!opts.onFieldError?.(field, message);
      if (!handled) opts.setError(message);
    } else {
      console.error('Initial setup flow failed:', err);
      opts.setError(opts.setupFailedMessage);
    }
    try {
      await opts.onSetupFailed?.();
    } catch {
      // Keep the original setup error when recovery navigation fails.
    }
    return false;
  }
  opts.onSetupSucceeded?.();
  try {
    await opts.login(username, password);
    return true;
  } catch (err) {
    console.error('Login after initial setup failed:', err);
    opts.onSetupCompleteLoginFailed?.();
    return false;
  }
}
