// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { jsonArrayFrom as msArr, jsonObjectFrom as msObj } from 'kysely/helpers/mssql';
import { jsonArrayFrom as myArr, jsonObjectFrom as myObj } from 'kysely/helpers/mysql';
import { jsonArrayFrom as pgArr, jsonObjectFrom as pgObj } from 'kysely/helpers/postgres';
import { jsonArrayFrom as slArr, jsonObjectFrom as slObj } from 'kysely/helpers/sqlite';
import { getJsonHelpers } from '../json_helpers';

test('getJsonHelpers returns postgres helpers for postgres and fallback', () => {
  const pg = getJsonHelpers('postgres');
  const fallback = getJsonHelpers('unknown' as any);

  expect(pg.jsonArrayFrom).toBe(pgArr);
  expect(pg.jsonObjectFrom).toBe(pgObj);
  expect(fallback.jsonArrayFrom).toBe(pgArr);
  expect(fallback.jsonObjectFrom).toBe(pgObj);
});

test('getJsonHelpers returns mysql/sqlite/mssql helpers for each dialect', () => {
  const mysql = getJsonHelpers('mysql');
  const sqlite = getJsonHelpers('sqlite');
  const mssql = getJsonHelpers('mssql');

  expect(mysql.jsonArrayFrom).toBe(myArr);
  expect(mysql.jsonObjectFrom).toBe(myObj);

  expect(sqlite.jsonArrayFrom).toBe(slArr);
  expect(sqlite.jsonObjectFrom).toBe(slObj);

  expect(mssql.jsonArrayFrom).toBe(msArr);
  expect(mssql.jsonObjectFrom).toBe(msObj);
});
