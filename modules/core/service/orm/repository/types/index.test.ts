// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as typesBarrel from './index';

/** Runtime helpers allowed on the SSOT types barrel (selection / condition). */
const TYPES_BARREL_RUNTIME_ALLOW = ['condition', 'fields', 'projectToSelection'] as const;

/** Engine ducks / Kysely / deps that must stay on ./engine (SF-1 / SF-6). */
const TYPES_BARREL_ENGINE_RUNTIME_DENY = [
  'ExpressionBuilder',
  'ExpressionWrapper',
  'SelectQueryBuilder',
  'Compilable',
  'InsertResult',
  'UpdateResult',
  'DeleteResult',
  'SimplifyResult',
  'RepositoryAliasableLike',
  'RepositoryQueryLike',
  'RepositoryExecute',
] as const;

test('orm/repository/types barrel runtime surface stays limited to selection/condition helpers (SF-6)', () => {
  expect(Object.keys(typesBarrel).sort()).toEqual([...TYPES_BARREL_RUNTIME_ALLOW].sort());
});

test('orm/repository/types barrel rejects engine runtime symbols (SF-6 deny list)', () => {
  const keys = new Set(Object.keys(typesBarrel));
  for (const name of TYPES_BARREL_ENGINE_RUNTIME_DENY) {
    expect(keys.has(name)).toBe(false);
  }
});
