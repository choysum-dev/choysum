// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Decimal, { isDecimal } from '@/core/utils/decimal';
import { formatCurrencyFromConfig, formatFixedDecimalString } from '@/web/web/stores/i18nStore';

export function getByPath(obj: unknown, path: string): unknown {
  return String(path)
    .split('.')
    .filter(Boolean)
    .reduce((a: any, k) => (a == null ? a : a[k]), obj as any);
}

export function resolveCurrencyValue(obj: unknown, currencyFieldName: string | undefined, bindingProp: string): unknown {
  const s = typeof currencyFieldName === 'string' ? currencyFieldName.trim() : '';
  if (!s || !obj) return undefined;
  const segs = String(bindingProp).split('.').filter(Boolean);
  if (segs.length) {
    segs[segs.length - 1] = s;
    const viaFullPath = getByPath(obj, segs.join('.'));
    if (viaFullPath != null) return viaFullPath;
  }
  return (obj as Record<string, unknown>)?.[s];
}

export function readCurrencyDigits(currency: unknown): number | undefined {
  if (currency == null) return undefined;
  if (typeof currency === 'object') {
    const record = currency as Record<string, unknown>;
    const n = Number(record.DecimalDigits ?? record.decimalDigits);
    if (Number.isInteger(n) && n >= 0 && n <= 18) return n;
  }
  return undefined;
}

export function readCurrencyCode(currency: unknown): string | undefined {
  if (!currency || typeof currency !== 'object') return undefined;
  const code = (currency as Record<string, unknown>).Code ?? (currency as Record<string, unknown>).code;
  return typeof code === 'string' && code.trim() ? code.trim() : undefined;
}

export function readCurrencySymbol(currency: unknown): string | undefined {
  if (!currency || typeof currency !== 'object') return undefined;
  const symbol = (currency as Record<string, unknown>).Symbol ?? (currency as Record<string, unknown>).symbol;
  return typeof symbol === 'string' && symbol.trim() ? symbol.trim() : undefined;
}

export function resolveMonetaryScaleFromRecord(
  obj: unknown,
  currencyFieldName: string | undefined,
  bindingProp: string,
  fallbackScale = 2
): number {
  try {
    const digits = readCurrencyDigits(resolveCurrencyValue(obj, currencyFieldName, bindingProp));
    if (digits != null) return digits;
  } catch {
    /* ignore */
  }
  return fallbackScale;
}

export function asMonetaryDecimal(v: unknown): Decimal | null {
  try {
    if (v == null || v === '') return null;
    if (isDecimal(v)) return v as Decimal;
    return new Decimal(v as any);
  } catch {
    return null;
  }
}

export type MonetaryDisplayFormatters = {
  formatCurrencyFromConfig: typeof formatCurrencyFromConfig;
  formatFixedDecimalString: typeof formatFixedDecimalString;
  numberFormat?: {
    thousandsSeparator?: string;
    decimalSeparator?: string;
    grouping?: number[];
    decimalDigits?: number;
  };
};

/** Format a monetary amount for display (symbol/code + quantized digits). */
export function formatMonetaryDisplayText(
  value: unknown,
  opts: {
    scale: number;
    roundingMode: Decimal.Rounding;
    currency: unknown;
    formatters?: MonetaryDisplayFormatters;
  }
): string {
  if (value == null || value === '') return '';
  const d = asMonetaryDecimal(value);
  if (!d) return '';
  const scale = opts.scale;
  try {
    const q = d.toDecimalPlaces(scale, opts.roundingMode);
    const code = readCurrencyCode(opts.currency);
    const symbol = readCurrencySymbol(opts.currency);
    const formatters = opts.formatters;
    const numberFormat = { ...(formatters?.numberFormat || {}), decimalDigits: scale };
    if (code && formatters?.formatCurrencyFromConfig) {
      return formatters.formatCurrencyFromConfig(Number(q.toString()), numberFormat, code);
    }
    const fixed = q.toFixed(scale);
    const formatted = formatters?.formatFixedDecimalString
      ? formatters.formatFixedDecimalString(fixed, numberFormat)
      : fixed;
    return symbol ? `${symbol} ${formatted}` : formatted;
  } catch {
    return d.toString();
  }
}

export function leafOfPath(path: string): string {
  const segs = String(path).split('.').filter(Boolean);
  return segs.length ? segs[segs.length - 1]! : String(path);
}

export type AggPropLike = string | { agg?: string; alias?: string };

/** Resolve aggregate/metric display fallback for decimal-like field widgets. */
export function resolveAggregateDisplayValue(
  raw: unknown,
  obj: unknown,
  opts: { bindingProp: string; agg?: AggPropLike }
): unknown {
  if (raw != null && raw !== '') return raw;
  if (!obj || typeof obj !== 'object') return raw;

  const path = String(opts.bindingProp || '');
  const leaf = leafOfPath(path);
  const row = obj as Record<string, any>;
  const metrics = row.metrics && typeof row.metrics === 'object' ? (row.metrics as Record<string, any>) : null;

  const agg = opts.agg;
  const suffixOf = (fn: string) => (fn === 'count' ? '__count' : `__${fn}`);

  const candidates: string[] = [];
  if (agg) {
    if (typeof agg === 'string') {
      const suf = suffixOf(agg);
      if (suf === '__count') candidates.push('__count');
      else candidates.push(`${path}${suf}`, `${leaf}${suf}`);
    } else if (agg.agg) {
      if (agg.alias && String(agg.alias).trim()) candidates.push(String(agg.alias).trim());
      const suf = suffixOf(agg.agg);
      if (suf === '__count') candidates.push('__count');
      else candidates.push(`${path}${suf}`, `${leaf}${suf}`);
    }
  }

  for (const k of candidates) {
    if (!k) continue;
    if (metrics && metrics[k] != null) return metrics[k];
    if (row[k] != null) return row[k];
  }

  const AGG_SUFFIX = ['__sum', '__avg', '__min', '__max', '__count', '__count_distinct'];
  const PURE_AGG_SUFFIX = ['__sum', '__avg', '__min', '__max', '__count_distinct'];
  if (metrics) {
    const mkeys = Object.keys(metrics);
    const strictByPath = mkeys.filter(k => k.startsWith(path + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
    if (strictByPath.length === 1 && metrics[strictByPath[0]] != null) return metrics[strictByPath[0]];
    const strictByLeaf = mkeys.filter(k => k.startsWith(leaf + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
    if (strictByLeaf.length === 1 && metrics[strictByLeaf[0]] != null) return metrics[strictByLeaf[0]];
    const byPath = mkeys.filter(k => k === '__count' || k.startsWith(path + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
    const byLeaf = mkeys.filter(k => k === '__count' || k.startsWith(leaf + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
    const uniq = byPath.length === 1 ? byPath[0] : byLeaf.length === 1 ? byLeaf[0] : null;
    if (uniq && metrics[uniq] != null) return metrics[uniq];
  }

  const tkeys = Object.keys(row);
  const tStrictByPath = tkeys.filter(k => k.startsWith(path + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
  if (tStrictByPath.length === 1 && row[tStrictByPath[0]] != null) return row[tStrictByPath[0]];
  const tStrictByLeaf = tkeys.filter(k => k.startsWith(leaf + '__')).filter(k => PURE_AGG_SUFFIX.some(s => k.endsWith(s)));
  if (tStrictByLeaf.length === 1 && row[tStrictByLeaf[0]] != null) return row[tStrictByLeaf[0]];
  const tByPath = tkeys.filter(k => k === '__count' || k.startsWith(path + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
  const tByLeaf = tkeys.filter(k => k === '__count' || k.startsWith(leaf + '__')).filter(k => AGG_SUFFIX.some(s => k.endsWith(s)));
  const tUniq = tByPath.length === 1 ? tByPath[0] : tByLeaf.length === 1 ? tByLeaf[0] : null;
  if (tUniq && row[tUniq] != null) return row[tUniq];

  return raw;
}

export function isIntermediateMonetaryInput(s: string | null): boolean {
  if (s == null) return false;
  if (s === '-' || s === '.' || s === '-.' || /^-?\d+\.$/.test(s)) return true;
  return false;
}

export function parseStrictMonetary(
  s: string,
  scale: number,
  opts: { precision: number; min?: Decimal | null; max?: Decimal | null }
): Decimal | null {
  const t = s.trim();
  if (!/^[-]?\d*(\.\d*)?$/.test(t)) return null;
  try {
    const d = new Decimal(t);
    if (!d.isFinite()) return null;
    const places = d.decimalPlaces();
    if (places != null && places > scale) return null;
    const digits = d.abs().sd(true);
    if (digits != null && digits > opts.precision) return null;
    if (opts.min && d.lessThan(opts.min)) return null;
    if (opts.max && d.greaterThan(opts.max)) return null;
    return d;
  } catch {
    return null;
  }
}

export function clampMonetaryValue(
  d: Decimal,
  scale: number,
  roundingMode: Decimal.Rounding,
  opts: { precision: number; min?: Decimal | null; max?: Decimal | null }
): Decimal {
  let v = d.toDecimalPlaces(scale, roundingMode);
  const digits = v.abs().sd(true) ?? 0;
  if (digits > opts.precision) {
    const shift = digits - opts.precision;
    v = v.div(new Decimal(10).pow(shift)).toDecimalPlaces(scale, roundingMode);
  }
  if (opts.min) v = Decimal.max(v, opts.min);
  if (opts.max) v = Decimal.min(v, opts.max);
  return v;
}

export function quantizeMonetaryForCompare(x: unknown, scale: number, roundingMode: Decimal.Rounding): Decimal | null {
  const d = asMonetaryDecimal(x);
  if (!d) return null;
  return d.toDecimalPlaces(scale, roundingMode);
}

export function validateMonetaryValue(
  value: unknown,
  scale: number,
  opts: { precision: number; min?: Decimal | null; max?: Decimal | null },
  t: (msg: string, ...args: unknown[]) => string
): string | null {
  if (value == null || value === '') return null;
  const d = asMonetaryDecimal(value);
  if (!d || !d.isFinite()) return t('Must be a valid number');
  if ((d.decimalPlaces() ?? 0) > scale) return t('Decimal places must not exceed %s', scale);
  const digits = d.abs().sd(true) ?? 0;
  if (digits > opts.precision) return t('Total digits must not exceed %s', opts.precision);
  if (opts.min && d.lessThan(opts.min)) return t('Must not be less than %s', opts.min.toString());
  if (opts.max && d.greaterThan(opts.max)) return t('Must not be greater than %s', opts.max.toString());
  return null;
}

export function currencyFieldPaths(bindingProp: string, currencyFieldName: string | undefined): string[] {
  const s = typeof currencyFieldName === 'string' ? currencyFieldName.trim() : '';
  if (!s) return [];
  const segs = String(bindingProp).split('.').filter(Boolean);
  if (!segs.length) return [];
  segs[segs.length - 1] = s;
  const base = segs.join('.');
  return [base, `${base}.DecimalDigits`, `${base}.Symbol`, `${base}.Code`];
}
