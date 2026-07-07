// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type { Context } from '../runtime/context';
export {
  getActiveCompanyId,
  getContextLang,
  getContextTimezone,
  getCurrentReq,
  getCtxValue,
  getEnabledCompanyIds,
  getIdentity,
  getJsCtxAndReq,
  getOrInitReqServiceState,
  getReadonlyCtx,
  getReqMeta,
  getUserId,
  withContext,
  deleteReqStateKeysByPrefix,
  invalidateJsCtxSymbolCache,
} from '../runtime/context';
