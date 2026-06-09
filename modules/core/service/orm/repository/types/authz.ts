// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition } from './query';

export type RecordRuleOp = 'read' | 'write' | 'create' | 'delete';
export type ConditionExpr = BaseQueryCondition;

export type ConditionEnvelope =
  | {
      kind: 'true';
      reason?: string;
    }
  | {
      kind: 'false';
      reason?: string;
    }
  | {
      kind: 'expr';
      expr: ConditionExpr;
      reason?: string;
    };
