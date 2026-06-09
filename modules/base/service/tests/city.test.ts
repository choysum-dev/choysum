// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import City from '@/base/service/models/city';
import Country from '@/base/service/models/country';
import State from '@/base/service/models/state';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';

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

test('base.city: enforces State.CountryId consistency and name uniqueness', async () => {
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

  const created = await City.Create(
    {
      Name: sharedName,
      Code: ' ab12 ',
      CountryId: (countryA as any).Id,
      StateId: (stateA as any).Id,
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );
  expect(String((created as any).Code)).toBe('AB12');

  await expectRepoValidationFailed(async () => {
    await City.Create(
      {
        Name: sharedName,
        CountryId: (countryA as any).Id,
        StateId: (stateA as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  const stateA2 = await State.Create(
    {
      Name: uid('StateA2'),
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  // Same name in a different state is allowed
  const okDifferentState = await City.Create(
    {
      Name: sharedName,
      CountryId: (countryA as any).Id,
      StateId: (stateA2 as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((okDifferentState as any).Id)).toBe(true);

  // StateId=null participates in uniqueness too
  const noStateName = uid('NoStateCity');
  await City.Create(
    {
      Name: noStateName,
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed(async () => {
    await City.Create(
      {
        Name: noStateName,
        CountryId: (countryA as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });
});
