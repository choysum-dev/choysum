// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Dialect, Kysely, PostgresAdapter, MysqlAdapter, SqliteAdapter, PostgresIntrospector, MysqlIntrospector, SqliteIntrospector } from 'kysely';
import { ChoysumPostgresQueryCompiler, ChoysumMysqlQueryCompiler, ChoysumSqliteQueryCompiler } from './compiler';
import { ChoysumDriver } from './driver';

export class ChoysumDialect implements Dialect {
  #dialectName(): 'postgres' | 'mysql' | 'sqlite' | 'mssql' {
    if ($choysum.db.dialectName === undefined) {
      throw new Error('Dialect not set');
    }
    return $choysum.db.dialectName;
  }

  createAdapter(): PostgresAdapter | MysqlAdapter | SqliteAdapter {
    switch (this.#dialectName()) {
      case 'postgres':
        return new PostgresAdapter();
      case 'mysql':
        return new MysqlAdapter();
      case 'sqlite':
        return new SqliteAdapter();
      default:
        throw new Error('Unsupported dialect');
    }
  }

  createDriver(): ChoysumDriver {
    return new ChoysumDriver();
  }

  createIntrospector(db: Kysely<unknown>): PostgresIntrospector | MysqlIntrospector | SqliteIntrospector {
    switch (this.#dialectName()) {
      case 'postgres':
        return new PostgresIntrospector(db);
      case 'mysql':
        return new MysqlIntrospector(db);
      case 'sqlite':
        return new SqliteIntrospector(db);
      default:
        throw new Error('Unsupported dialect');
    }
  }

  createQueryCompiler(): ChoysumPostgresQueryCompiler | ChoysumMysqlQueryCompiler | ChoysumSqliteQueryCompiler {
    switch (this.#dialectName()) {
      case 'postgres':
        return new ChoysumPostgresQueryCompiler();
      case 'mysql':
        return new ChoysumMysqlQueryCompiler();
      case 'sqlite':
        return new ChoysumSqliteQueryCompiler();
      default:
        throw new Error('Unsupported dialect');
    }
  }
}
