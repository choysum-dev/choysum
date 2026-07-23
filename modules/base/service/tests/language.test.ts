// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Language from '@/base/service/models/language';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { companyCode8, uid } from './_helpers';

test('base.language: Direction selection uses _lt msgid labels', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('Direction');
  const selection = field?.selection as Array<{ value: string; label: string; labelText?: { src?: string; scope?: string } }> | undefined;

  expect(selection?.map(item => item.value)).toEqual(['ltr', 'rtl']);
  expect(selection?.map(item => item.label)).toEqual(['Left to right', 'Right to left']);
  expect(selection?.every(item => item.labelText?.src === item.label)).toBe(true);
  expect(selection?.every(item => item.labelText?.scope === 'base.model.Language.fields')).toBe(true);
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

test('base.language: CurrencySymbolPosition selection uses _lt msgid labels', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('CurrencySymbolPosition');
  const selection = field?.selection as Array<{ value: string; label: string; labelText?: { src?: string; scope?: string } }> | undefined;

  expect(selection?.map(item => item.value)).toEqual(['before', 'after']);
  expect(selection?.map(item => item.label)).toEqual(['Before amount', 'After amount']);
  expect(selection?.every(item => item.labelText?.src === item.label)).toBe(true);
  expect(selection?.every(item => item.labelText?.scope === 'base.model.Language.fields')).toBe(true);
});

test('base.language: CurrencySymbolPosition invalid is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('Language'),
        Code: companyCode8(),
        Direction: 'ltr' as any,
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

test('base.language: CurrencySymbolPosition defaults + spacing defaults', async () => {
  const created = await Language.Create(
    {
      Name: uid('Language'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      CurrencySymbolPosition: null as any,
      CurrencySymbolSpacing: null as any,
      IsActive: true,
    } as any,
    ['Id', 'CurrencySymbolPosition', 'CurrencySymbolSpacing', 'Grouping'] as any
  );

  expect(String((created as any).CurrencySymbolPosition)).toBe('before');
  expect(Boolean((created as any).CurrencySymbolSpacing)).toBe(false);
  expect(String((created as any).Grouping)).toBe('[3,0]');
});

test('base.language: CurrencySymbolPosition blank is rejected', async () => {
  let error: unknown;
  try {
    await Language.Create(
      {
        Name: uid('LanguageBlank'),
        Code: companyCode8(),
        Direction: 'ltr' as any,
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

test('base.language: Create with format fields succeeds', async () => {
  const created = await Language.Create(
    {
      Name: uid('LanguageFmt'),
      Code: companyCode8(),
      Direction: 'ltr' as any,
      IsActive: true,
      DecimalSeparator: '.',
      ThousandSeparator: ',',
      Grouping: '[3,0]',
      DateFormat: 'YYYY-MM-DD',
      TimeFormat: 'HH:mm:ss',
      FirstDayOfWeek: 1,
      CurrencySymbolPosition: 'before' as any,
      CurrencySymbolSpacing: false,
    } as any,
    ['Id', 'Code', 'DecimalSeparator', 'ThousandSeparator', 'Grouping', 'FirstDayOfWeek'] as any
  );

  expect(String((created as any).DecimalSeparator)).toBe('.');
  expect(String((created as any).ThousandSeparator)).toBe(',');
  expect(String((created as any).Grouping)).toBe('[3,0]');
  expect(Number((created as any).FirstDayOfWeek)).toBe(1);
});

test('base.language: GetActiveLanguages returns only active rows', async () => {
  const suffix = companyCode8();
  await Language.Create(
    {
      Name: uid('LangActive'),
      Code: `a_${suffix}`.slice(0, 16),
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  await Language.Create(
    {
      Name: uid('LangInactive'),
      Code: `i_${suffix}`.slice(0, 16),
      Direction: 'ltr' as any,
      IsActive: false,
    } as any,
    ['Id'] as any
  );

  const rows = await Language.GetActiveLanguages();
  expect(Array.isArray(rows)).toBe(true);
  expect(rows.every(r => r.Code && r.Name)).toBe(true);
  expect(rows.some(r => r.Code === `i_${suffix}`.slice(0, 16))).toBe(false);
  expect(rows.some(r => r.Code === `a_${suffix}`.slice(0, 16))).toBe(true);
});

test('base.language: Name translate metadata is enabled', () => {
  const field = MetadataStorage.instance.getModelMetadata(Language).fields.get('Name');
  expect(field?.translate).toBe(true);
  expect(field?.type).toBe('varchar');
  expect(field?.column?.size).toBeUndefined();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(100);
});

test('base.language: Name bilingual write/read unwraps by lang', async () => {
  const code = `t_${companyCode8()}`.slice(0, 16);
  const enName = uid('LangEn');
  const zhName = uid('LangZh');

  const created = await Language.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: code,
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((created as any).Name)).toBe(enName);

  const id = String((created as any).Id);
  const enBrowse = await withContext({ lang: 'en_US' }, () => Language.Browse(id, ['Id', 'Name'] as any));
  expect(String((enBrowse as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () => Language.Browse(id, ['Id', 'Name'] as any));
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  const fallbackBrowse = await withContext({ lang: 'fr_FR' }, () => Language.Browse(id, ['Id', 'Name'] as any));
  expect(String((fallbackBrowse as any).Name)).toBe(enName);

  await withContext({ lang: 'zh_CN' }, async () => {
    await Language.UpdateById(id, { Name: `${zhName}_upd` } as any, ['Id', 'Name'] as any);
  });
  const afterZh = await withContext({ lang: 'zh_CN' }, () => Language.Browse(id, ['Id', 'Name'] as any));
  expect(String((afterZh as any).Name)).toBe(`${zhName}_upd`);
  const afterEn = await withContext({ lang: 'en_US' }, () => Language.Browse(id, ['Id', 'Name'] as any));
  expect(String((afterEn as any).Name)).toBe(enName);

  const hit = await withContext({ lang: 'zh_CN' }, () =>
    Language.Search(['Name', 'ilike', `${zhName}_upd`] as any, { fields: ['Id', 'Name', 'Code'], limit: 5 } as any)
  );
  expect(hit?.some((r: any) => String(r.Code) === code)).toBe(true);

  // UI keyword search historically compiled DisplayName SqlCompute → bare Name jsonb LIKE.
  const hitDisplayName = await Language.Search(['DisplayName', 'like', `%${enName}%`] as any, {
    fields: ['Id', 'Code'],
    limit: 5,
  } as any);
  expect(hitDisplayName?.some((r: any) => String(r.Code) === code)).toBe(true);

  const active = await withContext({ lang: 'zh_CN' }, () => Language.GetActiveLanguages());
  const row = active.find(r => r.Code === code);
  expect(row?.Name).toBe(`${zhName}_upd`);
});

test('base.language: Get/UpdateFieldTranslations maintain lang map', async () => {
  const code = `f_${companyCode8()}`.slice(0, 16);
  const created = await Language.Create(
    {
      Name: { en_US: 'Alpha', zh_CN: '阿尔法' } as any,
      Code: code,
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  const id = String((created as any).Id);

  const all = await Language.GetFieldTranslations(id, 'Name');
  expect(all).toEqual({ en_US: 'Alpha', zh_CN: '阿尔法' });
  expect(await Language.GetFieldTranslations(id, 'Name', ['zh_CN'])).toEqual({ zh_CN: '阿尔法' });

  await Language.UpdateFieldTranslations(id, 'Name', {
    zh_CN: '甲',
    fr_FR: 'AlphaFR',
  });
  expect(await Language.GetFieldTranslations(id, 'Name')).toEqual({
    en_US: 'Alpha',
    zh_CN: '甲',
    fr_FR: 'AlphaFR',
  });

  await Language.UpdateFieldTranslations(id, 'Name', { fr_FR: false, de_DE: '' });
  expect(await Language.GetFieldTranslations(id, 'Name')).toEqual({
    en_US: 'Alpha',
    zh_CN: '甲',
    de_DE: '',
  });

  let baseDeleteErr: unknown;
  try {
    await Language.UpdateFieldTranslations(id, 'Name', { en_US: false });
  } catch (err) {
    baseDeleteErr = err;
  }
  expect(String((baseDeleteErr as Error)?.message || baseDeleteErr)).toMatch(/cannot delete base language|en_US/);

  const zh = await withContext({ lang: 'zh_CN' }, () => Language.Browse(id, ['Name'] as any));
  expect(String((zh as any).Name)).toBe('甲');
});

test('base.language: partial IsActive=false still blocks deactivating en_US', async () => {
  const enRows = await Language.Search(['Code', '=', 'en_US'] as any, { fields: ['Id', 'Code', 'IsActive'], limit: 1 } as any);
  expect(enRows?.length).toBe(1);
  const enId = String((enRows[0] as any).Id);

  // Ensure another active language exists so "last active" is not the failure mode.
  await Language.Create(
    {
      Name: uid('LangKeepActive'),
      Code: `k_${companyCode8()}`.slice(0, 16),
      Direction: 'ltr' as any,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  let error: unknown;
  try {
    await Language.UpdateById(enId, { IsActive: false } as any, ['Id', 'IsActive'] as any);
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(String(oe.message || '')).toMatch(/en_US|root language/i);

  const after = await Language.Browse(enId, ['Id', 'IsActive'] as any);
  expect(Boolean((after as any).IsActive)).toBe(true);
});
