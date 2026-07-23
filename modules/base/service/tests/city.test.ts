// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import City from '@/base/service/models/city';
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

test('base.city: Name translate metadata; Code unique within Country+State', async () => {
  const nameField = MetadataStorage.instance.getModelMetadata(City).fields.get('Name');
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

  const stateA = await State.Create(
    {
      Name: uid('StateA'),
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const stateB = await State.Create(
    {
      Name: uid('StateB'),
      CountryId: (countryB as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed(async () => {
    await City.Create(
      {
        Name: uid('CityBad'),
        CountryId: (countryA as any).Id,
        StateId: (stateB as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  const sharedName = uid('SharedCity');
  const enName = uid('CityEn');
  const zhName = uid('CityZh');

  const created = await City.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: ' ab12 ',
      CountryId: (countryA as any).Id,
      StateId: (stateA as any).Id,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((created as any).Code)).toBe('AB12');
  expect(String((created as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () =>
    City.Browse(String((created as any).Id), ['Id', 'Name'] as any)
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  // Duplicate Code in same Country+State fails
  await expectRepoValidationFailed(async () => {
    await City.Create(
      {
        Name: uid('OtherCity'),
        Code: 'AB12',
        CountryId: (countryA as any).Id,
        StateId: (stateA as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  // Same display Name allowed (no longer unique); same Code in different state ok
  const stateA2 = await State.Create(
    {
      Name: uid('StateA2'),
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const okDifferentState = await City.Create(
    {
      Name: sharedName,
      Code: 'AB12',
      CountryId: (countryA as any).Id,
      StateId: (stateA2 as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((okDifferentState as any).Id)).toBe(true);

  // Duplicate Name without Code is allowed after translate migration
  await City.Create(
    {
      Name: sharedName,
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  const dupName = await City.Create(
    {
      Name: sharedName,
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((dupName as any).Id)).toBe(true);
});
