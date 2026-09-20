// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { UntypedQueryCondition } from './query';

export type RecordRuleOp = 'read' | 'write' | 'create' | 'delete';

/** Shared observability fields on ACL/RR/FR decision envelopes. */
export type AuthzDecisionDiagnostics = {
  reason?: string;
  /** Role* rule row Ids that participated in the decision (empty when none). */
  hitRuleIds?: string[];
};

export type ConditionEnvelope =
  | ({
      kind: 'true';
    } & AuthzDecisionDiagnostics)
  | ({
      kind: 'false';
    } & AuthzDecisionDiagnostics)
  | ({
      kind: 'expr';
      expr: UntypedQueryCondition;
    } & AuthzDecisionDiagnostics);
