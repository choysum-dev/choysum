/*
 * SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
 * SPDX-License-Identifier: Apache-2.0
 */

import type Decimal from '@/core/utils/decimal';
import type { formatFixedDecimalString } from '@/web/web/stores/i18nStore';

export type DecimalNumberFormat = {
  thousandsSeparator?: string;
  decimalSeparator?: string;
  grouping?: number[];
};

/**
 * Readonly decimal display:
 * - fixed scale present → quantize + pad with toFixed(scale)
 * - no fixed scale → significant digits via toString() (no trailing-zero pad)
 */
export function formatODecimalDisplayText(
  d: Decimal,
  fixedScale: number | undefined,
  round: Decimal.Rounding,
  formatters?: {
    numberFormat?: DecimalNumberFormat | null;
    formatFixedDecimalString?: typeof formatFixedDecimalString;
  }
): string {
  let text: string;
  if (typeof fixedScale === 'number' && Number.isInteger(fixedScale) && fixedScale >= 0 && fixedScale <= 18) {
    text = d.toDecimalPlaces(fixedScale, round).toFixed(fixedScale);
  } else {
    text = d.toString();
  }
  const numberFormat = formatters?.numberFormat;
  const formatFixed = formatters?.formatFixedDecimalString;
  if (numberFormat && formatFixed) {
    try {
      return formatFixed(text, numberFormat);
    } catch {
      return text;
    }
  }
  return text;
}

/** Edit/validate scale: declared fixed scale, else DB soft max (NUMERIC scale 18). */
export function resolveODecimalEditScale(fixedScale: number | undefined): number {
  if (typeof fixedScale === 'number' && Number.isInteger(fixedScale) && fixedScale >= 0 && fixedScale <= 18) {
    return fixedScale;
  }
  return 18;
}
