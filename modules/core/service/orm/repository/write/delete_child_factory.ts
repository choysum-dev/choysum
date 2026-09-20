// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { UntypedQueryCondition, DeleteResult, SelectResult } from '../types/engine';

export type RepositoryDeleteChild = {
  softDeleteEnabled: () => boolean;
  delete: (condition: UntypedQueryCondition) => Promise<DeleteResult[]>;
  hardDelete: (condition: UntypedQueryCondition) => Promise<DeleteResult[]>;
  count: (condition: UntypedQueryCondition) => Promise<number>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  update: (vals: SelectResult, condition: UntypedQueryCondition) => Promise<unknown>;
};

type RepositoryDeleteChildSource = {
  softDeleteEnabled: () => boolean;
  delete: (condition: UntypedQueryCondition) => Promise<DeleteResult[]>;
  hardDelete: (condition: UntypedQueryCondition) => Promise<DeleteResult[]>;
  count: (condition: UntypedQueryCondition) => Promise<number>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  update: (vals: SelectResult, condition: UntypedQueryCondition) => Promise<unknown>;
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
