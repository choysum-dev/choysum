// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Locale from '@/base/service/models/locale';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';

import { companyCode8, uid } from './_helpers';

test('base.locale: CurrencySymbolPosition selection uses _lt msgid labels', () => {
  const field = MetadataStorage.instance.getModelMetadata(Locale).fields.get('CurrencySymbolPosition');
  const selection = field?.selection as Array<{ value: string; label: string; labelText?: { src?: string; scope?: string } }> | undefined;

  expect(selection?.map(item => item.value)).toEqual(['before', 'after']);
  expect(selection?.map(item => item.label)).toEqual(['Before amount', 'After amount']);
  expect(selection?.every(item => item.labelText?.src === item.label)).toBe(true);
  expect(selection?.every(item => item.labelText?.scope === 'base.model.Locale.fields')).toBe(true);
});

test('base.locale: CurrencySymbolPosition invalid is rejected', async () => {
  let error: unknown;
  try {
    await Locale.Create(
      {
        Name: uid('Locale'),
        Code: companyCode8(),
        CurrencySymbolPosition: 'middle' as any,
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

test('base.locale: CurrencySymbolPosition defaults + spacing defaults', async () => {
  const created = await Locale.Create(
    {
      Name: uid('Locale'),
      Code: companyCode8(),
      CurrencySymbolPosition: null as any,
      CurrencySymbolSpacing: null as any,
      IsActive: true,
    } as any,
    ['Id', 'CurrencySymbolPosition', 'CurrencySymbolSpacing'] as any
  );

  expect(String((created as any).CurrencySymbolPosition)).toBe('before');
  expect(Boolean((created as any).CurrencySymbolSpacing)).toBe(false);
});

test('base.locale: CurrencySymbolPosition blank is rejected', async () => {
  let error: unknown;
  try {
    await Locale.Create(
      {
        Name: uid('LocaleBlank'),
        Code: companyCode8(),
        CurrencySymbolPosition: '' as any,
        IsActive: true,
      } as any,
      ['Id', 'CurrencySymbolPosition'] as any
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
