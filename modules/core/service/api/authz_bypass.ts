// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxAndReq, getOrInitReqServiceState, withBypassDepths } from './context';

/**
 * Execute fn with RecordRule and FieldRule bypass.
 */
export async function withPermissionGraphBypass<T>(fn: () => Promise<T>): Promise<T> {
  const { req } = getJsCtxAndReq();
  if (!req) return await fn();

  const state = getOrInitReqServiceState(req);

  const hadCompanyMode = Object.prototype.hasOwnProperty.call(req, 'companyMode');
  const prevCompanyMode = req.companyMode;
  req.companyMode = 'skip';

  try {
    return await withBypassDepths(state, ['recordRuleBypassDepth', 'fieldRuleBypassDepth'], fn);
  } finally {
    if (hadCompanyMode) req.companyMode = prevCompanyMode;
    else delete req.companyMode;
  }
}
