// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { companyCode8, currencyCode3, uid } from './_helpers';

async function expectRepoValidationFailed(mode: 'create' | 'update', fn: () => Promise<void>): Promise<void> {
  let error: unknown;
  try {
    await fn();
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe(mode);
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
}

test('base.company: invalid timezone is rejected', async () => {
  const currency = await Currency.Create(
    {
      Name: uid('Currency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('create', async () => {
    await Company.Create(
      {
        Name: uid('Company'),
        Code: companyCode8(),
        Timezone: 'Invalid/Timezone',
        CurrencyId: currency.Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.company: empty timezone is rejected', async () => {
  const currency = await Currency.Create(
    {
      Name: uid('CurrencyEmptyTz'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('create', async () => {
    await Company.Create(
      {
        Name: uid('CompanyEmptyTz'),
        Code: companyCode8(),
        Timezone: '   ',
        CurrencyId: currency.Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.company: Timezone FieldsGet exposes dynamic IANA selection', async () => {
  const meta = await Company.FieldsGet(['Timezone'], ['type', 'selectionKind', 'selection']);
  expect(meta.Timezone?.type).toBe('selection');
  expect(meta.Timezone?.selectionKind).toBe('dynamic');
  const selection = meta.Timezone?.selection || [];
  expect(selection.length).toBeGreaterThan(100);
  expect(selection.some((item: { value?: string }) => item.value === 'UTC')).toBe(true);
});

test('base.company: missing CurrencyId is rejected', async () => {
  await expectRepoValidationFailed('create', async () => {
    await Company.Create(
      {
        Name: uid('Company'),
        Code: companyCode8(),
        Timezone: 'UTC',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.company: ParentId update rejects self and descendant cycle', async () => {
  const currency = await Currency.Create(
    {
      Name: uid('Currency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const parent = await Company.Create(
    {
      Name: uid('ParentCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: currency.Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const child = await Company.Create(
    {
      Name: uid('ChildCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: currency.Id,
      ParentId: (parent as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('update', async () => {
    await Company.UpdateById(
      String((parent as any).Id),
      {
        ParentId: (parent as any).Id,
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed('update', async () => {
    await Company.UpdateById(
      String((parent as any).Id),
      {
        ParentId: (child as any).Id,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.company: enforces global uniqueness for Code (Name may repeat)', async () => {
  const nameField = MetadataStorage.instance.getModelMetadata(Company).fields.get('Name');
  expect(nameField?.translate).toBe(true);
  expect(nameField?.column?.unique).toBeFalsy();
  expect(nameField?.column?.index).toBe('trigram');

  const currency = await Currency.Create(
    {
      Name: uid('CurrencyUniq'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const enName = uid('UniqCompanyName');
  const zhName = uid('UniqCompanyZh');
  const base = await Company.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: currency.Id,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((base as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () =>
    Company.Browse(String((base as any).Id), ['Id', 'Name'] as any)
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  await expectRepoValidationFailed('create', async () => {
    await Company.Create(
      {
        Name: uid('OtherName'),
        Code: (base as any).Code,
        Timezone: 'UTC',
        CurrencyId: currency.Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  // Duplicate Name with a different Code is allowed after D21 translate migration
  const dupName = await Company.Create(
    {
      Name: enName,
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: currency.Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((dupName as any).Id)).toBe(true);
});
