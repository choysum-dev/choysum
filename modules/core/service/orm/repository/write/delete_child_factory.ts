// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, DeleteResult, Entity } from '../types';

export type RepositoryDeleteChild = {
  softDeleteEnabled: () => boolean;
  delete: (condition: BaseQueryCondition) => Promise<DeleteResult[]>;
  hardDelete: (condition: BaseQueryCondition) => Promise<DeleteResult[]>;
  count: (condition: BaseQueryCondition) => Promise<number>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  update: (vals: Entity, condition: BaseQueryCondition) => Promise<unknown>;
};

type RepositoryDeleteChildSource = {
  softDeleteEnabled: () => boolean;
  delete: (condition: BaseQueryCondition) => Promise<DeleteResult[]>;
  hardDelete: (condition: BaseQueryCondition) => Promise<DeleteResult[]>;
  count: (condition: BaseQueryCondition) => Promise<number>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  update: (vals: Entity, condition: BaseQueryCondition) => Promise<unknown>;
};

export function createRepositoryDeleteChild(source: RepositoryDeleteChildSource): RepositoryDeleteChild {
  return {
    softDeleteEnabled: () => source.softDeleteEnabled(),
    delete: condition => source.delete(condition),
    hardDelete: condition => source.hardDelete(condition),
    count: condition => source.count(condition),
    withFieldRuleBypass: fn => source.withFieldRuleBypass(fn),
    update: (vals, condition) => source.update(vals, condition),
  };
}
