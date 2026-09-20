// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal from 'decimal.js';
import { asObjectRecord, hasOwnKey, isObjectRecord, isStringNumberEnvelope } from './object';
import type { ObjectRecord } from './types';

export type DecimalRound =
  | 'ROUND_UP'
  | 'ROUND_DOWN'
  | 'ROUND_CEIL'
  | 'ROUND_FLOOR'
  | 'ROUND_HALF_UP'
  | 'ROUND_HALF_DOWN'
  | 'ROUND_HALF_EVEN'
  | 'ROUND_HALF_CEIL'
  | 'ROUND_HALF_FLOOR';

type DeserializeEnvelopeLike = { $bigdecimal: unknown } | { $bigint: unknown };
type SerializeTransformOutput<T> = T extends null | undefined
  ? T
  : T extends Array<infer U>
    ? Array<SerializeTransformOutput<U>>
    : T extends object
      ? { [K in keyof T]: SerializeTransformOutput<T[K]> }
      : T;
type DeserializeTransformOutput<T> = T extends null | undefined
  ? T
  : T extends Array<infer U>
    ? Array<DeserializeTransformOutput<U>>
    : T extends DeserializeEnvelopeLike
      ? unknown
      : T extends object
        ? { [K in keyof T]: DeserializeTransformOutput<T[K]> }
        : T;

export function toDecimalRounding(round?: DecimalRound | number | null | undefined): Decimal.Rounding {
  if (typeof round === 'number') {
    return round as Decimal.Rounding;
  }
  switch (round) {
    case 'ROUND_UP':
      return Decimal.ROUND_UP;
    case 'ROUND_DOWN':
      return Decimal.ROUND_DOWN;
    case 'ROUND_CEIL':
      return Decimal.ROUND_CEIL;
    case 'ROUND_FLOOR':
      return Decimal.ROUND_FLOOR;
    case 'ROUND_HALF_UP':
      return Decimal.ROUND_HALF_UP;
    case 'ROUND_HALF_DOWN':
      return Decimal.ROUND_HALF_DOWN;
    case 'ROUND_HALF_EVEN':
      return Decimal.ROUND_HALF_EVEN;
    case 'ROUND_HALF_CEIL':
      return Decimal.ROUND_HALF_CEIL;
    case 'ROUND_HALF_FLOOR':
      return Decimal.ROUND_HALF_FLOOR;
    default:
      return Decimal.ROUND_HALF_UP;
  }
}

// Global singleton config: high precision for intermediate arithmetic, bankers rounding, and no exponential notation.
Decimal.set({
  precision: 50,
  rounding: Decimal.ROUND_HALF_EVEN,
  toExpNeg: -1e9,
  toExpPos: 1e9,
});

/**
 * Checks whether a value is a Decimal instance.
 */
export function isDecimal(v: unknown): v is Decimal {
  const record = asObjectRecord(v);
  if (!record) return false;

  // 1) Preferred: use the library guard when available, but tolerate missing APIs across versions or bundles.
  try {
    const decimalStatic = Decimal as unknown as { isDecimal?: (value: unknown) => boolean };
    if (typeof decimalStatic.isDecimal === 'function' && decimalStatic.isDecimal(v)) {
      return true;
    }
  } catch {
    /* ignore */
  }

  // 2) Fallback: use duck typing across copies or realms.
  // - Match the constructor name and key instance methods so leaked {s,e,d} shapes are not misclassified.
  const ctor = asObjectRecord(record.constructor);
  const nameOk = !!ctor && typeof ctor.name === 'string' && ctor.name.startsWith('Decimal');
  const methodsOk =
    typeof record.toString === 'function' && typeof record.toNumber === 'function' && typeof record.plus === 'function' && typeof record.times === 'function';

  return !!nameOk && !!methodsOk;
}

/**
 * NEW: Broader "Decimal-like" detection based on duck typing.
 */
export function isDecimalLike(v: unknown): boolean {
  const record = asObjectRecord(v);
  if (!record) return false;
  try {
    const hasToString = typeof record.toString === 'function';
    const hasDecimalPlaces = typeof record.decimalPlaces === 'function';
    const hasSd = typeof record.sd === 'function' || typeof record.sd === 'number';
    return (hasToString && hasDecimalPlaces) || hasSd;
  } catch {
    return false;
  }
}

/**
 * Detects whether a value is a leaked Decimal internal structure.
 * Decimal instances may serialize to a {s, e, d} shape in some QuickJS paths.
 */
export function isDecimalLeak(v: unknown): v is { s: number; e: number; d: number[] } {
  const record = asObjectRecord(v);
  if (!record) return false;
  const keys = Object.keys(record);
  if (keys.length !== 3) return false;
  if (!keys.includes('s') || !keys.includes('e') || !keys.includes('d')) return false;
  const s = record.s;
  const e = record.e;
  const d = record.d;
  return typeof s === 'number' && typeof e === 'number' && Array.isArray(d) && d.every(n => typeof n === 'number');
}

/**
 * Serializes Decimal and BigInt values into {$bigdecimal/$bigint} envelopes.
 * Used for RPC transport and database persistence.
 */
export function serialize<T = unknown>(val: T): SerializeTransformOutput<T> {
  const seen = new WeakMap<object, unknown>();
  return _serialize(val, seen) as SerializeTransformOutput<T>;
}

function _serialize(val: unknown, seen: WeakMap<object, unknown>): unknown {
  if (val == null) return val;
  const type = typeof val;

  // Handle BigInt.
  if (type === 'bigint') {
    return { $bigint: String(val) };
  }

  // Normalize {$bigdecimal} envelopes in place.
  if (isBigdecimalEnvelope(val)) {
    return { $bigdecimal: String(val.$bigdecimal) };
  }

  // Handle Decimal instances.
  if (isDecimal(val)) {
    try {
      return { $bigdecimal: val.toString() };
    } catch (err) {
      console.warn('Failed to serialize Decimal:', err);
    }
  }

  // Recurse into arrays.
  if (Array.isArray(val)) {
    return val.map(item => _serialize(item, seen));
  }

  // Preserve non-plain objects to avoid losing Date/Map/Set semantics.
  if (type === 'object' && !isObjectRecord(val)) {
    // Best-effort stringify leaked Decimal structures.
    if (isDecimalLeak(val)) {
      try {
        const reconstructed = new Decimal(0);
        const reconstructedLike = reconstructed as unknown as { s: number; e: number; d: number[] };
        reconstructedLike.s = val.s;
        reconstructedLike.e = val.e;
        reconstructedLike.d = val.d.slice();
        return { $bigdecimal: reconstructed.toString() };
      } catch (err) {
        console.warn('Failed to reconstruct leaked Decimal:', err);
      }
    }
    return val;
  }

  // Recurse into plain objects with cycle protection.
  if (type === 'object') {
    const record = asObjectRecord(val);
    if (!record) return val;
    if (seen.has(record)) return seen.get(record);
    const out: ObjectRecord = {};
    seen.set(record, out);
    for (const key of Object.keys(record)) {
      out[key] = _serialize(record[key], seen);
    }
    return out;
  }

  return val;
}

/**
 * Deserializes {$bigdecimal/$bigint} envelopes back into Decimal and BigInt values.
 * Used for RPC inputs and database reads.
 */
export function deserialize<T = unknown>(val: T): DeserializeTransformOutput<T> {
  return _deserialize(val) as DeserializeTransformOutput<T>;
}

function _deserialize(val: unknown): unknown {
  if (val == null) return val;

  // Recurse into arrays.
  if (Array.isArray(val)) {
    return val.map(_deserialize);
  }

  // Only process plain objects; return other objects as-is.
  if (typeof val === 'object' && isObjectRecord(val)) {
    const record = asObjectRecord(val);
    if (!record) return val;
    const keys = Object.keys(record);

    // Deserialize {$bigdecimal}.
    if (keys.length === 1 && hasOwnKey(record, '$bigdecimal')) {
      const bigdecimalVal = record.$bigdecimal;
      if (typeof bigdecimalVal === 'string' || typeof bigdecimalVal === 'number') {
        const str = String(bigdecimalVal);
        try {
          return new Decimal(str);
        } catch (err) {
          console.warn('Failed to deserialize $bigdecimal:', err);
          return str; // Fall back to the raw string.
        }
      }
    }

    // Deserialize {$bigint}.
    if (keys.length === 1 && hasOwnKey(record, '$bigint')) {
      const bigintVal = record.$bigint;
      if (typeof bigintVal === 'string' || typeof bigintVal === 'number') {
        try {
          return BigInt(String(bigintVal));
        } catch (err) {
          console.warn('Failed to deserialize $bigint:', err);
          return String(bigintVal); // Fall back to the raw string.
        }
      }
    }

    // Recurse into ordinary objects.
    const result: ObjectRecord = {};
    for (const key of keys) {
      result[key] = _deserialize(record[key]);
    }
    return result;
  }

  return val;
}

/**
 * Compares Decimal-like values in diff-oriented paths.
 * - Uses numeric equivalence, so 1 and 1.0 are treated as equal.
 */
export function decimalEqual(a: unknown, b: unknown): boolean {
  const toDec = (x: unknown): Decimal | undefined => {
    try {
      if (isDecimal(x)) return x;
      if (isBigdecimalEnvelope(x)) return new Decimal(String(x.$bigdecimal));
      if (typeof x === 'number' || typeof x === 'string') return new Decimal(x);
    } catch {
      /* ignore */
    }
    return undefined;
  };

  const da = toDec(a);
  const db = toDec(b);
  if (!da || !db) return false;
  try {
    return da.eq(db);
  } catch {
    return false;
  }
}

// Keep this helper for string comparisons if needed, but it is no longer used by decimalEqual.
function toDecimalComparable(v: unknown): string | undefined {
  if (isDecimal(v)) return v.toString();
  if (isBigdecimalEnvelope(v)) return String(v.$bigdecimal);
  if (typeof v === 'number' || typeof v === 'string') return String(v);
  return undefined;
}

/**
 * Checks whether a value is a Bigdecimal envelope object.
 * Envelope shape: {$bigdecimal: string | number}.
 */
export function isBigdecimalEnvelope(v: unknown): v is { $bigdecimal: string | number } {
  return isStringNumberEnvelope(v, '$bigdecimal');
}

export function asBigdecimal(v: unknown): { $bigdecimal: string } | unknown {
  if (v == null) return v;
  if (isBigdecimalEnvelope(v)) return { $bigdecimal: String(v.$bigdecimal) };
  if (isDecimal(v)) {
    try {
      return { $bigdecimal: v.toString() };
    } catch {
      return v;
    }
  }
  if (typeof v === 'number' || typeof v === 'string') {
    return { $bigdecimal: String(v) };
  }
  return v;
}

export function toDecimalString(v: unknown): string | undefined {
  try {
    if (isDecimal(v)) return v.toString();
    if (isBigdecimalEnvelope(v)) return new Decimal(String(v.$bigdecimal)).toString();
    if (typeof v === 'number' || typeof v === 'string') return new Decimal(v).toString();
  } catch {
    // Invalid numeric strings and similar inputs return undefined.
  }
  return undefined;
}

/**
 * Reads scale and round from field metadata for field-level rounding.
 * - fm may be FieldMetadata or a lightweight object to avoid cyclic type dependencies.
 * - ROUND_HALF_UP remains the default because it better matches ERP expectations.
 */
export function getScaleAndRound(fm: { column?: unknown; select?: unknown } | undefined): { scale?: number; round: Decimal.Rounding } {
  const col = asObjectRecord(fm?.column) ?? asObjectRecord(fm?.select);
  const scale = typeof col?.scale === 'number' ? col.scale : undefined;
  // Accept either string or number round values.
  const round = toDecimalRounding(col?.round as DecimalRound | number | undefined);
  return { scale, round };
}

/**
 * Computes the number of fractional decimal places.
 */
function decimalPlacesOf(d: Decimal): number {
  try {
    const n = d.decimalPlaces();
    return typeof n === 'number' && n >= 0 ? n : 0;
  } catch {
    return 0;
  }
}

/**
 * Computes the number of integer digits (>= 1) for NUMERIC(precision, scale) validation.
 * - 0.xxx counts as one integer digit to match database semantics.
 */
function integerDigitsOf(d: Decimal): number {
  const abs = d.abs();
  if (abs.lessThan(1)) return 1;
  try {
    // toString runs without exponential notation, so it is safe here.
    const s = abs.floor().toString();
    // s is a plain integer string.
    return s.startsWith('-') ? s.length - 1 : s.length;
  } catch {
    // Fallback: approximate with logarithms, which may be off by one digit.
    const log10 = abs.log(10);
    const k = log10.isFinite() ? Math.floor(log10.toNumber()) + 1 : 1;
    return k > 0 ? k : 1;
  }
}

/**
 * Normalizes a Decimal value with field metadata so scale and rounding stay consistent.
 * - Inputs may be Decimal, {$bigdecimal}, string, number, or any value with toString().
 * - Returns the normalized Decimal, or undefined when parsing fails, the value is non-finite, or DB limits are exceeded.
 * - Idempotent: calling it again on an already normalized value does not change the number.
 *
 * Unified NUMERIC(38,18) soft validation:
 * - Fractional digits <= 18
 * - Integer digits <= 20 (38 - 18)
 * - If the field declares a smaller precision, total significant digits (sd) must stay within that precision
 */
export function normalizeDecimalByMeta(fm: { column?: unknown; select?: unknown } | undefined, value: unknown): Decimal | undefined {
  if (value == null) return undefined;

  let d: Decimal | undefined;
  try {
    if (isDecimal(value)) {
      // Rebind to the current global Decimal so cross-realm or cloned values stay consistent.
      d = new Decimal(value.toString());
    } else if (isBigdecimalEnvelope(value)) {
      d = new Decimal(String(value.$bigdecimal));
    } else if (typeof value === 'string' || typeof value === 'number') {
      d = new Decimal(value);
    } else {
      const valueRecord = asObjectRecord(value);
      if (valueRecord && typeof valueRecord.toString === 'function') {
        d = new Decimal(String(valueRecord.toString()));
      }
    }
  } catch {
    return undefined;
  }

  if (!d || !d.isFinite()) return undefined;

  // 1) Apply field-level quantization when scale is declared.
  const { scale, round } = getScaleAndRound(fm);
  if (typeof scale === 'number') {
    d = d.toDecimalPlaces(scale, round);
  }

  // 2) Apply unified DB soft limits for NUMERIC(38,18).
  const DB_PRECISION = 38;
  const DB_SCALE = 18;
  const DB_INT_DIGITS = DB_PRECISION - DB_SCALE; // 20

  // 2.1 Fractional digits must not exceed 18.
  const dp = decimalPlacesOf(d);
  if (dp > DB_SCALE) {
    return undefined;
  }

  // 2.2 Integer digits must not exceed 20.
  const intDigits = integerDigitsOf(d);
  if (intDigits > DB_INT_DIGITS) {
    return undefined;
  }

  // 2.3 If the field declares a smaller business precision, validate total significant digits (sd).
  const col = asObjectRecord(fm?.column) ?? asObjectRecord(fm?.select);
  const metaPrecision = typeof col?.precision === 'number' ? col.precision : undefined;
  if (typeof metaPrecision === 'number') {
    try {
      const sd = d.sd(true); // sd(true) includes trailing zeros.
      if (typeof sd === 'number' && sd > metaPrecision) {
        return undefined;
      }
    } catch {
      // Ignore sd failures and fail closed.
      return undefined;
    }
  }

  return d;
}

(() => {
  // Runtime environments auto-inject these helpers into $choysum.utils.
  if (typeof globalThis === 'undefined') return;
  const globalRecord = asObjectRecord(globalThis);
  const choysum = asObjectRecord(globalRecord?.$choysum);
  if (!choysum) return;

  const utils = asObjectRecord(choysum.utils) || {};
  utils.isDecimal = isDecimal;
  utils.isDecimalLike = isDecimalLike;
  utils.isDecimalLeak = isDecimalLeak;
  utils.serialize = serialize;
  utils.deserialize = deserialize;
  utils.decimalEqual = decimalEqual;
  utils.asBigdecimal = asBigdecimal;
  utils.isBigdecimalEnvelope = isBigdecimalEnvelope;
  utils.toDecimalString = toDecimalString;

  // Expose normalization helpers for runtime debugging and reuse.
  utils.getScaleAndRound = getScaleAndRound;
  utils.normalizeDecimalByMeta = normalizeDecimalByMeta;

  choysum.utils = utils;
})();

export default Decimal;
