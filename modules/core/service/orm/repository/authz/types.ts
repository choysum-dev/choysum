// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryPermissionDeniedFn = (code: string, message: string, metadata?: Record<string, string>) => Error;

export type RepositoryAuthzDecisionLayer = 'method_access' | 'record_rule' | 'company_filter' | 'field_rule' | 'unknown';

export type RepositoryAuthzDecision = 'allow' | 'deny';

export type RepositoryAuthzDecisionSummary = ObjectRecord & {
  layer?: RepositoryAuthzDecisionLayer | string;
  decision?: RepositoryAuthzDecision | string;
  basis?: string;
  fullMethod?: string;
  method?: string;
  model?: string;
  op?: string;
  userId?: string;
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
  companyMode?: string;
  recordRuleMode?: string;
  fieldRuleMode?: string;
  /** Machine-readable decision reason (promoted from metadata.reason when missing). */
  reason?: string;
  /** Role* rule row Ids that participated in the decision. */
  hitRuleIds?: string[];
  message?: string;
  metadata?: Record<string, string>;
};

export type RepositoryEmitAuthzDecisionSummary = (summary: RepositoryAuthzDecisionSummary) => void;
