// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { PostgresQueryCompiler, MysqlQueryCompiler, MssqlQueryCompiler, SqliteQueryCompiler } from "kysely";

export class ChoysumPostgresQueryCompiler extends PostgresQueryCompiler {
  // protected override getCurrentParameterPlaceholder(): string {
  //   return "?";
  // }
}

export class ChoysumMysqlQueryCompiler extends MysqlQueryCompiler {
  // protected override getCurrentParameterPlaceholder(): string {
  //   return "?";
  // }
}

export class ChoysumMssqlQueryCompiler extends MssqlQueryCompiler {
  // protected override getCurrentParameterPlaceholder(): string {
  //   return "?";
  // }
}

export class ChoysumSqliteQueryCompiler extends SqliteQueryCompiler {
  // protected override getCurrentParameterPlaceholder(): string {
  //   return "?";
  // }
}
