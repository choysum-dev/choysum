// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError } from '@/core/service/error';
import UoM from '@/base/service/models/uom';
import UoMCategory from '@/base/service/models/uom_category';
import { uid } from './_helpers';

async function createCategory(): Promise<string> {
  const category = await UoMCategory.Create(
    {
      Name: uid('UomCategory'),
      Code: uid('UCODE').replace(/[^A-Za-z0-9]/g, '').slice(-12) || 'UCODE',
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((category as any).Id);
}

async function createUnit(categoryId: string, factor: string, isReference: boolean, rounding?: string): Promise<string> {
  const row = await UoM.Create(
    {
      Name: uid(isReference ? 'Ref' : 'Uom'),
      CategoryId: categoryId,
      IsReference: isReference,
      Factor: factor,
      Rounding: rounding ?? null,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((row as any).Id);
}

test('base.UoM Convert scales by factor within one category and rounds to the target', async () => {
  const categoryId = await createCategory();
  const gramId = await createUnit(categoryId, '1', true);
  const kilogramId = await createUnit(categoryId, '1000', false, '0.01');

  const result = await UoM.Convert({
    Amount: '2',
    FromUoMId: kilogramId,
    ToUoMId: gramId,
  });
  expect(result.Amount.eq(new Decimal('2000'))).toBe(true);

  const decimalAmount = await UoM.Convert({
    Amount: new Decimal('2'),
    FromUoMId: kilogramId,
    ToUoMId: gramId,
  });
  expect(decimalAmount.Amount.eq(new Decimal('2000'))).toBe(true);

  const back = await UoM.Convert({
    Amount: '15',
    FromUoMId: gramId,
    ToUoMId: kilogramId,
  });
  expect(back.Amount.eq(new Decimal('0.02'))).toBe(true);
});

test('base.UoM Convert reports NotFound when a UoM id is missing', async () => {
  const categoryId = await createCategory();
  const gramId = await createUnit(categoryId, '1', true);
  let error: unknown;
  try {
    await UoM.Convert({ Amount: '1', FromUoMId: gramId, ToUoMId: 'missing_uom_id' });
  } catch (err) {
    error = err;
  }
  expect(error instanceof ChoysumError).toBe(true);
  expect((error as ChoysumError).code).toBe('NotFound');
});

test('base.UoM Convert rejects units from different categories', async () => {
  const weightId = await createCategory();
  const lengthId = await createCategory();
  const gramId = await createUnit(weightId, '1', true);
  const meterId = await createUnit(lengthId, '1', true);

  let error: unknown;
  try {
    await UoM.Convert({ Amount: '1', FromUoMId: gramId, ToUoMId: meterId });
  } catch (err) {
    error = err;
  }
  expect(error instanceof ChoysumError).toBe(true);
  expect((error as ChoysumError).domain).toBe('base');
  expect((error as ChoysumError).code).toBe('InvalidArgument');
});

test('base.UoM Convert rejects a malformed amount', async () => {
  const categoryId = await createCategory();
  const gramId = await createUnit(categoryId, '1', true);
  let error: unknown;
  try {
    await UoM.Convert({ Amount: 'not-a-number', FromUoMId: gramId, ToUoMId: gramId });
  } catch (err) {
    error = err;
  }
  expect(error instanceof ChoysumError).toBe(true);
  expect((error as ChoysumError).code).toBe('InvalidArgument');
});

test('base.UoM Convert rejects a non-decimal Factor as InvalidArgument', async () => {
  const categoryId = await createCategory();
  const gramId = await createUnit(categoryId, '1', true);
  const bad = await UoM.Create(
    {
      Name: uid('BadFactor'),
      CategoryId: categoryId,
      IsReference: false,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  const badId = String((bad as any).Id);
  // Bypass model validation by converting with a stub Browse that returns garbage Factor.
  const { convertUoM } = await import('@/base/service/models/_uom_convert');
  let error: unknown;
  try {
    await convertUoM(
      {
        Browse: async (id: string) => {
          if (id === gramId) return { Id: gramId, CategoryId: categoryId, Factor: '1', Rounding: null } as any;
          return { Id: badId, CategoryId: categoryId, Factor: 'not-a-factor', Rounding: null } as any;
        },
      },
      { Amount: '1', FromUoMId: gramId, ToUoMId: badId }
    );
  } catch (err) {
    error = err;
  }
  expect(error instanceof ChoysumError).toBe(true);
  expect((error as ChoysumError).code).toBe('InvalidArgument');
});

test('base.UoM Convert ignores a non-numeric rounding step', async () => {
  const { convertUoM } = await import('@/base/service/models/_uom_convert');
  const result = await convertUoM(
    {
      Browse: async (id: string) => {
        if (id === 'from') return { Id: 'from', CategoryId: 'cat', Factor: '1', Rounding: null } as any;
        return { Id: 'to', CategoryId: 'cat', Factor: '1', Rounding: 'nope' } as any;
      },
    },
    { Amount: '1.5', FromUoMId: 'from', ToUoMId: 'to' }
  );
  expect(result.Amount.eq(new Decimal('1.5'))).toBe(true);
});
