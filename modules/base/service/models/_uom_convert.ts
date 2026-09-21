// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import type { FieldSelection, Projected } from '@/core/service/api/selection';
import { parseDecimalInput, resolveModelRefId } from '@/core/service/utils/normalization';
import { mapNormalizationToBase, assertRefId } from './_normalizers';
import type UoM from './uom';
import type { UoMConvertParams, UoMConvertResult } from './uom';

const { _t } = createTranslate('base');

const UOM_FIELDS = ['Id', 'CategoryId', 'Factor', 'Rounding'] as const;
type UoMRow = Projected<UoM, typeof UOM_FIELDS>;

type UoMOps = {
  Browse: (id: string, fields?: FieldSelection<UoM>) => Promise<UoM | null | undefined>;
};

function notFound(message: string): never {
  throw new ChoysumError({ domain: 'base', code: 'NotFound', message }).withGrpcCode(GrpcCode.NotFound);
}

function invalid(message: string): never {
  throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
}

/**
 * Round `amount` to the UoM rounding step (half up). No-op when rounding is unset.
 */
function roundToUoM(amount: Decimal, rounding: unknown): Decimal {
  if (rounding == null || rounding === '') return amount;
  let step: Decimal;
  try {
    step = rounding instanceof Decimal ? rounding : new Decimal(String(rounding));
  } catch {
    return amount;
  }
  if (!step.isFinite() || step.lte(0)) return amount;
  return amount.div(step).toDecimalPlaces(0, Decimal.ROUND_HALF_UP).mul(step);
}

/**
 * Convert an amount between two UoMs in the same category.
 *
 * Factor is the multiplier from this unit to the category reference unit:
 * `amount_to = amount * from.Factor / to.Factor`, then rounded to the target UoM rounding.
 */
export async function convertUoM(UoMModel: UoMOps, params: UoMConvertParams | undefined | null): Promise<UoMConvertResult> {
  const fromUoMId = assertRefId(params?.FromUoMId, 'FromUoMId');
  const toUoMId = assertRefId(params?.ToUoMId, 'ToUoMId');
  const amount = mapNormalizationToBase(
    () => parseDecimalInput(params?.Amount, { allowNumber: false }),
    () => _t('Invalid Amount', { scope: 'service/models/_uom_convert' })
  );

  const load = async (id: string, label: string): Promise<UoMRow> => {
    const row = await UoMModel.Browse(id, UOM_FIELDS);
    if (!row) notFound(_t('%s not found', { scope: 'service/models/_uom_convert' }, label));
    return row as UoMRow;
  };

  const [from, to] = await Promise.all([load(fromUoMId, 'FromUoMId'), load(toUoMId, 'ToUoMId')]);
  const fromCategory = String(resolveModelRefId(from, 'CategoryId') ?? '').trim();
  const toCategory = String(resolveModelRefId(to, 'CategoryId') ?? '').trim();
  if (!fromCategory || !toCategory || fromCategory !== toCategory) {
    invalid(_t('UoMs must belong to the same category', { scope: 'service/models/_uom_convert' }));
  }

  const parseFactor = (value: unknown): Decimal => {
    try {
      const factor = value instanceof Decimal ? value : new Decimal(String(value));
      if (!factor.isFinite() || factor.lte(0)) {
        invalid(_t('UoM factor must be greater than 0', { scope: 'service/models/_uom_convert' }));
      }
      return factor;
    } catch (err) {
      if (err instanceof ChoysumError) throw err;
      invalid(_t('UoM factor must be greater than 0', { scope: 'service/models/_uom_convert' }));
    }
  };
  const fromFactor = parseFactor(from.Factor);
  const toFactor = parseFactor(to.Factor);

  const converted = amount.mul(fromFactor).div(toFactor);
  return { Amount: roundToUoM(converted, to.Rounding) };
}
