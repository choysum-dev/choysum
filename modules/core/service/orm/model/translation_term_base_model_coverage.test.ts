// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator';
import { ChoysumError } from '@/core/service/error';
import { MetadataStorage } from '../metadata/storage';
import TranslationTermBaseModel, { __resetTranslationTermUniqueIndexTablesForTest } from './translation_term_base_model';

@Model('TranslationTerm', { application: 'ttcov', softDelete: false })
class TtCovTerm extends TranslationTermBaseModel {}

@Model('TranslationTerm', { application: 'core', softDelete: false })
class TtCoreTerm extends TranslationTermBaseModel {}

@Model('TranslationTerm', { application: 'ttsoft' })
class TtSoftTerm extends TranslationTermBaseModel {}

async function expectRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe(code);
  }
}

function installSearch(ctor: typeof TtCovTerm, rows: any[] | null | undefined) {
  const original = ctor.Search;
  ctor.Search = (async () => rows) as any;
  return () => {
    ctor.Search = original;
  };
}

test('GetTranslations rejects missing lang and softDelete models', async () => {
  await expectRejects(TtCovTerm.GetTranslations({} as any), 'TRANSLATION_TERM_LANG_REQUIRED');
  await expectRejects(TtCovTerm.GetTranslations({ lang: '   ' }), 'TRANSLATION_TERM_LANG_REQUIRED');
  await expectRejects(TtSoftTerm.GetTranslations({ lang: 'zh_CN' }), 'TRANSLATION_TERM_SOFT_DELETE');
});

test('GetTranslations core/empty application returns empty stable hash', async () => {
  const empty = await TtCoreTerm.GetTranslations({ lang: 'en_US' });
  expect(empty.hash).toBe('e3b0c44298fc1c14');
  expect(empty.unchanged).toBe(false);
  expect(empty.terms_by_module).toEqual({});

  const match = await TtCoreTerm.GetTranslations({ lang: 'en_US', hash: 'e3b0c44298fc1c14' });
  expect(match.unchanged).toBe(true);
  expect(match.terms_by_module).toEqual({});
});

test('GetTranslations hashes language-wide and filters module_names', async () => {
  __resetTranslationTermUniqueIndexTablesForTest();
  const rows = [
    { Module: 'auth', Scope: 'ui', Src: 'Hello', Value: '你好', Kind: 'literal', Source: 'packaged' },
    { Module: 'auth', Scope: 'ui', Src: 'Bye', Value: '再见', Kind: 'model', Source: 'packaged' },
    { Module: 'base', Scope: 'msg', Src: 'Ok', Value: '好', Kind: '', Source: '' },
    { Module: '  ', Scope: 'x', Src: 'skip', Value: 'n', Kind: 'literal', Source: 'packaged' },
    { Module: 'auth', Scope: 'ui', Src: '', Value: 'empty-src', Kind: 'literal', Source: 'packaged' },
    { Module: 'web', Scope: 'a', Src: 'A', Value: '1', Kind: 'literal', Source: 'override' },
    { Module: 'web', Scope: 'b', Src: 'B', Value: '2', Kind: 'literal', Source: 'packaged' },
  ];
  const restore = installSearch(TtCovTerm, rows);
  const originalChoysum = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { db: { dialectName: 'sqlite', execute: async () => undefined } };
  try {
    const emptyMods = await TtCovTerm.GetTranslations({ lang: 'zh_CN' });
    expect(emptyMods.unchanged).toBe(false);
    expect(emptyMods.terms_by_module).toEqual({});
    expect(emptyMods.hash).not.toBe('e3b0c44298fc1c14');

    const filtered = await TtCovTerm.GetTranslations({
      lang: 'zh_CN',
      module_names: ['auth', 'auth', '  ', 'missing', 'web'],
    });
    expect(filtered.hash).toBe(emptyMods.hash);
    expect(filtered.terms_by_module).toEqual({
      auth: { ui: { Hello: '你好' } },
      web: { a: { A: '1' }, b: { B: '2' } },
    });

    const camel = await TtCovTerm.GetTranslations({
      lang: 'zh_CN',
      moduleNames: ['base'],
    });
    expect(camel.terms_by_module).toEqual({
      base: { msg: { Ok: '好' } },
    });

    const unchanged = await TtCovTerm.GetTranslations({ lang: 'zh_CN', hash: emptyMods.hash });
    expect(unchanged).toEqual({ lang: 'zh_CN', hash: emptyMods.hash, unchanged: true });
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    restore();
    __resetTranslationTermUniqueIndexTablesForTest();
  }
});

test('GetTranslations tolerates non-array Search and empty catalog', async () => {
  __resetTranslationTermUniqueIndexTablesForTest();
  const restore = installSearch(TtCovTerm, null as any);
  const originalChoysum = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { db: { dialectName: 'sqlite' } };
  try {
    const out = await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['auth'] });
    expect(out.hash).toBe('e3b0c44298fc1c14');
    expect(out.terms_by_module).toEqual({});
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    restore();
    __resetTranslationTermUniqueIndexTablesForTest();
  }
});

test('ensureTermUniqueIndex covers dialects, cache, and duplicate errors', async () => {
  __resetTranslationTermUniqueIndexTablesForTest();
  const restore = installSearch(TtCovTerm, []);
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];

  try {
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'postgres',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls.some(d => d.includes('IF NOT EXISTS') && d.includes('"'))).toBe(true);
    const afterPg = ddls.length;
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls.length).toBe(afterPg);

    __resetTranslationTermUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'postgresql',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls[ddls.length - 1]).toContain('CREATE UNIQUE INDEX IF NOT EXISTS');

    __resetTranslationTermUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'mysql',
        execute: async (ddl: string) => {
          ddls.push(ddl);
          throw new Error('Duplicate key name');
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    const afterDup = ddls.length;
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls.length).toBe(afterDup);

    __resetTranslationTermUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'mysql',
        execute: async () => {
          throw new Error('connection reset');
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });

    __resetTranslationTermUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls[ddls.length - 1]).toContain('`');

    // tableName as function + empty table early-return path via metadata tweak
    __resetTranslationTermUniqueIndexTablesForTest();
    const meta = MetadataStorage.instance.getModelMetadata(TtCovTerm as any);
    const originalTable = meta.tableName;
    meta.tableName = (() => 'tt_fn_table') as any;
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls.some(d => d.includes('tt_fn_table'))).toBe(true);

    meta.tableName = '' as any;
    __resetTranslationTermUniqueIndexTablesForTest();
    const beforeEmpty = ddls.length;
    await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['x'] });
    expect(ddls.length).toBe(beforeEmpty);
    meta.tableName = originalTable;
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    restore();
    __resetTranslationTermUniqueIndexTablesForTest();
  }
});

test('computeTermHash sort comparator covers all key dimensions', async () => {
  __resetTranslationTermUniqueIndexTablesForTest();
  // Pairs differ only on one sort key so every branch of the comparator runs.
  const rows = [
    { Module: 'b', Scope: 's', Src: 's', Value: 'v', Kind: 'literal', Source: 'packaged' },
    { Module: 'a', Scope: 's', Src: 's', Value: 'v', Kind: 'literal', Source: 'packaged' },
    { Module: 'a', Scope: 't', Src: 's', Value: 'v', Kind: 'literal', Source: 'packaged' },
    { Module: 'a', Scope: 's', Src: 't', Value: 'v', Kind: 'literal', Source: 'packaged' },
    { Module: 'a', Scope: 's', Src: 's', Value: 'v', Kind: 'model', Source: 'packaged' },
    { Module: 'a', Scope: 's', Src: 's', Value: 'w', Kind: 'literal', Source: 'packaged' },
    { Module: 'a', Scope: 's', Src: 's', Value: 'v', Kind: 'literal', Source: 'override' },
    // Identical keys → return 0 branch
    { Module: 'z', Scope: 'z', Src: 'z', Value: 'z', Kind: 'literal', Source: 'packaged' },
    { Module: 'z', Scope: 'z', Src: 'z', Value: 'z', Kind: 'literal', Source: 'packaged' },
  ];
  const restore = installSearch(TtCovTerm, rows);
  const originalChoysum = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { db: { dialectName: 'sqlite', execute: async () => undefined } };
  try {
    const out = await TtCovTerm.GetTranslations({ lang: 'en_US', module_names: ['a', 'b', 'z'] });
    expect(out.hash).toHaveLength(16);
    expect(out.terms_by_module.a.s.s).toBeDefined();
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    restore();
    __resetTranslationTermUniqueIndexTablesForTest();
  }
});

test('parseModuleNames ignores non-array module_names', async () => {
  __resetTranslationTermUniqueIndexTablesForTest();
  const restore = installSearch(TtCovTerm, [
    { Module: 'auth', Scope: 'ui', Src: 'Hi', Value: '你好', Kind: 'literal', Source: 'packaged' },
  ]);
  const originalChoysum = (globalThis as any).$choysum;
  (globalThis as any).$choysum = { db: { dialectName: 'sqlite', execute: async () => undefined } };
  try {
    const out = await TtCovTerm.GetTranslations({ lang: 'zh_CN', module_names: 'auth' as any });
    expect(out.terms_by_module).toEqual({});
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    restore();
    __resetTranslationTermUniqueIndexTablesForTest();
  }
});
