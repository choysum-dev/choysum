// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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

test('base.state: enforces name/code uniqueness within Country and normalizes Code', async () => {
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

  const nameA = uid('StateName');
  const created = await State.Create(
    {
      Name: nameA,
      Code: ' ca ',
      CountryId: (countryA as any).Id,
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );
  expect(String((created as any).Code)).toBe('CA');

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

  await expectRepoValidationFailed(async () => {
    await State.Create(
      {
        Name: nameA,
        Code: uid('X').slice(0, 8),
        CountryId: (countryA as any).Id,
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

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
