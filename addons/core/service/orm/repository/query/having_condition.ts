// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, RepositoryConditionConverterLike } from '../types';
import { asObjectRecord } from '../../../../utils/object';

type RepositoryHavingExpressionBuilder = {
  (lhs: unknown, op: unknown, rhs: unknown): unknown;
  and: (parts: unknown[]) => unknown;
  or: (parts: unknown[]) => unknown;
  ref: (name: string) => unknown;
};

type RepositoryHavingConditionDeps = {
  convertCondition: RepositoryConditionConverterLike<BaseQueryCondition>;
  selfTable?: string;
};

export function convertRepositoryHavingCondition(
  deps: RepositoryHavingConditionDeps,
  eb: unknown,
  condition: BaseQueryCondition,
  knownAliases: Set<string>
): unknown {
  const builder = eb as RepositoryHavingExpressionBuilder;

  const asPredicate = (current: BaseQueryCondition): unknown => {
    if (Array.isArray(current)) {
      if (current.length === 0) return builder.and([]);
      if (current.length !== 3) {
        throw new Error(`invalid condition tuple length in HAVING: ${current.length}`);
      }

      let [lhs, op, rhs] = current as [string, unknown, unknown];
      const lhsStr = String(lhs);
      const rawLowerOp = String(op || '').toLowerCase();
      if (rhs == null) {
        if (rawLowerOp === '=' || rawLowerOp === '==') op = 'is';
        else if (rawLowerOp === '!=' || rawLowerOp === '<>') op = 'is not';
      }

      if (knownAliases.has(lhsStr)) {
        return builder(builder.ref(lhsStr), op, rhs);
      }

      return deps.convertCondition(builder, current, deps.selfTable);
    }

    const envelope = asObjectRecord(current);
    const andConditions = envelope?.And;
    if (Array.isArray(andConditions)) {
      const parts = andConditions as BaseQueryCondition[];
      return builder.and(parts.map(part => asPredicate(part)));
    }

    const orConditions = envelope?.Or;
    if (Array.isArray(orConditions)) {
      const parts = orConditions as BaseQueryCondition[];
      return builder.or(parts.map(part => asPredicate(part)));
    }

    return builder.and([]);
  };

  return asPredicate(condition);
}
