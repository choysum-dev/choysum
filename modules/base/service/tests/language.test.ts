// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Language from '@/base/service/models/language';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { createTermReference } from '@/core/service/i18n';

import { companyCode8, uid } from './_helpers';

test('base.language: Direction selection exposes localized labels without changing values', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('Direction');

  expect(field?.selection).toEqual([
    {
      value: 'ltr',
      label: 'Left to right',
      labelText: createTermReference('base', 'Left to right', { scope: 'base.Language.Direction.ltr' }),
    },
    {
      value: 'rtl',
      label: 'Right to left',
      labelText: createTermReference('base', 'Right to left', { scope: 'base.Language.Direction.rtl' }),
    },
  ]);
});

test('base.language: Direction invalid is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('Language'),
        Code: companyCode8(),
        Direction: 'sideways' as any,
      } as any,
      ['Id'] as any
    );
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
});

test('base.language: Direction ltr is accepted', async () => {
  const created = await Language.Create(
    {
      Name: uid('Language'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id', 'Direction'] as any
  );

  expect(String((created as any).Direction)).toBe('ltr');
});
