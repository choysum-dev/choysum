// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumConnection } from './connection';
import { DbErrCode } from '../error';

type DbMock = {
  savepoint: (name: string) => Promise<string>;
  releaseSavepoint: (name: string) => Promise<void>;
  rollbackToSavepoint: (name: string) => Promise<void>;
  query: (sql: string, params: string) => Promise<string>;
  execute: (sql: string, params: string) => Promise<string>;
};

function setupChoysumDb(overrides?: Partial<DbMock>) {
  const previous = (globalThis as any).$choysum;
  const calls = {
    savepoint: [] as string[],
    release: [] as string[],
    rollback: [] as string[],
    query: [] as Array<{ sql: string; params: string }>,
    execute: [] as Array<{ sql: string; params: string }>,
  };

  let seq = 0;
  const db: DbMock = {
    savepoint: async (name: string) => {
      calls.savepoint.push(name);
      return name;
    },
    releaseSavepoint: async (name: string) => {
      calls.release.push(name);
    },
    rollbackToSavepoint: async (name: string) => {
      calls.rollback.push(name);
    },
    query: async (sql: string, params: string) => {
      calls.query.push({ sql, params });
      return JSON.stringify([{ Id: 'q1', Name: 'alpha' }]);
    },
    execute: async (sql: string, params: string) => {
      calls.execute.push({ sql, params });
      return JSON.stringify({ RowsAffected: 2, LastInsertId: 'ins_1' });
    },
  };

  (globalThis as any).$choysum = {
    ...((globalThis as any).$choysum || {}),
    xid: {
      New: () => {
        seq += 1;
        return `x${seq}`;
      },
    },
    db: {
      ...db,
      ...(overrides || {}),
    },
  };

  return {
    calls,
    restore: () => {
      if (previous === undefined) {
        delete (globalThis as any).$choysum;
        return;
      }
      (globalThis as any).$choysum = previous;
    },
  };
}

function expectDbError(error: any, code: string) {
  expect(error?.domain).toBe('core.db');
  expect(error?.code).toBe(code);
}

test('connection begin, commit and rollback use savepoints in stack order', async () => {
  const { calls, restore } = setupChoysumDb();
  const conn = new ChoysumConnection();
  try {
    await conn.beginTransaction();
    await conn.beginTransaction();
    await conn.commitTransaction();
    await conn.rollbackTransaction();

    expect(calls.savepoint).toEqual(['tx_x1', 'tx_x2']);
    expect(calls.release).toEqual(['tx_x2']);
    expect(calls.rollback).toEqual(['tx_x1']);
  } finally {
    restore();
  }
});

test('connection commit and rollback without begin surface wrapped not-started errors', async () => {
  const { restore } = setupChoysumDb();
  const conn = new ChoysumConnection();
  try {
    let commitErr: any;
    try {
      await conn.commitTransaction();
    } catch (e) {
      commitErr = e;
    }
    expectDbError(commitErr, DbErrCode.TRANSACTION_COMMIT_FAILED);
    expect(commitErr?.cause?.code).toBe(DbErrCode.TRANSACTION_NOT_STARTED);

    let rollbackErr: any;
    try {
      await conn.rollbackTransaction();
    } catch (e) {
      rollbackErr = e;
    }
    expectDbError(rollbackErr, DbErrCode.TRANSACTION_ROLLBACK_FAILED);
    expect(rollbackErr?.cause?.code).toBe(DbErrCode.TRANSACTION_NOT_STARTED);
  } finally {
    restore();
  }
});

test('connection executeQuery uses query for result-set sql and execute for mutation sql', async () => {
  const { calls, restore } = setupChoysumDb();
  const conn = new ChoysumConnection();
  try {
    const queryResult = await conn.executeQuery<any>({
      sql: 'select * from auth_user where id = ?',
      parameters: ['u1'],
      query: { kind: 'SelectQueryNode' },
    } as any);

    expect(queryResult.rows).toEqual([{ Id: 'q1', Name: 'alpha' }]);
    expect(calls.query).toEqual([{ sql: 'select * from auth_user where id = ?', params: '["u1"]' }]);

    const execResult = await conn.executeQuery<any>({
      sql: 'update auth_user set name = ? where id = ?',
      parameters: ['new', 'u1'],
      query: { kind: 'UpdateQueryNode' },
    } as any);

    expect(Number(execResult.numChangedRows as any)).toBe(2);
    expect(Number(execResult.numAffectedRows as any)).toBe(2);
    expect(execResult.insertId as any).toBe('ins_1');
    expect(execResult.rows).toEqual([]);
    expect(calls.execute).toEqual([{ sql: 'update auth_user set name = ? where id = ?', params: '["new","u1"]' }]);
  } finally {
    restore();
  }
});

test('connection executeQuery wraps select failure as execution error', async () => {
  const { restore } = setupChoysumDb({
    query: async () => {
      throw new Error('query down');
    },
  });
  const conn = new ChoysumConnection();
  try {
    let err: any;
    try {
      await conn.executeQuery<any>({
        sql: 'select 1',
        parameters: [],
        query: { kind: 'SelectQueryNode' },
      } as any);
    } catch (e) {
      err = e;
    }

    expectDbError(err, DbErrCode.EXECUTION_FAILED);
    expect(err?.cause?.code).toBe(DbErrCode.QUERY_FAILED);
  } finally {
    restore();
  }
});

test('connection streamQuery throws not-supported database error', async () => {
  const { restore } = setupChoysumDb();
  const conn = new ChoysumConnection();
  const iterator = conn.streamQuery<any>({} as any, 100);
  try {
    let err: any;
    try {
      await iterator.next();
    } catch (e) {
      err = e;
    }

    expectDbError(err, DbErrCode.STREAMING_NOT_SUPPORTED);
  } finally {
    restore();
  }
});

test('connection savepoint wrappers map db failures to dedicated error codes', async () => {
  const { restore } = setupChoysumDb({
    savepoint: async () => {
      throw new Error('savepoint failed');
    },
    rollbackToSavepoint: async () => {
      throw new Error('rollback failed');
    },
    releaseSavepoint: async () => {
      throw new Error('release failed');
    },
  });
  const conn = new ChoysumConnection();
  try {
    let saveErr: any;
    try {
      await conn.savepoint('sp_1');
    } catch (e) {
      saveErr = e;
    }
    expectDbError(saveErr, DbErrCode.SAVEPOINT_CREATE_FAILED);

    let rollbackErr: any;
    try {
      await conn.rollbackToSavepoint('sp_1');
    } catch (e) {
      rollbackErr = e;
    }
    expectDbError(rollbackErr, DbErrCode.SAVEPOINT_ROLLBACK_FAILED);

    let releaseErr: any;
    try {
      await conn.releaseSavepoint('sp_1');
    } catch (e) {
      releaseErr = e;
    }
    expectDbError(releaseErr, DbErrCode.SAVEPOINT_RELEASE_FAILED);
  } finally {
    restore();
  }
});

test('connection withSavepoint returns callback result even when release fails', async () => {
  const { calls, restore } = setupChoysumDb({
    releaseSavepoint: async () => {
      throw new Error('release failed');
    },
  });
  const conn = new ChoysumConnection();

  const warnCalls: any[] = [];
  const oldWarn = console.warn;
  (console as any).warn = (...args: any[]) => {
    warnCalls.push(args);
  };

  try {
    let result: any;
    try {
      result = await conn.withSavepoint(async () => 'ok', 'sp_manual');
    } finally {
      (console as any).warn = oldWarn;
    }

    expect(result).toBe('ok');
    expect(calls.savepoint).toEqual(['sp_manual']);
    expect(calls.release).toEqual([]);
    expect(warnCalls.length).toBe(1);
  } finally {
    restore();
  }
});

test('connection withSavepoint wraps operation failure and rollback failure paths', async () => {
  const conn1 = new ChoysumConnection();
  const env1 = setupChoysumDb();
  try {
    let opErr: any;
    try {
      await conn1.withSavepoint(async () => {
        throw new Error('boom-op');
      }, 'sp_op');
    } catch (e) {
      opErr = e;
    }
    expectDbError(opErr, DbErrCode.SAVEPOINT_OPERATION_FAILED);
    expect(String(opErr?.message || '')).toContain('Operation failed in savepoint sp_op');
  } finally {
    env1.restore();
  }

  const env2 = setupChoysumDb({
    rollbackToSavepoint: async () => {
      throw new Error('boom-rollback');
    },
  });
  const conn2 = new ChoysumConnection();
  try {
    let rollbackErr: any;
    try {
      await conn2.withSavepoint(async () => {
        throw new Error('boom-op');
      }, 'sp_rb');
    } catch (e) {
      rollbackErr = e;
    }
    expectDbError(rollbackErr, DbErrCode.SAVEPOINT_OPERATION_FAILED);
    expect(String(rollbackErr?.message || '')).toContain('also failed');
  } finally {
    env2.restore();
  }
});

test('connection withSavepoint maps savepoint creation failure to dedicated code', async () => {
  const { restore } = setupChoysumDb({
    savepoint: async () => {
      throw new Error('cannot create savepoint');
    },
  });
  const conn = new ChoysumConnection();
  try {
    let err: any;
    try {
      await conn.withSavepoint(async () => 'never');
    } catch (e) {
      err = e;
    }

    expectDbError(err, DbErrCode.SAVEPOINT_CREATE_FAILED);
  } finally {
    restore();
  }
});
