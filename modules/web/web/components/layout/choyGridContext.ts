// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

/** Parent ChoyGrid column count for ChoyCol span clamping. */
export const ChoyGridColsKey: InjectionKey<Ref<number>> = Symbol.for('choysum.choyGridCols');

/** Whether the parent ChoyGrid remaps spans by container width (default true). */
export const ChoyGridResponsiveKey: InjectionKey<Ref<boolean>> = Symbol.for(
  'choysum.choyGridResponsive',
);

/** Upper bound for generated grid tracks (guards against pathological input). */
export const MAX_CHOY_GRID_COLS = 24;

/** Wide-slot width in a 12-track 4-column (421) field grid. */
export const CHOY_GRID_WIDE_SLOT = 3;

/**
 * Container-band span map for CSS variables / @container rules.
 * Keys are container bands (not viewport Tailwind breakpoints).
 */
export type ChoyColSpanBands = { base: number; md: number; lg: number };

/**
 * ChoyCol `span` input: fixed lg target (scales under responsive), object bands,
 * or omitted (421 default slot when responsive).
 */
export type ChoyColSpanObject = {
  base?: number;
  sm?: number;
  md?: number;
  lg?: number;
};

export type ChoyColSpanInput = number | ChoyColSpanObject;

/** Reads a template-attribute number, treating blank / whitespace as unset. */
function toAttrNumber(value: unknown): number {
  return value == null || String(value).trim() === '' ? Number.NaN : Number(value);
}

/** Normalizes a parent grid track count to a positive integer (defaults to 12). */
export function normalizeChoyGridCols(cols?: number): number {
  const colsRaw = toAttrNumber(cols);
  return Number.isFinite(colsRaw)
    ? Math.min(MAX_CHOY_GRID_COLS, Math.max(1, Math.floor(colsRaw)))
    : 12;
}

/**
 * Clamps a numeric span against the parent track count.
 * Invalid / missing cols default to 12; invalid span defaults to `fallback` (full width).
 */
export function clampChoyColSpan(span: unknown, cols?: number, fallback?: number): number {
  const safeCols = normalizeChoyGridCols(cols);
  const spanRaw = toAttrNumber(span);
  const spanBase = Number.isFinite(spanRaw)
    ? Math.floor(spanRaw)
    : fallback == null
      ? safeCols
      : fallback;
  return Math.min(Math.max(spanBase, 1), safeCols);
}

/**
 * Clamps a ChoyCol span against the parent track count.
 * Invalid / missing cols default to 12; span defaults to full width.
 */
export function resolveChoyColSpan(span?: number | null, cols?: number): number {
  return clampChoyColSpan(span, cols);
}

/** True when `span` is a plain object band map (not a Number wrapper). */
export function isChoyColSpanObject(span: unknown): span is ChoyColSpanObject {
  return span != null && typeof span === 'object' && !Array.isArray(span);
}

/**
 * Scales an lg (wide) target span into 421 bands: wide / mid / narrow.
 * `slots = max(1, ceil(span / 3))` then wide=slots*3, mid=slots*6, narrow=full.
 */
export function scaleChoyColSpanFromLg(lgSpan: number, cols?: number): ChoyColSpanBands {
  const safeCols = normalizeChoyGridCols(cols);
  const lg = clampChoyColSpan(lgSpan, safeCols, CHOY_GRID_WIDE_SLOT);
  const slots = Math.max(1, Math.ceil(lg / CHOY_GRID_WIDE_SLOT));
  return {
    base: safeCols,
    md: Math.min(safeCols, slots * 6),
    lg: Math.min(safeCols, slots * CHOY_GRID_WIDE_SLOT),
  };
}

/** Default one-slot field bands under responsive 421. */
export function defaultChoyColSpanBands(cols?: number): ChoyColSpanBands {
  return scaleChoyColSpanFromLg(CHOY_GRID_WIDE_SLOT, cols);
}

/**
 * Resolves object span with mobile-first fallbacks into base/md/lg bands.
 * `sm` merges into the base band chain (`base ?? sm`).
 */
export function resolveChoyColSpanObjectBands(
  span: ChoyColSpanObject,
  cols?: number,
): ChoyColSpanBands {
  const safeCols = normalizeChoyGridCols(cols);
  const defaults = defaultChoyColSpanBands(safeCols);
  const baseRaw = span.base ?? span.sm;
  const baseBand =
    baseRaw == null ? defaults.base : clampChoyColSpan(baseRaw, safeCols, defaults.base);
  const mdBand =
    span.md == null ? baseBand : clampChoyColSpan(span.md, safeCols, baseBand);
  const lgBand = span.lg == null ? mdBand : clampChoyColSpan(span.lg, safeCols, mdBand);
  return { base: baseBand, md: mdBand, lg: lgBand };
}

/**
 * Total entry: omitted / number / object × responsive flag → CSS band spans.
 * When `responsive` is false, all bands equal the fixed numeric (or object lg??md??sm??base).
 */
export function resolveChoyColSpanBands(
  span: ChoyColSpanInput | null | undefined,
  cols?: number,
  responsive = true,
): ChoyColSpanBands {
  const safeCols = normalizeChoyGridCols(cols);

  if (!responsive) {
    let fixed: number;
    if (isChoyColSpanObject(span)) {
      const pick = span.lg ?? span.md ?? span.sm ?? span.base;
      fixed = clampChoyColSpan(pick, safeCols, safeCols);
    } else if (span == null || (typeof span === 'number' && !Number.isFinite(toAttrNumber(span)))) {
      // Match legacy resolveChoyColSpan(undefined): full width.
      fixed = safeCols;
    } else {
      fixed = clampChoyColSpan(span, safeCols, safeCols);
    }
    return { base: fixed, md: fixed, lg: fixed };
  }

  if (span == null) {
    return defaultChoyColSpanBands(safeCols);
  }

  if (isChoyColSpanObject(span)) {
    return resolveChoyColSpanObjectBands(span, safeCols);
  }

  // Blank attribute string → treat as omitted (421 default).
  if (typeof span === 'string' && String(span).trim() === '') {
    return defaultChoyColSpanBands(safeCols);
  }

  const lgTarget = clampChoyColSpan(span as number, safeCols, CHOY_GRID_WIDE_SLOT);
  return scaleChoyColSpanFromLg(lgTarget, safeCols);
}
