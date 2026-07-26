// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Model } from '../decorator/model';
import { getCurrencyFieldName, isDecimalLikeField, isDecimalLikeFieldType } from './decimal_like';
import {
  BASE_CURRENCY_FULL_NAME,
  isCurrencyRelationField,
  normalizeCurrencyRefId,
  readDecimalDigitsFromCurrencyValue,
  validateModelMonetaryCurrencyFields,
} from './monetary_currency';
import {
  monetaryCurrencyRequiredError,
  resolveMonetaryScaleForWrite,
  resolveMonetaryScaleFromPayload,
  resolveMonetaryScaleFromRow,
} from './monetary_scale_resolve';
import { buildHiddenScaleAlias } from '../repository/hidden_scale_alias';

test('decimal_like helpers recognize decimal and monetary', () => {
  expect(isDecimalLikeFieldType('decimal')).toBe(true);
  expect(isDecimalLikeFieldType('monetary')).toBe(true);
  expect(isDecimalLikeFieldType('varchar')).toBe(false);
  expect(isDecimalLikeFieldType(undefined)).toBe(false);
  expect(isDecimalLikeField({ type: 'monetary' } as any)).toBe(true);
  expect(isDecimalLikeField(null)).toBe(false);
  expect(getCurrencyFieldName(undefined)).toBeUndefined();
  expect(getCurrencyFieldName('x')).toBeUndefined();
  expect(getCurrencyFieldName({ currencyField: 1 })).toBeUndefined();
  expect(getCurrencyFieldName({ currencyField: '  ' })).toBeUndefined();
  expect(getCurrencyFieldName({ currencyField: ' CurrencyId ' })).toBe('CurrencyId');
});

test('normalizeCurrencyRefId and readDecimalDigitsFromCurrencyValue cover shapes', () => {
  expect(normalizeCurrencyRefId(null)).toBeUndefined();
  expect(normalizeCurrencyRefId('  ')).toBeUndefined();
  expect(normalizeCurrencyRefId(' CUR-1 ')).toBe('CUR-1');
  expect(normalizeCurrencyRefId({ Id: 'A' })).toBe('A');
  expect(normalizeCurrencyRefId({ id: 'b' })).toBe('b');
  expect(normalizeCurrencyRefId({ Id: 1 })).toBeUndefined();
  expect(normalizeCurrencyRefId(12)).toBeUndefined();

  expect(readDecimalDigitsFromCurrencyValue(null)).toBeUndefined();
  expect(readDecimalDigitsFromCurrencyValue({ DecimalDigits: 2 })).toBe(2);
  expect(readDecimalDigitsFromCurrencyValue({ decimalDigits: 0 })).toBe(0);
  expect(readDecimalDigitsFromCurrencyValue({ DecimalDigits: 19 })).toBeUndefined();
  expect(readDecimalDigitsFromCurrencyValue({ DecimalDigits: 1.5 })).toBeUndefined();
});

test('isCurrencyRelationField accepts Ref and local ManyToOne to base.Currency', () => {
  expect(isCurrencyRelationField(undefined)).toBe(false);
  expect(isCurrencyRelationField({ type: 'varchar' } as any)).toBe(false);
  expect(
    isCurrencyRelationField({
      type: 'ManyToOneRef',
      relation: { targetModel: 'base.Currency' },
    } as any)
  ).toBe(true);
  expect(
    isCurrencyRelationField({
      type: 'ManyToOneRef',
      relation: { targetModel: '  ' },
    } as any)
  ).toBe(false);
  expect(
    isCurrencyRelationField({
      type: 'ManyToOneRef',
      relation: { targetModel: 'base.Country' },
    } as any)
  ).toBe(false);
  expect(isCurrencyRelationField({ type: 'ManyToOneRef', relation: undefined } as any)).toBe(false);

  @Model('Currency', { application: 'base' })
  class LocalCurrency extends BaseModel {}

  expect(
    isCurrencyRelationField({
      type: 'ManyToOne',
      relation: { targetModel: () => LocalCurrency },
    } as any)
  ).toBe(true);

  expect(
    isCurrencyRelationField({
      type: 'ManyToOne',
      relation: { targetModel: () => null as any },
    } as any)
  ).toBe(false);

  expect(
    isCurrencyRelationField({
      type: 'ManyToOne',
      relation: {
        targetModel: () => {
          throw new Error('boom');
        },
      },
    } as any)
  ).toBe(false);

  expect(
    isCurrencyRelationField({
      type: 'ManyToOne',
      relation: { targetModel: 'not-a-fn' as any },
    } as any)
  ).toBe(false);
});

test('validateModelMonetaryCurrencyFields covers missing and invalid siblings', () => {
  validateModelMonetaryCurrencyFields({ fields: undefined } as any);
  validateModelMonetaryCurrencyFields({ fields: new Map() } as any);

  expect(() =>
    validateModelMonetaryCurrencyFields({
      fields: new Map([['Amount', { type: 'monetary', column: {} }]]),
    } as any)
  ).toThrow('monetary requires currencyField');

  expect(() =>
    validateModelMonetaryCurrencyFields({
      fields: new Map([['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }]]),
    } as any)
  ).toThrow('does not exist on the model');

  expect(() =>
    validateModelMonetaryCurrencyFields({
      fields: new Map([
        ['CurrencyId', { type: 'varchar' }],
        ['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }],
      ]),
    } as any)
  ).toThrow(`targeting ${BASE_CURRENCY_FULL_NAME}`);

  validateModelMonetaryCurrencyFields({
    fields: new Map([
      ['CurrencyId', { type: 'ManyToOneRef', relation: { targetModel: 'base.Currency' } }],
      ['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }],
      ['Name', { type: 'varchar' }],
      ['Broken', null as any],
    ]),
  } as any);
});

test('resolveMonetaryScaleFromPayload / ForWrite / FromRow cover S1 and E1 branches', () => {
  expect(resolveMonetaryScaleFromPayload(undefined, {})).toEqual({ needsBrowse: false });
  expect(resolveMonetaryScaleFromPayload({ type: 'decimal' } as any, {})).toEqual({ needsBrowse: false });

  const fm = { type: 'monetary', name: 'Amount', column: { currencyField: 'CurrencyId' } } as any;
  const hidden = buildHiddenScaleAlias('Amount');
  expect(resolveMonetaryScaleFromPayload(fm, { [hidden]: 3 })).toEqual({ scale: 3, needsBrowse: false });

  expect(
    resolveMonetaryScaleFromPayload(fm, { CurrencyId: { Id: 'C1', DecimalDigits: 0 } })
  ).toEqual({ scale: 0, currencyId: 'C1', needsBrowse: false });

  expect(resolveMonetaryScaleFromPayload(fm, { CurrencyId: 'C2' }, null)).toEqual({
    currencyId: 'C2',
    needsBrowse: true,
  });

  expect(resolveMonetaryScaleFromPayload(fm, {}, { CurrencyId: 'C3' })).toEqual({
    currencyId: 'C3',
    needsBrowse: true,
  });

  expect(() => resolveMonetaryScaleFromPayload(fm, {})).toThrow(/currency required/);
  expect(() =>
    resolveMonetaryScaleFromPayload({ type: 'monetary', name: 'Amount', column: {} } as any, { Amount: 1 })
  ).toThrow(/currency required/);

  expect(resolveMonetaryScaleForWrite(fm, { [hidden]: 4 })).toBe(4);
  expect(() => resolveMonetaryScaleForWrite(fm, { CurrencyId: 'C9' })).toThrow(/currency required/);
  expect(monetaryCurrencyRequiredError('X').message).toContain('X');

  expect(resolveMonetaryScaleFromRow(undefined, 'Amount', {})).toBeUndefined();
  expect(resolveMonetaryScaleFromRow({ type: 'decimal' } as any, 'Amount', {})).toBeUndefined();
  expect(resolveMonetaryScaleFromRow(fm, 'Amount', null)).toBeUndefined();
  expect(resolveMonetaryScaleFromRow(fm, 'Amount', { [hidden]: 1 })).toBe(1);
  expect(resolveMonetaryScaleFromRow(fm, 'Amount', { CurrencyId: { DecimalDigits: 2 } })).toBe(2);
  expect(resolveMonetaryScaleFromRow(fm, 'Amount', { CurrencyId: 'C1' })).toBeUndefined();
  expect(
    resolveMonetaryScaleFromRow({ type: 'monetary', name: 'Amount', column: {} } as any, 'Amount', { x: 1 })
  ).toBeUndefined();
});

test('validateModelMonetaryCurrencyFields skips unresolved ManyToOne targets during load', () => {
  expect(() =>
    validateModelMonetaryCurrencyFields({
      fields: new Map([
        [
          'CurrencyId',
          {
            type: 'ManyToOne',
            relation: {
              targetModel: () => {
                throw new Error('not registered yet');
              },
            },
          },
        ],
        ['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }],
      ]),
    } as any)
  ).not.toThrow();
});

test('validateModelMonetaryCurrencyFields accepts ManyToOne targeting base.Currency', () => {
  @Model('Currency', { application: 'base' })
  class BaseCurrencyModel extends BaseModel {}

  validateModelMonetaryCurrencyFields({
    fields: new Map([
      [
        'CurrencyId',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => BaseCurrencyModel },
        },
      ],
      ['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }],
    ]),
  } as any);
});

test('validateModelMonetaryCurrencyFields rejects ManyToOne targeting non-Currency', () => {
  @Model('Country', { application: 'base' })
  class BaseCountryModel extends BaseModel {}

  expect(() =>
    validateModelMonetaryCurrencyFields({
      fields: new Map([
        [
          'CurrencyId',
          {
            type: 'ManyToOne',
            relation: { targetModel: () => BaseCountryModel },
          },
        ],
        ['Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } }],
      ]),
    } as any)
  ).toThrow(`targeting ${BASE_CURRENCY_FULL_NAME}`);
});

test('resolveMonetaryScaleFromPayload ignores non-integer hidden scale alias', () => {
  const fm = { type: 'monetary', name: 'Amount', column: { currencyField: 'CurrencyId' } } as any;
  const hidden = buildHiddenScaleAlias('Amount');
  expect(
    resolveMonetaryScaleFromPayload(fm, {
      [hidden]: 'x',
      CurrencyId: { Id: 'C1', DecimalDigits: 2 },
    })
  ).toEqual({ scale: 2, currencyId: 'C1', needsBrowse: false });

  expect(() => resolveMonetaryScaleForWrite(undefined as any, {})).toThrow(/currency required/);
});
