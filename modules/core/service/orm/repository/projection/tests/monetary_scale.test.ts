// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../../model/model';
import { buildHiddenScaleAlias } from '../../hidden_scale_alias';
import {
  browseCurrencyDecimalDigits,
  collectMonetaryCurrencyFieldCompanions,
  currencyIdOf,
  readCurrencyDigitsInline,
  stampMonetaryScalesForWrite,
  stampMonetaryScalesForWriteMany,
} from '../monetary_scale';

function monetaryMeta() {
  return {
    fields: new Map<string, any>([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
      ['Note', { type: 'varchar', column: { name: 'Note' } }],
    ]),
  } as any;
}

test('stampMonetaryScalesForWrite stamps inline digits and skips unrelated writes', async () => {
  const meta = monetaryMeta();
  expect(await stampMonetaryScalesForWrite(meta, null as any)).toBeNull();
  expect(await stampMonetaryScalesForWrite({ fields: undefined } as any, { Amount: 1 } as any)).toEqual({ Amount: 1 });

  const skipped = await stampMonetaryScalesForWrite(meta, { Note: 'x' } as any);
  expect(skipped).toEqual({ Note: 'x' });
  expect((skipped as any)[buildHiddenScaleAlias('Amount')]).toBeUndefined();

  const stamped = await stampMonetaryScalesForWrite(meta, {
    Amount: '1.23',
    CurrencyId: { Id: 'C1', DecimalDigits: 0 },
  } as any);
  expect((stamped as any)[buildHiddenScaleAlias('Amount')]).toBe(0);

  const fromCurrent = await stampMonetaryScalesForWrite(
    meta,
    { Amount: '1.23' } as any,
    { CurrencyId: { Id: 'C1', DecimalDigits: 3 } } as any
  );
  expect((fromCurrent as any)[buildHiddenScaleAlias('Amount')]).toBe(3);
});

test('stampMonetaryScalesForWrite browses currency digits and throws E1 when missing', async () => {
  const meta = monetaryMeta();
  const browser = async (ids: string[]) => {
    expect(ids).toEqual(['C9']);
    return new Map([['C9', 2]]);
  };
  const stamped = await stampMonetaryScalesForWrite(meta, { Amount: '1.239', CurrencyId: 'C9' } as any, null, browser);
  expect((stamped as any)[buildHiddenScaleAlias('Amount')]).toBe(2);

  let missingBrowseErr = '';
  try {
    await stampMonetaryScalesForWrite(meta, { Amount: '1.239', CurrencyId: 'MISSING' } as any, null, async () => new Map());
  } catch (error) {
    missingBrowseErr = String((error as Error).message || error);
  }
  expect(missingBrowseErr.includes('currency required for monetary field Amount')).toBe(true);

  let missingCurrencyErr = '';
  try {
    await stampMonetaryScalesForWrite(meta, { Amount: '1.239' } as any);
  } catch (error) {
    missingCurrencyErr = String((error as Error).message || error);
  }
  expect(missingCurrencyErr.includes('currency required for monetary field Amount')).toBe(true);

  // Writing only currency without amount still stamps when digits are available.
  const currencyOnly = await stampMonetaryScalesForWrite(meta, {
    CurrencyId: { Id: 'C1', DecimalDigits: 4 },
  } as any);
  expect((currencyOnly as any)[buildHiddenScaleAlias('Amount')]).toBe(4);
});

test('stampMonetaryScalesForWriteMany batches one browse across entities', async () => {
  const meta = monetaryMeta();
  let browseCalls = 0;
  const browser = async (ids: string[]) => {
    browseCalls += 1;
    expect([...ids].sort()).toEqual(['C1', 'C2']);
    return new Map([
      ['C1', 2],
      ['C2', 0],
    ]);
  };

  const stamped = await stampMonetaryScalesForWriteMany(
    meta,
    [
      { input: { Amount: '1.239', CurrencyId: 'C1' } as any },
      { input: { Amount: '9.87', CurrencyId: 'C2' } as any },
    ],
    browser
  );

  expect(browseCalls).toBe(1);
  expect((stamped[0] as any)[buildHiddenScaleAlias('Amount')]).toBe(2);
  expect((stamped[1] as any)[buildHiddenScaleAlias('Amount')]).toBe(0);
  expect(await stampMonetaryScalesForWriteMany(meta, [], browser)).toEqual([]);
  expect(browseCalls).toBe(1);
});

test('stampMonetaryScalesForWrite covers currency-only browse and empty amount edges', async () => {
  const meta = monetaryMeta();
  const browser = async (ids: string[]) => {
    expect(ids).toEqual(['C9']);
    return new Map([['C9', 3]]);
  };
  const currencyOnly = await stampMonetaryScalesForWrite(meta, { CurrencyId: 'C9' } as any, null, browser);
  expect((currencyOnly as any)[buildHiddenScaleAlias('Amount')]).toBe(3);

  const emptyAmount = await stampMonetaryScalesForWrite(meta, { Amount: '', CurrencyId: { Id: 'C1', DecimalDigits: 1 } } as any);
  expect((emptyAmount as any)[buildHiddenScaleAlias('Amount')]).toBe(1);

  let manyErr = '';
  try {
    await stampMonetaryScalesForWriteMany(meta, [{ input: { Amount: '1.2' } as any }]);
  } catch (error) {
    manyErr = String((error as Error).message || error);
  }
  expect(manyErr.includes('currency required for monetary field Amount')).toBe(true);

  let currencyOnlyMiss = '';
  try {
    await stampMonetaryScalesForWrite(meta, { CurrencyId: 'MISSING' } as any, null, async () => new Map());
  } catch (error) {
    currencyOnlyMiss = String((error as Error).message || error);
  }
  expect(currencyOnlyMiss.includes('currency required for monetary field Amount')).toBe(true);
});

test('stampMonetaryScalesForWriteMany soft-skips collect when currencyField missing', async () => {
  const meta = {
    fields: new Map([['Amount', { type: 'monetary', column: {} }]]),
  } as any;
  let err = '';
  try {
    // collectPendingCurrencyIdsForStamp swallows resolve errors; stamp pass then raises E1.
    await stampMonetaryScalesForWriteMany(meta, [{ input: { Amount: '1.20' } as any }]);
  } catch (error) {
    err = String((error as Error).message || error);
  }
  expect(err.includes('currency required for monetary field Amount')).toBe(true);
});

test('collectMonetaryCurrencyFieldCompanions and inline helpers', () => {
  const meta = monetaryMeta();
  expect(collectMonetaryCurrencyFieldCompanions(meta, ['Amount', 'Note', 'Missing'])).toEqual(['CurrencyId']);
  expect(collectMonetaryCurrencyFieldCompanions({ fields: undefined } as any, ['Amount'])).toEqual([]);
  expect(
    collectMonetaryCurrencyFieldCompanions(
      {
        fields: new Map([['Amount', { type: 'monetary', column: {} }]]),
      } as any,
      ['Amount']
    )
  ).toEqual([]);
  expect(readCurrencyDigitsInline({ DecimalDigits: 2 })).toBe(2);
  expect(currencyIdOf(' X ')).toBe('X');
});

test('browseCurrencyDecimalDigits uses BaseModel.resolveModelConstructor when present', async () => {
  expect((await browseCurrencyDecimalDigits([])).size).toBe(0);
  expect((await browseCurrencyDecimalDigits(['', '  '])).size).toBe(0);

  const original = BaseModel.resolveModelConstructor;
  try {
    (BaseModel as any).resolveModelConstructor = () => undefined;
    expect((await browseCurrencyDecimalDigits(['C1'])).size).toBe(0);

    (BaseModel as any).resolveModelConstructor = () => ({});
    expect((await browseCurrencyDecimalDigits(['C1'])).size).toBe(0);

    (BaseModel as any).resolveModelConstructor = () => ({
      BrowseMany: async () => [
        { Id: 'C1', DecimalDigits: 2 },
        { Id: '', DecimalDigits: 2 },
        { Id: 'C2', DecimalDigits: 99 },
        { Id: 'C3', DecimalDigits: 0 },
        null,
      ],
    });
    const map = await browseCurrencyDecimalDigits(['C1', 'C1', 'C3']);
    expect(map.get('C1')).toBe(2);
    expect(map.get('C3')).toBe(0);
    expect(map.has('C2')).toBe(false);
  } finally {
    (BaseModel as any).resolveModelConstructor = original;
  }
});
