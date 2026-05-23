// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MysqlAdapter, MysqlIntrospector, PostgresAdapter, PostgresIntrospector, SqliteAdapter, SqliteIntrospector } from 'kysely';
import { ChoysumMysqlQueryCompiler, ChoysumPostgresQueryCompiler, ChoysumSqliteQueryCompiler } from './compiler';
import { ChoysumDialect } from './dialect';
import { ChoysumDriver } from './driver';

function withDialectName<T>(dialectName: unknown, fn: () => T): T {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];

  (globalThis as Record<string, unknown>)[key] = {
    ...(previous as Record<string, unknown>),
    db: {
      ...((previous as any)?.db || {}),
      dialectName,
    },
  };

  try {
    return fn();
  } finally {
    if (hadOwn) {
      (globalThis as Record<string, unknown>)[key] = previous;
    } else {
      delete (globalThis as Record<string, unknown>)[key];
    }
  }
}

test('dialect creates adapter/introspector/compiler for postgres/mysql/sqlite', () => {
  withDialectName('postgres', () => {
    const dialect = new ChoysumDialect();
    expect(dialect.createAdapter() instanceof PostgresAdapter).toBe(true);
    expect(dialect.createIntrospector({} as any) instanceof PostgresIntrospector).toBe(true);
    expect(dialect.createQueryCompiler() instanceof ChoysumPostgresQueryCompiler).toBe(true);
  });

  withDialectName('mysql', () => {
    const dialect = new ChoysumDialect();
    expect(dialect.createAdapter() instanceof MysqlAdapter).toBe(true);
    expect(dialect.createIntrospector({} as any) instanceof MysqlIntrospector).toBe(true);
    expect(dialect.createQueryCompiler() instanceof ChoysumMysqlQueryCompiler).toBe(true);
  });

  withDialectName('sqlite', () => {
    const dialect = new ChoysumDialect();
    expect(dialect.createAdapter() instanceof SqliteAdapter).toBe(true);
    expect(dialect.createIntrospector({} as any) instanceof SqliteIntrospector).toBe(true);
    expect(dialect.createQueryCompiler() instanceof ChoysumSqliteQueryCompiler).toBe(true);
  });
});

test('dialect createDriver returns ChoysumDriver', () => {
  withDialectName('sqlite', () => {
    const dialect = new ChoysumDialect();
    expect(dialect.createDriver() instanceof ChoysumDriver).toBe(true);
  });
});

test('dialect throws when dialectName is missing or unsupported', () => {
  let missingMessage = '';
  withDialectName(undefined, () => {
    const dialect = new ChoysumDialect();
    try {
      dialect.createAdapter();
    } catch (error) {
      missingMessage = String((error as Error)?.message || error);
    }
  });
  expect(missingMessage).toContain('Dialect not set');

  withDialectName('mssql', () => {
    const dialect = new ChoysumDialect();

    let adapterMessage = '';
    try {
      dialect.createAdapter();
    } catch (error) {
      adapterMessage = String((error as Error)?.message || error);
    }
    expect(adapterMessage).toContain('Unsupported dialect');

    let introspectorMessage = '';
    try {
      dialect.createIntrospector({} as any);
    } catch (error) {
      introspectorMessage = String((error as Error)?.message || error);
    }
    expect(introspectorMessage).toContain('Unsupported dialect');

    let compilerMessage = '';
    try {
      dialect.createQueryCompiler();
    } catch (error) {
      compilerMessage = String((error as Error)?.message || error);
    }
    expect(compilerMessage).toContain('Unsupported dialect');
  });
});
