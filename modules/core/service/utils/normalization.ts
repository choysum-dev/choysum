// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from '../../utils/decimal';

/**
 * Return a trimmed string, or undefined when input is empty/null.
 */
export function normalizeOptionalString(value: unknown): string | undefined {
  const normalized = String(value || '').trim();
  return normalized || undefined;
}

/**
 * Normalize a possibly-mixed array into a deduplicated list of non-empty trimmed strings.
 */
export function normalizeStringArray(value: unknown): string[] {
  const arr = Array.isArray(value) ? value : [];
  return Array.from(new Set(arr.map(item => String(item || '').trim()).filter(Boolean)));
}

/**
 * Extract the identifier from either a plain string value or an object with
 * an `Id` property (e.g. a FK reference). Returns undefined for empty input.
 */
export function readRefId(value: unknown): string | undefined {
  if (!value) return undefined;
  if (typeof value === 'string') return normalizeOptionalString(value);
  if (typeof value === 'object') return normalizeOptionalString((value as any).Id);
  return undefined;
}

/**
 * Normalize a relation reference into a trimmed Id string.
 *
 * Accepts a plain string id, an object with an Id (or id) property, or null/undefined.
 * Returns null when the input cannot be resolved to a non-empty string.
 */
export function normalizeRefId(value: unknown): string | null {
  if (value == null) return null;
  const raw = typeof value === 'object' ? ((value as any).Id ?? (value as any).id ?? null) : value;
  const s = String(raw ?? '').trim();
  return s ? s : null;
}

/**
 * Normalize a mixed value or array into a deduplicated list of non-null Id strings.
 *
 * - Wraps a non-array, non-null input as a single-element array (singleton coercion).
 * - Extracts the Id from each element via {@link normalizeRefId}.
 * - Filters null results and deduplicates.
 */
export function normalizeRefIdList(value: unknown): string[] {
  if (value == null) return [];
  const arr = Array.isArray(value) ? value : [value];
  const set = new Set<string>();
  for (const it of arr) {
    const id = normalizeRefId(it);
    if (id) set.add(id);
  }
  return Array.from(set);
}

/**
 * Normalize an offset (non-negative finite integer, floor).
 */
export function normalizeOffset(raw: unknown): number {
  const value = Number(raw);
  if (!Number.isFinite(value) || value < 0) return 0;
  return Math.floor(value);
}

/**
 * Normalize a limit (positive finite integer, floor). Returns null for invalid/zero/negative.
 */
export function normalizeLimit(raw: unknown): number | null {
  const value = Number(raw);
  if (!Number.isFinite(value) || value <= 0) return null;
  return Math.floor(value);
}

/**
 * Normalize a field name list: trim, deduplicate, filter empty.
 */
export function normalizeFields(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const seen = new Set<string>();
  for (const item of raw) {
    const field = String(item || '').trim();
    if (!field) continue;
    seen.add(field);
  }
  return Array.from(seen);
}

/**
 * Normalize a mixed list into unique non-empty trimmed strings.
 *
 * Accepts an array of values (strings, numbers, etc.) and returns a
 * deduplicated array of non-empty strings. Non-array input is treated
 * as an empty list.
 */
export function uniqStrings(xs: unknown): string[] {
  return Array.from(new Set((Array.isArray(xs) ? xs : []).map(v => String(v ?? '').trim()).filter(Boolean)));
}

/**
 * Normalize supported require keys into canonical rpc:/ form.
 */
export function normalizeRpcRequireKey(key: string): string {
  const k = String(key || '').trim();
  if (!k) return '';
  if (k.startsWith('rpc:/')) return k;
  if (k.startsWith('service:/')) return `rpc:/${k.slice('service:/'.length)}`;
  return '';
}

/**
 * Convert an rpc:/model/method key into rpc:/model/* wildcard.
 */
export function rpcServiceWildcard(key: string): string {
  const k = String(key || '').trim();
  if (!k.startsWith('rpc:/')) return '';
  if (k.endsWith('/*')) return k;
  const i = k.lastIndexOf('/');
  if (i <= 'rpc:/'.length) return '';
  return `${k.slice(0, i)}/*`;
}

/**
 * Return a stable sorted copy of a string array.
 */
export function sortStrings(xs: string[]): string[] {
  return (xs || []).slice().sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

/**
 * Extract the identifier from either a plain string value or an object with
 * an `Id` (or `id`) property. Returns undefined for empty input.
 *
 * This is a relaxed variant of {@link readRefId} that also checks lowercase `id`.
 */
export function maybeRefId(value: unknown): string | undefined {
  if (!value) return undefined;
  if (typeof value === 'string') return normalizeOptionalString(value);
  if (typeof value === 'object')
    return normalizeOptionalString((value as Record<string, unknown>).Id) ?? normalizeOptionalString((value as Record<string, unknown>).id);
  return undefined;
}

function normalizeRefLikeIdString(raw: unknown): string {
  if (raw == null) return '';
  if (typeof raw === 'object') {
    return String((raw as Record<string, unknown>).Id ?? (raw as Record<string, unknown>).id ?? '').trim();
  }
  return String(raw ?? '').trim();
}

/**
 * Normalize an application or scope reference into its string id.
 *
 * Returns '' when the input cannot be resolved to a non-empty string (unlike
 * {@link normalizeRefId} which returns null for the same case).
 */
export function normalizeScopeRefId(raw: unknown): string {
  return normalizeRefLikeIdString(raw);
}

/**
 * Normalize a UI resource reference into its string id.
 */
export function normalizeUiResourceId(raw: unknown): string {
  return normalizeRefLikeIdString(raw);
}

/**
 * Parse a string or structured value into a normalized string array.
 *
 * Handles plain JSON arrays, objects with `value`/`values`/`items` keys,
 * numeric-indexed objects, and singleton string inputs.
 */
export function parseJsonStringArray(raw: unknown): string[] {
  const normalize = (xs: unknown): string[] => uniqStrings(xs);

  const tryObjectSnapshot = (value: unknown): string[] | null => {
    if (!value || typeof value !== 'object') return null;

    for (const key of ['value', 'values', 'items']) {
      try {
        const direct = (value as Record<string, unknown>)[key];
        if (Array.isArray(direct)) return normalize(direct);
      } catch {
        // fallthrough
      }
    }

    try {
      const numericKeys = Object.keys(value as Record<string, unknown>)
        .filter(key => /^\d+$/.test(key))
        .sort((a, b) => Number(a) - Number(b));
      if (numericKeys.length > 0) return normalize(numericKeys.map(key => (value as Record<string, unknown>)[key]));
    } catch {
      // fallthrough
    }
    return null;
  };

  if (Array.isArray(raw)) return normalize(raw);
  if (raw == null) return [];

  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    try {
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    } catch {
      // fallthrough
    }
    return normalize([s]);
  }

  const snapResult = tryObjectSnapshot(raw);
  if (snapResult) return snapResult;

  try {
    if (typeof (raw as Record<string, unknown>)?.toString === 'function') {
      const s = String((raw as Record<string, unknown>).toString() || '').trim();
      if (!s || s === '[object Object]') return [];
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    }
  } catch {
    // fallthrough
  }
  return [];
}

/**
 * Normalize a company id-like value to a trimmed string.
 */
export function normalizeScopeId(value: unknown): string {
  const id = maybeRefId(value);
  if (id) return String(id).trim();
  return String(value ?? '').trim();
}

/**
 * Return a stable unique copy of scope ids while preserving first-seen order.
 */
export function uniqScopeIds(ids: string[]): string[] {
  return Array.from(new Set((ids || []).map(v => normalizeScopeId(v)).filter(Boolean)));
}

/**
 * Normalize a dynamic preferences value into a plain JSON object.
 */
export function normalizePreferences(value: unknown): Record<string, unknown> {
  if (!value) return {};

  if (typeof value === 'string') {
    const s = value.trim();
    if (!s) return {};
    try {
      const parsed = JSON.parse(s);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
    } catch {
      return {};
    }
  }

  if (typeof value === 'object') {
    try {
      const snapshot = JSON.parse(JSON.stringify(value));
      if (snapshot && typeof snapshot === 'object' && !Array.isArray(snapshot)) return snapshot as Record<string, unknown>;
    } catch {
      // fallthrough
    }
    return value as Record<string, unknown>;
  }

  return {};
}

/**
 * Merge company scope preferences while preserving unrelated preference fields.
 */
export function buildScopePreferences(basePrefs: Record<string, unknown>, active: string, enabled: string[]): Record<string, unknown> {
  const base = basePrefs && typeof basePrefs === 'object' && !Array.isArray(basePrefs) ? basePrefs : {};
  return {
    ...base,
    activeCompanyId: active,
    enabledCompanyIds: enabled,
  };
}

/**
 * Coerce a value to a BigInt with lenient defaults.
 * Returns 0n for empty/falsy values (suitable for reading existing DB values).
 */
export function asBigInt(v: unknown): bigint {
  if (typeof v === 'bigint') return v;
  if (v && typeof v === 'object' && typeof (v as any).$bigint === 'string') return BigInt((v as any).$bigint);
  if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.trunc(v));
  const s = String((v as any) ?? '').trim();
  if (!s) return 0n;
  return BigInt(s);
}

/**
 * Extract a YYYY-MM-DD date-only string from a Date or string input.
 * Returns empty string for falsy/empty input.
 */
export function toDateOnlyString(input: unknown): string {
  if (input instanceof Date) return input.toISOString().slice(0, 10);
  const s = String((input as any) ?? '').trim();
  if (!s) return '';
  return s.length >= 10 ? s.slice(0, 10) : '';
}

export { buildPaddedNumberItems, formatPaddedNumber, resolvePaddedNumberFormat } from './format';

export type NormalizationErrorCode =
  | 'required'
  | 'string_too_long'
  | 'invalid_decimal'
  | 'non_positive_decimal'
  | 'number_not_allowed'
  | 'invalid_integer'
  | 'integer_too_small'
  | 'invalid_bigint'
  | 'invalid_date_format'
  | 'invalid_date_value'
  | 'invalid_enum_value';

/**
 * Resolve a model relation field to its canonical id.
 *
 * When the field value is an object with an `Id` property (e.g. a FK relation
 * record), returns that Id; otherwise returns the raw field value.  Returns
 * undefined when the input object is falsy or not an object.
 */
export function resolveModelRefId(obj: unknown, fieldName: string): unknown {
  if (!obj || typeof obj !== 'object') return undefined;
  const field = (obj as Record<string, unknown>)[fieldName];
  if (!field || typeof field !== 'object') return field;
  return (field as Record<string, unknown>).Id ?? field;
}

/**
 * Normalize a value against a fixed set of allowed string literals.
 *
 * - Returns {@link defaultValue} when value is undefined, null, or empty.
 * - Returns the matching allowed value when present.
 * - Throws {@link NormalizationError} with code `invalid_enum_value` otherwise.
 */
export function normalizeEnumValue<T extends string>(value: unknown, allowed: readonly T[], defaultValue: T): T {
  if (value === undefined || value === null || value === '') return defaultValue;
  const s = String(value).trim();
  if ((allowed as readonly string[]).includes(s)) return s as T;
  raiseNormalizationError('invalid_enum_value');
}

/**
 * Domain-agnostic error used by normalization utilities.
 */
export class NormalizationError extends Error {
  readonly code: NormalizationErrorCode;

  constructor(code: NormalizationErrorCode, message?: string) {
    super(message || code);
    this.name = 'NormalizationError';
    this.code = code;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

function raiseNormalizationError(code: NormalizationErrorCode): never {
  throw new NormalizationError(code);
}

/**
 * Parse arbitrary input into Decimal.
 */
export function parseDecimalInput(value: unknown, opts?: { allowNumber?: boolean }): Decimal {
  const allowNumber = opts?.allowNumber !== false;
  if (value === undefined || value === null || value === '') {
    raiseNormalizationError('required');
  }

  try {
    if (value instanceof Decimal) return value;

    if (typeof value === 'number') {
      if (!allowNumber) raiseNormalizationError('number_not_allowed');
      return new Decimal(value);
    }

    if (typeof value === 'object' && value && typeof (value as any).$bigdecimal === 'string') {
      return new Decimal(String((value as any).$bigdecimal));
    }

    if (typeof value === 'string') {
      return new Decimal(value);
    }
  } catch (err) {
    if (err instanceof NormalizationError) throw err;
    raiseNormalizationError('invalid_decimal');
  }

  raiseNormalizationError('invalid_decimal');
}

/**
 * Parse and validate a positive decimal (> 0).
 */
export function toPositiveDecimal(value: unknown): Decimal {
  const decimal = parseDecimalInput(value);
  if (!decimal.gt(0)) {
    raiseNormalizationError('non_positive_decimal');
  }
  return decimal;
}

/**
 * Normalize positive decimal as canonical string.
 */
export function normalizePositiveDecimalString(value: unknown): string {
  return toPositiveDecimal(value).toString();
}

/**
 * Normalize a required code string: trim, optional uppercase, and reject empty.
 */
export function normalizeCodeRequired(value: unknown, opts?: { uppercase?: boolean }): string {
  let code = String(value ?? '').trim();
  if (opts?.uppercase !== false) {
    code = code.toUpperCase();
  }
  if (!code) {
    raiseNormalizationError('required');
  }
  return code;
}

/**
 * Normalize an optional code string with null/undefined preservation.
 */
export function normalizeCodeOptional(value: unknown, opts?: { uppercase?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  let code = String(value ?? '').trim();
  if (opts?.uppercase !== false) {
    code = code.toUpperCase();
  }
  return code || null;
}

/**
 * Normalize a required name string.
 */
export function normalizeName(value: unknown): string {
  const name = String(value ?? '').trim();
  if (!name) {
    raiseNormalizationError('required');
  }
  return name;
}

/**
 * Resolve and require a non-empty reference id.
 */
export function requireRefId(value: unknown): string {
  const id = normalizeRefId(value);
  if (!id) {
    raiseNormalizationError('required');
  }
  return id;
}

/**
 * Normalize nullable string input: null/undefined/empty -> null.
 */
export function normalizeNullableString(value: unknown): string | null {
  if (value === undefined || value === null) return null;
  const normalized = String(value).trim();
  return normalized || null;
}

/**
 * Normalize an optional user-provided text value.
 *
 * - undefined/null => undefined
 * - empty/whitespace => required error
 * - length > maxLength => string_too_long error
 */
export function normalizeOptionalNonEmptyString(value: unknown, opts?: { maxLength?: number }): string | undefined {
  if (value === undefined || value === null) return undefined;

  const normalized = String(value).trim();
  if (!normalized) {
    raiseNormalizationError('required');
  }

  const maxLength = Number(opts?.maxLength);
  if (Number.isFinite(maxLength) && maxLength > 0 && normalized.length > maxLength) {
    raiseNormalizationError('string_too_long');
  }

  return normalized;
}

/**
 * Return whether a timestamp-like value is expired at nowMs.
 * Missing/invalid values are treated as expired.
 */
export function isExpiredAt(value: unknown, nowMs: number = Date.now()): boolean {
  if (!value) return true;
  const ms = new Date(value as any).getTime();
  if (!Number.isFinite(ms)) return true;
  return ms <= nowMs;
}

export type CurrencyRoundingSpec = {
  DecimalDigits?: unknown;
  Rounding?: unknown;
};

/**
 * Round an amount according to currency digits and optional rounding step.
 */
export function roundToCurrencyAmount(amount: Decimal, currency?: CurrencyRoundingSpec | null, overrideDigits?: number): Decimal {
  const digits = Number.isFinite(overrideDigits as any)
    ? Math.max(0, Math.floor(overrideDigits as any))
    : Math.max(0, Math.floor(Number(currency?.DecimalDigits) || 0));

  try {
    const step = currency?.Rounding;
    if (step != null) {
      const decimalStep = step instanceof Decimal ? step : new Decimal((step as any).$bigdecimal ?? step);
      if (decimalStep.gt(0)) {
        const q = amount.div(decimalStep);
        const qRounded = q.toDecimalPlaces(0, Decimal.ROUND_HALF_UP);
        const stepped = qRounded.times(decimalStep);
        return stepped.toDecimalPlaces(digits, Decimal.ROUND_HALF_UP);
      }
    }
  } catch {
    // Fall back to digits-only rounding when step parsing fails.
  }

  return amount.toDecimalPlaces(digits, Decimal.ROUND_HALF_UP);
}

/**
 * Normalize required text by trimming and rejecting empty values.
 */
export function normalizeRequiredText(value: unknown): string {
  const normalized = String(value ?? '').trim();
  if (!normalized) {
    raiseNormalizationError('required');
  }
  return normalized;
}

/**
 * Parse positive integer (>= 1).
 */
export function parsePositiveInt(value: unknown): number {
  const n = Number(value);
  if (!Number.isFinite(n) || Math.floor(n) !== n) {
    raiseNormalizationError('invalid_integer');
  }
  if (n < 1) {
    raiseNormalizationError('integer_too_small');
  }
  return n;
}

/**
 * Parse bigint-like input into bigint.
 */
export function parseBigInt(value: unknown): bigint {
  try {
    if (typeof value === 'bigint') return value;
    if (typeof value === 'number' && Number.isFinite(value)) return BigInt(Math.trunc(value));
    if (value && typeof value === 'object' && typeof (value as any).$bigint === 'string') return BigInt((value as any).$bigint);
    const text = String(value ?? '').trim();
    if (!text) {
      raiseNormalizationError('required');
    }
    return BigInt(text);
  } catch (err) {
    if (err instanceof NormalizationError) throw err;
    raiseNormalizationError('invalid_bigint');
  }
}

/**
 * Parse non-negative integer decimal digits.
 */
export function normalizeDecimalDigits(value: unknown): number {
  if (value === undefined || value === null || value === '') {
    raiseNormalizationError('required');
  }
  const n = Number(value);
  if (!Number.isFinite(n) || Math.floor(n) !== n || n < 0) {
    raiseNormalizationError('invalid_integer');
  }
  return n;
}

/**
 * Parse and validate YYYY-MM-DD date-only string.
 */
export function normalizeDateString(value: unknown): string {
  if (value === undefined || value === null || value === '') {
    raiseNormalizationError('required');
  }
  if (value instanceof Date) {
    raiseNormalizationError('invalid_date_format');
  }
  const raw = String(value).trim();
  if (!/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    raiseNormalizationError('invalid_date_format');
  }
  const date = new Date(`${raw}T00:00:00.000Z`);
  if (Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== raw) {
    raiseNormalizationError('invalid_date_value');
  }
  return raw;
}
