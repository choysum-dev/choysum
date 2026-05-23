// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Address from '@/base/service/models/address';
import City from '@/base/service/models/city';
import Country from '@/base/service/models/country';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import State from '@/base/service/models/state';

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

test('base.address: enforce country/state/city consistency', async () => {
  const countryA = await Country.Create(
    {
      Name: uid('CountryA'),
      Code: countryCode8(),
      ZipRequired: true,
      StateRequired: true,
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

  const cityA = await City.Create(
    {
      Name: uid('CityA'),
      CountryId: (countryA as any).Id,
      StateId: (stateA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const stateA2 = await State.Create(
    {
      Name: uid('StateA2'),
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const cityA2 = await City.Create(
    {
      Name: uid('CityA2'),
      CountryId: (countryA as any).Id,
      StateId: (stateA2 as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const cityB = await City.Create(
    {
      Name: uid('CityB'),
      CountryId: (countryB as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed(async () => {
    await Address.Create(
      {
        CountryId: (countryA as any).Id,
        StateId: (stateB as any).Id,
        Zip: '100000',
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed(async () => {
    await Address.Create(
      {
        CountryId: (countryA as any).Id,
        Zip: '100000',
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed(async () => {
    await Address.Create(
      {
        CountryId: (countryA as any).Id,
        StateId: (stateA as any).Id,
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed(async () => {
    await Address.Create(
      {
        CountryId: (countryA as any).Id,
        StateId: (stateA as any).Id,
        CityId: (cityA2 as any).Id,
        Zip: '100000',
      } as any,
      ['Id'] as any
    );
  });

  const ok = await Address.Create(
    {
      CountryId: (countryA as any).Id,
      StateId: (stateA as any).Id,
      CityId: (cityA as any).Id,
      Zip: '100000',
    } as any,
    ['Id'] as any
  );

  expect(Boolean((ok as any).Id)).toBe(true);

  const okNoStateRequired = await Address.Create(
    {
      CountryId: (countryB as any).Id,
      CityId: (cityB as any).Id,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((okNoStateRequired as any).Id)).toBe(true);
});
