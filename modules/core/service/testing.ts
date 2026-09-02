// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildRelationAliasCandidates as buildOrmRelationAliasCandidates } from './orm/relation/relation_alias';
import { RepositoryFactory } from './orm/repository/repository_factory';

type TestRepositoryCtor = Parameters<typeof RepositoryFactory.getRepository>[0];
type TestRepository = ReturnType<typeof RepositoryFactory.getRepository>;

export function getTestRepository(modelCtor: TestRepositoryCtor): TestRepository {
  return RepositoryFactory.getRepository(modelCtor);
}

export function setTestRepository(modelCtor: TestRepositoryCtor, repository: TestRepository): void {
  RepositoryFactory.setRepository(modelCtor, repository);
}

export const buildRelationAliasCandidates = buildOrmRelationAliasCandidates;

export {
  invalidateAuthzCachesForUsers,
  withPermissionGraphBypass,
} from './testing/authz';
