// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as repositoryBarrel from './index';

/** Former repository barrel keys that must stay off the public surface (SF-1 / SF-6). */
const REPOSITORY_ENGINE_RUNTIME_DENY = [
  'ExpressionBuilder',
  'SelectQueryBuilder',
  'convertCondition',
  'createRepositorySearchDeps',
  'executeRepositorySearch',
  'makeSelectCtx',
  'fields',
  'projectToSelection',
  'condition',
] as const;

test('orm/repository barrel is runtime-only (SF-1 / SF-D)', () => {
  expect(Object.keys(repositoryBarrel).sort()).toEqual(['Repository', 'RepositoryFactory', 'db']);
});

test('orm/repository barrel rejects engine runtime symbols (SF-6 deny list)', () => {
  const keys = new Set(Object.keys(repositoryBarrel));
  for (const name of REPOSITORY_ENGINE_RUNTIME_DENY) {
    expect(keys.has(name)).toBe(false);
  }
});
