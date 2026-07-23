// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Country from '@/base/service/models/country';
import { ChoysumError } from '@/core/service/error';
import { ValidationPipelineError } from '@/core/service/api/constraint';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { getTestRepository } from '@/core/service/testing';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { countryCode8, uid } from './_helpers';

function isPostgresDialect(): boolean {
  const dialect = String((globalThis as any)?.$choysum?.db?.dialectName || '')
    .trim()
    .toLowerCase();
  return dialect === 'postgres';
}

function tempTableName(prefix: string): string {
  const seed = String(uid(prefix) || '')
    .toLowerCase()
    .replace(/[^a-z0-9_]/g, '')
    .slice(0, 24);
  return `tmp_${prefix}_${seed || 'x'}`;
}

test('base.country: normalizes Code and validates AddressFormat tokens', async () => {
  const expectedCode = countryCode8();
  const created = await Country.Create(
    {
      Name: uid('Country'),
      Code: ` ${expectedCode.toLowerCase()} `,
      ZipRequired: true,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );
  expect(String((created as any).Code)).toBe(expectedCode);

  let error: unknown;
  try {
    await Country.Create(
      {
        Name: uid('CountryBadFmt'),
        Code: countryCode8(),
        AddressFormat: 'X %(BadToken)s Y',
        ZipRequired: true,
        StateRequired: false,
        IsActive: true,
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

  const ok = await Country.Create(
    {
      Name: uid('CountryOkFmt'),
      Code: countryCode8(),
      AddressFormat: '%(Street1)s %(Zip)s %(CountryCode)s',
      ZipRequired: true,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  expect(Boolean((ok as any).Id)).toBe(true);
});

test('base.country: repository exposes stable kernelCode metadata for kernel failures', async () => {
  let error: unknown;
  try {
    await Country.Create(
      {
        Name: uid('CountryKernelCode'),
        // Code is required, so the missing value should trigger the kernel required error.
        ZipRequired: true,
        StateRequired: false,
        IsActive: true,
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
  expect(String(oe.metadata?.mode || '')).toBe('create');

  const summary = resolveValidationSummary(oe);
  const kernelCode = String(summary.kernelCode || '');
  expect(kernelCode.startsWith('kernel_')).toBe(true);

  const kernelIssueCodes = summary.issues.filter(item => String(item?.scope || '') === 'kernel').map(item => String(item?.code || ''));
  expect(kernelIssueCodes.includes(kernelCode)).toBe(true);

  expect(String(summary.getFieldFirstCode('Code') || '').startsWith('kernel_')).toBe(true);
  expect(summary.getFieldKernelCode('Code')).toBe(kernelCode);
  expect(summary.fieldIssueSummary.Code?.firstKernelCode).toBe(kernelCode);
  expect(summary.fieldIssueSummary.Code?.kernelCode).toBe(kernelCode);
  expect(String(summary.fieldIssueSummary.Code?.issueCount || 0)).toBe('1');
});

test('base.country: fieldIssueSummary keeps firstCode and kernelCode stable for platform->kernel mixed order', async () => {
  const repo = getTestRepository(Country as any) as any;
  const pipelineError = new ValidationPipelineError('mixed summary test', [
    {
      scope: 'platform',
      field: 'Code',
      code: 'platform_unknown_field',
      message: 'platform issue first',
      severity: 'error',
    },
    {
      scope: 'kernel',
      field: 'Code',
      code: 'kernel_required_missing',
      message: 'kernel issue second',
      severity: 'error',
    },
  ]);

  const wrapped = repo.wrapValidationError(pipelineError, 'create') as ChoysumError;
  const summary = resolveValidationSummary(wrapped);

  expect(summary.getFieldFirstCode('Code')).toBe('platform_unknown_field');
  expect(summary.getFieldKernelCode('Code')).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Code?.firstKernelCode).toBe('kernel_required_missing');
  expect(summary.fieldIssueSummary.Code?.kernelCode).toBe('kernel_required_missing');
  expect(String(summary.fieldIssueSummary.Code?.issueCount || 0)).toBe('2');
});

test('base.country: sqlite unique violation is normalized to sql metadata', async () => {
  const repo = getTestRepository(Country as any) as any;

  let error: unknown;
  try {
    repo.wrapSqlWriteError(new Error('UNIQUE constraint failed: base_country.code'), 'create');
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('create');
  const summary = resolveValidationSummary(oe);
  expect(oe.metadata?.scope).toBe('sql');
  expect(summary.sqlCode).toBe('sql_unique_violation');
  expect(String(oe.metadata?.sqlField || '').length > 0).toBe(true);

  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_unique_violation')).toBe(true);
});

test('base.country: sqlite check violation text is normalized to sql_check_violation', async () => {
  const repo = getTestRepository(Country as any) as any;

  let error: unknown;
  try {
    repo.wrapSqlWriteError(new Error('CHECK constraint failed: base_country_code_len_check'), 'create');
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('create');
  expect(oe.metadata?.scope).toBe('sql');
  expect(oe.metadata?.sqlCode).toBe('sql_check_violation');
  expect(String(oe.metadata?.sqlConstraint || '').includes('base_country_code_len_check')).toBe(true);

  const summary = resolveValidationSummary(oe);
  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_check_violation')).toBe(true);
});

test('base.country: postgres fk violation text is normalized to sql_fk_violation', async () => {
  const repo = getTestRepository(Country as any) as any;

  let error: unknown;
  try {
    repo.wrapSqlWriteError(new Error('insert or update on table "base_country" violates foreign key constraint "base_country_company_id_fkey"'), 'update');
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe('update');
  expect(oe.metadata?.scope).toBe('sql');
  expect(oe.metadata?.sqlCode).toBe('sql_fk_violation');
  expect(String(oe.metadata?.sqlConstraint || '').includes('base_country_company_id_fkey')).toBe(true);

  const summary = resolveValidationSummary(oe);
  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_fk_violation')).toBe(true);
});

test('base.country: postgres real unique violation is normalized to sql_unique_violation', async () => {
  if (!isPostgresDialect()) return;

  const repo = getTestRepository(Country as any) as any;
  const tableName = tempTableName('pguniq');

  await (globalThis as any).$choysum.db.execute(`create temporary table ${tableName} (id text primary key, code text unique)`, '[]');
  await (globalThis as any).$choysum.db.execute(`insert into ${tableName} (id, code) values ('1', 'dup')`, '[]');

  let wrapped: unknown;
  try {
    await (globalThis as any).$choysum.db.execute(`insert into ${tableName} (id, code) values ('2', 'dup')`, '[]');
  } catch (err) {
    try {
      repo.wrapSqlWriteError(err, 'create');
    } catch (w) {
      wrapped = w;
    }
  }

  expect(wrapped instanceof ChoysumError).toBe(true);
  const oe = wrapped as ChoysumError;
  expect(oe.code).toBe('validation_failed');
  const summary = resolveValidationSummary(oe);
  expect(summary.sqlCode).toBe('sql_unique_violation');
  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_unique_violation')).toBe(true);
});

test('base.country: postgres real check violation is normalized to sql_check_violation', async () => {
  if (!isPostgresDialect()) return;

  const repo = getTestRepository(Country as any) as any;
  const tableName = tempTableName('pgcheck');

  await (globalThis as any).$choysum.db.execute(`create temporary table ${tableName} (id text primary key, qty int check (qty > 0))`, '[]');

  let wrapped: unknown;
  try {
    await (globalThis as any).$choysum.db.execute(`insert into ${tableName} (id, qty) values ('1', 0)`, '[]');
  } catch (err) {
    try {
      repo.wrapSqlWriteError(err, 'create');
    } catch (w) {
      wrapped = w;
    }
  }

  expect(wrapped instanceof ChoysumError).toBe(true);
  const oe = wrapped as ChoysumError;
  expect(oe.code).toBe('validation_failed');
  const summary = resolveValidationSummary(oe);
  expect(summary.sqlCode).toBe('sql_check_violation');
  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_check_violation')).toBe(true);
});

test('base.country: postgres real fk violation is normalized to sql_fk_violation', async () => {
  if (!isPostgresDialect()) return;

  const repo = getTestRepository(Country as any) as any;
  const parentTable = tempTableName('pgparent');
  const childTable = tempTableName('pgchild');

  await (globalThis as any).$choysum.db.execute(`create temporary table ${parentTable} (id text primary key)`, '[]');
  await (globalThis as any).$choysum.db.execute(
    `create temporary table ${childTable} (id text primary key, parent_id text references ${parentTable}(id))`,
    '[]'
  );

  let wrapped: unknown;
  try {
    await (globalThis as any).$choysum.db.execute(`insert into ${childTable} (id, parent_id) values ('1', 'missing')`, '[]');
  } catch (err) {
    try {
      repo.wrapSqlWriteError(err, 'update');
    } catch (w) {
      wrapped = w;
    }
  }

  expect(wrapped instanceof ChoysumError).toBe(true);
  const oe = wrapped as ChoysumError;
  expect(oe.code).toBe('validation_failed');
  const summary = resolveValidationSummary(oe);
  expect(summary.sqlCode).toBe('sql_fk_violation');
  expect(summary.issues.some(item => item.scope === 'sql' && item.code === 'sql_fk_violation')).toBe(true);
});

test('base.country: Name translate metadata is enabled', () => {
  const field = MetadataStorage.instance.getModelMetadata(Country).fields.get('Name');
  expect(field?.translate).toBe(true);
  expect(field?.type).toBe('varchar');
  expect(field?.column?.size).toBeUndefined();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(100);
});

test('base.country: Name bilingual write/read unwraps by lang', async () => {
  const code = countryCode8();
  const enName = uid('CountryEn');
  const zhName = uid('CountryZh');

  const created = await Country.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: code,
      ZipRequired: true,
      StateRequired: false,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((created as any).Name)).toBe(enName);

  const id = String((created as any).Id);
  const enBrowse = await withContext({ lang: 'en_US' }, () => Country.Browse(id, ['Id', 'Name'] as any));
  expect(String((enBrowse as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () => Country.Browse(id, ['Id', 'Name'] as any));
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  const fallbackBrowse = await withContext({ lang: 'fr_FR' }, () => Country.Browse(id, ['Id', 'Name'] as any));
  expect(String((fallbackBrowse as any).Name)).toBe(enName);

  const hit = await withContext({ lang: 'zh_CN' }, () =>
    Country.Search(['Name', 'ilike', zhName] as any, { fields: ['Id', 'Name', 'Code'], limit: 5 } as any)
  );
  expect(hit?.some((r: any) => String(r.Code) === code)).toBe(true);
});
