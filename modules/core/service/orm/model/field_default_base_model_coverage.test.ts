// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import BaseModel from './model';
import { MetadataStorage } from '../metadata/storage';
import FieldDefaultBaseModel, { __resetFieldDefaultUniqueIndexTablesForTest } from './field_default_base_model';
import { ChoysumError } from '@/core/service/error';

@Model('Widget', { application: 'fd2cov' })
class CovWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => CovWidget } as any })
  ParentId?: CovWidget;

  @Field({ type: 'ManyToOneRef', relation: { targetModel: () => CovWidget } as any })
  ParentRef?: CovWidget;

  @Field({ type: 'image' })
  Avatar!: any;
}

@Model('FieldDefault', { application: 'fd2cov' })
class CovFieldDefault extends FieldDefaultBaseModel {}

@Model('FieldDefault', { application: 'core' })
class CoreFieldDefault extends FieldDefaultBaseModel {}

async function expectRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe(code);
  }
}

function withSavepointPassThrough() {
  const original = CovFieldDefault.withSavepoint;
  CovFieldDefault.withSavepoint = (async (fn: any) => await fn()) as any;
  return () => {
    CovFieldDefault.withSavepoint = original;
  };
}

test('FieldDefault scope dims reject empty and missing context', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  const created: any[] = [];
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => {
    created.push(value);
    return { Id: 'FD-co', ...value };
  }) as any;
  try {
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x', { userId: true }), 'FIELD_DEFAULT_INVALID_VALUE');
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x', { userId: '   ' }), 'FIELD_DEFAULT_INVALID_VALUE');
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x', { companyId: true }), 'FIELD_DEFAULT_INVALID_VALUE');
    await CovFieldDefault.withCompany('C1', async () => {
      await CovFieldDefault.Set('Widget', 'Name', 'co', { companyId: true });
    });
    expect(created[0]?.CompanyId).toBe('C1');
  } finally {
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
  }
});

test('FieldDefault resolveTargetModel/field edge failures', async () => {
  await expectRejects(CovFieldDefault.Set('', 'Name', 'x'), 'FIELD_DEFAULT_UNKNOWN_FIELD');
  await expectRejects(CovFieldDefault.Set('Widget', '', 'x'), 'FIELD_DEFAULT_UNKNOWN_FIELD');
  await expectRejects(CovFieldDefault.Set('MissingModel', 'Name', 'x'), 'FIELD_DEFAULT_CROSS_APP_MODEL');
  await expectRejects(CoreFieldDefault.Set('Widget', 'Name', 'x'), 'FIELD_DEFAULT_CROSS_APP_MODEL');
  await expectRejects(CovFieldDefault.Set('Widget', 'Avatar', 'x'), 'FIELD_DEFAULT_INVALID_VALUE');
  await expectRejects(CovFieldDefault.Set('Widget', 'Name', undefined as any), 'FIELD_DEFAULT_INVALID_VALUE');
});

test('FieldDefault ManyToOne normalize null, string, and empty Id object', async () => {
  const created: any[] = [];
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => {
    created.push(value);
    return { Id: `FD-${created.length}`, ...value };
  }) as any;
  try {
    await CovFieldDefault.Set('Widget', 'ParentId', null);
    expect(created[0].Value).toBe(null);
    await CovFieldDefault.Set('Widget', 'ParentRef', 'W-2');
    expect(created[1].Value).toBe('W-2');
    await expectRejects(CovFieldDefault.Set('Widget', 'ParentId', { Id: '  ' }), 'FIELD_DEFAULT_INVALID_VALUE');
    await expectRejects(CovFieldDefault.Set('Widget', 'ParentId', '   '), 'FIELD_DEFAULT_INVALID_VALUE');
  } finally {
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
  }
});

test('FieldDefault ensureScopeUniqueIndex postgres dialect and exec success/failure', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-x', ...value })) as any;

  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];
  (globalThis as any).$choysum = {
    db: {
      dialectName: 'postgres',
      query: async () => JSON.stringify([{ ok: 1 }]),
      execute: async (ddl: string) => {
        ddls.push(ddl);
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'pg-1');
    expect(ddls.some(d => d.includes('NULLS NOT DISTINCT'))).toBe(true);
    // Second call should skip ensured table.
    const before = ddls.length;
    await CovFieldDefault.Set('Widget', 'Name', 'pg-2');
    expect(ddls.length).toBe(before);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
  }
});

test('FieldDefault ensureScopeUniqueIndex caches permanent DDL failure but retries transient lock', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-ddlcache', ...value })) as any;
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];
  let mode: 'permanent' | 'transient' | 'serialize' = 'permanent';
  (globalThis as any).$choysum = {
    db: {
      dialectName: 'sqlite',
      query: async () => JSON.stringify([{ ok: 1 }]),
      execute: async (ddl: string) => {
        ddls.push(ddl);
        if (mode === 'permanent') {
          throw new Error('syntax error near UNIQUE');
        }
        if (mode === 'serialize') {
          throw new Error('could not serialize access due to concurrent update');
        }
        throw new Error('database is locked');
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'perm-fail');
    expect(ddls.length).toBe(1);
    await CovFieldDefault.Set('Widget', 'Name', 'perm-fail-2');
    expect(ddls.length).toBe(1);

    __resetFieldDefaultUniqueIndexTablesForTest();
    mode = 'transient';
    ddls.length = 0;
    await CovFieldDefault.Set('Widget', 'Name', 'lock-fail');
    expect(ddls.length).toBe(1);
    await CovFieldDefault.Set('Widget', 'Name', 'lock-fail-2');
    expect(ddls.length).toBe(2);

    __resetFieldDefaultUniqueIndexTablesForTest();
    mode = 'serialize';
    ddls.length = 0;
    await CovFieldDefault.Set('Widget', 'Name', 'serialize-fail');
    expect(ddls.length).toBe(1);
    await CovFieldDefault.Set('Widget', 'Name', 'serialize-fail-2');
    expect(ddls.length).toBe(2);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});

test('FieldDefault ensureScopeUniqueIndex swallows exec errors and skips without execute', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-y', ...value })) as any;
  const originalChoysum = (globalThis as any).$choysum;

  (globalThis as any).$choysum = {
    db: {
      dialectName: 'sqlite',
      // Probe says table exists so CREATE INDEX is attempted and can fail.
      query: async () => JSON.stringify([{ ok: 1 }]),
      execute: async () => {
        throw new Error('ddl failed');
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'after-fail');
  } finally {
    __resetFieldDefaultUniqueIndexTablesForTest();
    (globalThis as any).$choysum = { db: { dialectName: 'sqlite' } };
    try {
      // No db.execute: ensureScopeUniqueIndex should no-op and Set still succeed.
      await CovFieldDefault.Set('Widget', 'Name', 'no-exec');
    } finally {
      (globalThis as any).$choysum = originalChoysum;
      CovFieldDefault.Search = originalSearch;
      CovFieldDefault.Create = originalCreate;
      restore();
    }
  }
});

test('FieldDefault ensureScopeUniqueIndex skips CREATE when store table is missing', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-miss', ...value })) as any;
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];
  let probes = 0;
  (globalThis as any).$choysum = {
    db: {
      dialectName: 'sqlite',
      query: async () => {
        probes++;
        return JSON.stringify([]); // table missing
      },
      execute: async (ddl: string) => {
        ddls.push(ddl);
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'no-table-yet');
    expect(probes).toBe(1);
    expect(ddls.length).toBe(0);
    // Missing table must not be cached as ensured — next Set probes again.
    await CovFieldDefault.Set('Widget', 'Name', 'still-missing');
    expect(probes).toBe(2);
    expect(ddls.length).toBe(0);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});

test('FieldDefault ensureScopeUniqueIndex real-db probe skips missing fd2cov table', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-realprobe', ...value })) as any;
  const db = (globalThis as any).$choysum?.db;
  if (db == null || db.query == null || db.execute == null) {
    throw new Error(`missing db bridge: db=${db == null} query=${db?.query == null} execute=${db?.execute == null}`);
  }
  let executed = 0;
  const originalExecute = db.execute;
  db.execute = async (...args: any[]) => {
    executed++;
    return originalExecute.call(db, ...args);
  };
  try {
    const probeRaw = await db.query(
      "SELECT 1 AS ok FROM sqlite_master WHERE type = 'table' AND name = 'fd2cov_field_default' LIMIT 1",
      '[]'
    );
    const probeRows = typeof probeRaw === 'string' ? JSON.parse(probeRaw) : probeRaw;
    if (!Array.isArray(probeRows)) {
      throw new Error(`probeRows type=${typeof probeRows} rawType=${typeof probeRaw} raw=${String(probeRaw).slice(0, 200)}`);
    }
    if (probeRows.length !== 0) {
      throw new Error(`expected missing table, got rows=${JSON.stringify(probeRows)}`);
    }
    await CovFieldDefault.Set('Widget', 'Name', 'real-probe');
    if (executed !== 0) {
      throw new Error(`expected no execute, got executed=${executed}`);
    }
  } finally {
    db.execute = originalExecute;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});

test('FieldDefault ensureScopeUniqueIndex attempts CREATE when query probe is unavailable', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-noquery', ...value })) as any;
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];
  (globalThis as any).$choysum = {
    db: {
      dialectName: 'sqlite',
      query: null,
      execute: async (ddl: string) => {
        ddls.push(ddl);
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'no-query-probe');
    expect(ddls.length).toBe(1);
    expect(ddls[0]).toContain('CREATE UNIQUE INDEX');
    const before = ddls.length;
    await CovFieldDefault.Set('Widget', 'Name', 'no-query-probe-2');
    expect(ddls.length).toBe(before);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});

test('FieldDefault ensureScopeUniqueIndex treats unknown probe shape as table present', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-shape', ...value })) as any;
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];
  (globalThis as any).$choysum = {
    db: {
      dialectName: 'mysql',
      query: async () => JSON.stringify({ rows: [{ ok: 1 }] }),
      execute: async (ddl: string) => {
        ddls.push(ddl);
      },
    },
  };
  try {
    await CovFieldDefault.Set('Widget', 'Name', 'odd-shape');
    expect(ddls.length).toBe(1);
    expect(ddls[0]).toContain('coalesce(user_id');
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});

test('FieldDefault Set maps unique conflicts and rethrows other errors', async () => {
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async () => {
    throw new Error('UNIQUE constraint failed: idx');
  }) as any;
  try {
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x'), 'FIELD_DEFAULT_SCOPE_CONFLICT');
  } finally {
    CovFieldDefault.Create = originalCreate;
    CovFieldDefault.Search = originalSearch;
    restore();
  }

  const restore2 = withSavepointPassThrough();
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.Create = (async () => {
    throw new Error('NOT NULL constraint failed');
  }) as any;
  try {
    try {
      await CovFieldDefault.Set('Widget', 'Name', 'y');
      expect(false).toBe(true);
    } catch (err) {
      expect(String(err)).toContain('NOT NULL');
    }
  } finally {
    CovFieldDefault.Create = originalCreate;
    CovFieldDefault.Search = originalSearch;
    restore2();
  }

  const restore3 = withSavepointPassThrough();
  CovFieldDefault.Search = (async () => []) as any;
  CovFieldDefault.withSavepoint = (async () => {
    throw 'bare-string-error';
  }) as any;
  try {
    try {
      await CovFieldDefault.Set('Widget', 'Name', 'z');
      expect(false).toBe(true);
    } catch (err) {
      expect(err).toBe('bare-string-error');
    }
  } finally {
    restore3();
    CovFieldDefault.Search = originalSearch;
  }
});

test('FieldDefault GetEffective without company and with field filter', async () => {
  const originalSearch = CovFieldDefault.Search;
  CovFieldDefault.Search = (async () => [
    { Id: '1', Field: 'Name', UserId: null, CompanyId: null, Value: 'g' },
    { Id: '2', Field: 'Code', UserId: null, CompanyId: null, Value: 'c' },
  ]) as any;
  try {
    const out = await CovFieldDefault.GetEffective('Widget', ['Name']);
    expect(out).toEqual({ Name: 'g' });
    const all = await CovFieldDefault.GetEffective('Widget');
    expect(all.Name).toBe('g');
  } finally {
    CovFieldDefault.Search = originalSearch;
  }
});

test('FieldDefault Unset no-ops when row missing', async () => {
  const originalSearch = CovFieldDefault.Search;
  CovFieldDefault.Search = (async () => []) as any;
  try {
    await CovFieldDefault.Unset('Widget', 'Name');
  } finally {
    CovFieldDefault.Search = originalSearch;
  }
});

test('FieldDefault Get with companyId false and explicit user id', async () => {
  const originalSearch = CovFieldDefault.Search;
  let seen: any;
  CovFieldDefault.Search = (async (condition: any) => {
    seen = condition;
    return [{ Id: '1', Value: 'v' }];
  }) as any;
  try {
    const v = await CovFieldDefault.Get('Widget', 'Name', { userId: 'U-1', companyId: false });
    expect(v).toBe('v');
    expect(JSON.stringify(seen)).toContain('U-1');
  } finally {
    CovFieldDefault.Search = originalSearch;
  }
});

test('FieldDefault metadata edge branches for empty app, tableName fn, and null Search', async () => {
  __resetFieldDefaultUniqueIndexTablesForTest();
  const restore = withSavepointPassThrough();
  const originalSearch = CovFieldDefault.Search;
  const originalCreate = CovFieldDefault.Create;
  CovFieldDefault.Search = (async () => null) as any;
  CovFieldDefault.Create = (async (value: any) => ({ Id: 'FD-meta', ...value })) as any;

  const storeMeta = MetadataStorage.instance.getModelMetadata(CovFieldDefault as any) as any;
  const widgetMeta = MetadataStorage.instance.getModelMetadata(CovWidget as any) as any;
  const prevApp = storeMeta.application;
  const prevTable = storeMeta.tableName;
  const prevWidgetApp = widgetMeta.application;
  const prevFields = widgetMeta.fields;
  const originalChoysum = (globalThis as any).$choysum;

  try {
    storeMeta.application = undefined;
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x'), 'FIELD_DEFAULT_CROSS_APP_MODEL');
    storeMeta.application = '   ';
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x'), 'FIELD_DEFAULT_CROSS_APP_MODEL');
    storeMeta.application = prevApp;

    widgetMeta.application = '';
    await expectRejects(CovFieldDefault.Set('Widget', 'Name', 'x'), 'FIELD_DEFAULT_CROSS_APP_MODEL');
    widgetMeta.application = prevWidgetApp;

    await expectRejects(CovFieldDefault.Set('Widget', 'ParentId', { Id: null }), 'FIELD_DEFAULT_INVALID_VALUE');

    storeMeta.tableName = () => 'fd2cov_field_default_fn';
    (globalThis as any).$choysum = undefined;
    await CovFieldDefault.Set('Widget', 'Name', 'fn-table');

    storeMeta.tableName = '';
    __resetFieldDefaultUniqueIndexTablesForTest();
    await CovFieldDefault.Set('Widget', 'Name', 'empty-table');

    // postgresql dialect alias + fields=[] + Search null (rows || [])
    storeMeta.tableName = 'fd2cov_field_default';
    __resetFieldDefaultUniqueIndexTablesForTest();
    const ddls: string[] = [];
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'postgresql',
        query: async () => JSON.stringify([{ ok: 1 }]),
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await CovFieldDefault.Set('Widget', 'Name', 'pg-alias');
    expect(ddls.some(d => d.includes('NULLS NOT DISTINCT'))).toBe(true);

    const out = await CovFieldDefault.GetEffective('Widget', []);
    expect(out).toEqual({});

    // targetMeta.fields without keys() falls back to []
    widgetMeta.fields = { get: prevFields.get.bind(prevFields) };
    const out2 = await CovFieldDefault.GetEffective('Widget');
    expect(out2).toEqual({});
  } finally {
    storeMeta.application = prevApp;
    storeMeta.tableName = prevTable;
    widgetMeta.application = prevWidgetApp;
    widgetMeta.fields = prevFields;
    (globalThis as any).$choysum = originalChoysum;
    CovFieldDefault.Search = originalSearch;
    CovFieldDefault.Create = originalCreate;
    restore();
    __resetFieldDefaultUniqueIndexTablesForTest();
  }
});
