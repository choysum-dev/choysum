// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DeleteResult, InsertResult, UpdateResult } from 'kysely';
import { ChoysumDatabase } from './database';

function withPatchedChoysum<T>(fn: () => Promise<T>): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  let seq = 0;

  (globalThis as Record<string, unknown>)[key] = {
    ...(previous as Record<string, unknown>),
    xid: {
      New: () => {
        seq += 1;
        return `xid_${seq}`;
      },
    },
  };

  return fn().finally(() => {
    if (hadOwn) {
      (globalThis as Record<string, unknown>)[key] = previous;
    } else {
      delete (globalThis as Record<string, unknown>)[key];
    }
  });
}

function createCompilable(kind: string, node: Record<string, any> = {}) {
  return {
    compile() {
      return {
        query: {
          kind,
          ...node,
        },
      };
    },
  } as any;
}

function createDatabaseHarness(options?: { supportsReturning?: boolean; supportsOutput?: boolean; executeQueryResult?: any }) {
  const calls = {
    executeQuery: [] as Array<{ compiledQuery: any; xid: string }>,
    savepoint: [] as string[],
    rollback: [] as string[],
    release: [] as string[],
    withSavepoint: [] as string[],
    provideConnection: 0,
  };

  const connection = {
    async savepoint(name: string) {
      calls.savepoint.push(name);
      return `saved_${name}`;
    },
    async rollbackToSavepoint(name: string) {
      calls.rollback.push(name);
    },
    async releaseSavepoint(name: string) {
      calls.release.push(name);
    },
    async withSavepoint<T>(callback: () => Promise<T>, name: string): Promise<T> {
      calls.withSavepoint.push(name);
      return await callback();
    },
  };

  const executor = {
    adapter: {
      supportsReturning: options?.supportsReturning ?? false,
      supportsOutput: options?.supportsOutput ?? false,
    },
    async executeQuery(compiledQuery: any, xid: string) {
      calls.executeQuery.push({ compiledQuery, xid });
      return (
        options?.executeQueryResult || {
          rows: [],
          insertId: undefined,
          numAffectedRows: undefined,
          numChangedRows: undefined,
        }
      );
    },
    async provideConnection<T>(callback: (conn: typeof connection) => Promise<T>): Promise<T> {
      calls.provideConnection += 1;
      return await callback(connection);
    },
  };

  const db = Object.create(ChoysumDatabase.prototype) as ChoysumDatabase<any>;
  (db as any).getExecutor = () => executor;

  return { db: db as any, calls };
}

function createFakeDialectForCtor() {
  return {
    createAdapter() {
      return {
        supportsReturning: false,
        supportsOutput: false,
      } as any;
    },
    createDriver() {
      return {
        async init() {},
        async acquireConnection() {
          return {
            async executeQuery() {
              return { rows: [] };
            },
            async *streamQuery() {},
          } as any;
        },
        async beginTransaction() {},
        async commitTransaction() {},
        async rollbackTransaction() {},
        async releaseConnection() {},
        async destroy() {},
      } as any;
    },
    createQueryCompiler() {
      return {
        compileQuery() {
          return {
            sql: 'select 1',
            parameters: [],
            query: { kind: 'SelectQueryNode' },
          } as any;
        },
      } as any;
    },
  } as any;
}

test('database getExecutor returns constructor-initialized executor', async () => {
  const db = new ChoysumDatabase<any>({
    dialect: createFakeDialectForCtor(),
  } as any);

  try {
    const executor = db.getExecutor();
    expect(executor).toBeDefined();
    expect(typeof (executor as any).executeQuery).toBe('function');
  } finally {
    await db.destroy();
  }
});

test('database execute handles select and unknown-query paths while forwarding xid to executor', async () => {
  await withPatchedChoysum(async () => {
    const selectHarness = createDatabaseHarness({
      executeQueryResult: {
        rows: [{ Id: 's1', Name: 'demo' }],
      },
    });

    const selectResult = await ChoysumDatabase.prototype.execute.call(selectHarness.db, createCompilable('SelectQueryNode'));
    expect(selectResult).toEqual([{ Id: 's1', Name: 'demo' }]);
    expect(selectHarness.calls.executeQuery[0]?.xid).toBe('xid_1');

    const fallbackHarness = createDatabaseHarness({
      executeQueryResult: {
        rows: [{ Id: 'f1' }],
      },
    });

    const fallbackResult = await ChoysumDatabase.prototype.execute.call(fallbackHarness.db, createCompilable('SomeOtherNode'));
    expect(fallbackResult).toEqual([{ Id: 'f1' }]);
    expect(fallbackHarness.calls.executeQuery[0]?.xid).toBe('xid_2');
  });
});

test('database execute handles insert/delete/update returning and fallback result wrappers', async () => {
  await withPatchedChoysum(async () => {
    const insertReturningHarness = createDatabaseHarness({
      supportsReturning: true,
      executeQueryResult: {
        rows: [{ Id: 'inserted' }],
      },
    });
    const insertReturning = await ChoysumDatabase.prototype.execute.call(insertReturningHarness.db, createCompilable('InsertQueryNode', { returning: ['Id'] }));
    expect(insertReturning).toEqual([{ Id: 'inserted' }]);

    const insertFallbackHarness = createDatabaseHarness({
      executeQueryResult: {
        rows: [],
        insertId: 'ins-1',
        numAffectedRows: undefined,
      },
    });
    const insertFallback = await ChoysumDatabase.prototype.execute.call(insertFallbackHarness.db, createCompilable('InsertQueryNode'));
    expect(insertFallback[0] instanceof InsertResult).toBe(true);
    expect((insertFallback[0] as any).insertId).toBe('ins-1');
    expect((insertFallback[0] as any).numInsertedOrUpdatedRows).toBe(BigInt(0));

    const deleteReturningHarness = createDatabaseHarness({
      supportsOutput: true,
      executeQueryResult: {
        rows: [{ Id: 'deleted' }],
      },
    });
    const deleteReturning = await ChoysumDatabase.prototype.execute.call(deleteReturningHarness.db, createCompilable('DeleteQueryNode', { output: ['Id'] }));
    expect(deleteReturning).toEqual([{ Id: 'deleted' }]);

    const deleteFallbackHarness = createDatabaseHarness({
      executeQueryResult: {
        rows: [],
        numAffectedRows: BigInt(2),
      },
    });
    const deleteFallback = await ChoysumDatabase.prototype.execute.call(deleteFallbackHarness.db, createCompilable('DeleteQueryNode'));
    expect(deleteFallback[0] instanceof DeleteResult).toBe(true);
    expect((deleteFallback[0] as any).numDeletedRows).toBe(BigInt(2));

    const updateReturningHarness = createDatabaseHarness({
      supportsReturning: true,
      executeQueryResult: {
        rows: [{ Id: 'updated' }],
      },
    });
    const updateReturning = await ChoysumDatabase.prototype.execute.call(updateReturningHarness.db, createCompilable('UpdateQueryNode', { returning: ['Id'] }));
    expect(updateReturning).toEqual([{ Id: 'updated' }]);

    const updateFallbackHarness = createDatabaseHarness({
      executeQueryResult: {
        rows: [],
        numAffectedRows: BigInt(3),
        numChangedRows: BigInt(1),
      },
    });
    const updateFallback = await ChoysumDatabase.prototype.execute.call(updateFallbackHarness.db, createCompilable('UpdateQueryNode'));
    expect(updateFallback[0] instanceof UpdateResult).toBe(true);
    expect((updateFallback[0] as any).numUpdatedRows).toBe(BigInt(3));
    expect((updateFallback[0] as any).numChangedRows).toBe(BigInt(1));
  });
});

test('database savepoint wrappers delegate to provideConnection and connection methods', async () => {
  await withPatchedChoysum(async () => {
    const harness = createDatabaseHarness();

    const savepointName = await ChoysumDatabase.prototype.savepoint.call(harness.db, 'sp_manual');
    expect(savepointName).toBe('saved_sp_manual');

    await ChoysumDatabase.prototype.rollbackToSavepoint.call(harness.db, 'sp_manual');
    await ChoysumDatabase.prototype.releaseSavepoint.call(harness.db, 'sp_manual');

    expect(harness.calls.savepoint).toEqual(['sp_manual']);
    expect(harness.calls.rollback).toEqual(['sp_manual']);
    expect(harness.calls.release).toEqual(['sp_manual']);
    expect(harness.calls.provideConnection).toBe(3);
  });
});

test('database withSavepoint uses explicit name and auto-generated name branches', async () => {
  await withPatchedChoysum(async () => {
    const explicitHarness = createDatabaseHarness();
    const explicitResult = await ChoysumDatabase.prototype.withSavepoint.call(explicitHarness.db, async () => 'ok', 'sp_explicit');
    expect(explicitResult).toBe('ok');
    expect(explicitHarness.calls.withSavepoint).toEqual(['sp_explicit']);

    const autoHarness = createDatabaseHarness();
    const autoResult = await ChoysumDatabase.prototype.withSavepoint.call(autoHarness.db, async () => 'auto-ok');
    expect(autoResult).toBe('auto-ok');
    expect(autoHarness.calls.withSavepoint).toEqual(['sp_xid_1']);
  });
});
