// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { jsonArrayFrom as pgArr, jsonObjectFrom as pgObj } from 'kysely/helpers/postgres';
import { jsonArrayFrom as myArr, jsonObjectFrom as myObj } from 'kysely/helpers/mysql';
import { jsonArrayFrom as slArr, jsonObjectFrom as slObj } from 'kysely/helpers/sqlite';
import { jsonArrayFrom as msArr, jsonObjectFrom as msObj } from 'kysely/helpers/mssql';
import type { DialectName } from '../repository_dialect';

/**
 * Returns the dialect-specific JSON projection helpers used by repository projection builders.
 */
export function getJsonHelpers(dialect: DialectName) {
  switch (dialect) {
    case 'postgres':
      return { jsonArrayFrom: pgArr, jsonObjectFrom: pgObj };
    case 'mysql':
      return { jsonArrayFrom: myArr, jsonObjectFrom: myObj };
    case 'sqlite':
      return { jsonArrayFrom: slArr, jsonObjectFrom: slObj };
    case 'mssql':
      return { jsonArrayFrom: msArr, jsonObjectFrom: msObj };
    default:
      return { jsonArrayFrom: pgArr, jsonObjectFrom: pgObj };
  }
}
