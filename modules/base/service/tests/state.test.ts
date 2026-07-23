// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Country from '@/base/service/models/country';
import State from '@/base/service/models/state';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { countryCode8, uid } from './_helpers';

async function expectRepoValidationFailed(fn: () => Promise<void>): Promise<void> {
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
  expect(oe.metadata?.mode).toBe('create');
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
}

test('base.state: Name translate; Code unique within Country', async () => {
  const nameField = MetadataStorage.instance.getModelMetadata(State).fields.get('Name');
  expect(nameField?.translate).toBe(true);
  expect(nameField?.column?.index).toBe('trigram');
  expect(nameField?.column?.uniqueIndex).toBeFalsy();

  const countryA = await Country.Create(
    {
      Name: uid('CountryA'),
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const countryB = await Country.Create(
    {
      Name: uid('CountryB'),
      Code: countryCode8(),
      ZipRequired: false,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const enName = uid('StateEn');
  const zhName = uid('StateZh');
  const created = await State.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: ' ca ',
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((created as any).Code)).toBe('CA');
  expect(String((created as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () =>
    State.Browse(String((created as any).Id), ['Id', 'Name'] as any)
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  await expectRepoValidationFailed(async () => {
    await State.Create(
      {
        Name: uid('OtherName'),
        Code: 'CA',
        CountryId: (countryA as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  // Same display Name allowed within Country (unique moved off Name)
  const sameName = await State.Create(
    {
      Name: enName,
      Code: uid('X').slice(0, 8),
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((sameName as any).Id)).toBe(true);

  // Same code in a different country is allowed
  const okDifferentCountry = await State.Create(
    {
      Name: uid('StateInB'),
      Code: 'CA',
      CountryId: (countryB as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((okDifferentCountry as any).Id)).toBe(true);
});
