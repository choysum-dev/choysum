// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type { ProxyFactory } from './proxy';
export { ModelProxyFactory } from './proxy';
export { MODEL_SYMBOLS } from './symbols';

export { createPreviewProxy, type PreviewProxyCtx } from './onchangePreviewProxy';
export { createWriteProxy } from './onchangeDraftProxy';
export { getProxyKind, isBrandedProxy, markProxyKind, type ChoysumProxyKind } from './brand';
export {
  DRAFT_FORBIDDEN_PERSISTENCE_METHODS,
  createForbiddenPersistenceMethodStub,
  isDraftForbiddenPersistenceMethod,
  type DraftPersistenceGuardContext,
} from './draftPersistenceGuards';

import BaseModel from '../../orm/model/model';
import { createPreviewProxy, type PreviewProxyCtx } from './onchangePreviewProxy';
import { createWriteProxy } from './onchangeDraftProxy';

/**
 * Compose writeProxy as the inner layer and previewProxy as the outer layer.
 *
 * Used in Onchange flows to combine patch collection with read-only preview behavior.
 */
export function createOnchangeDraft<T extends BaseModel>(base: T, params: PreviewProxyCtx & { patchSink: (path: string, value: unknown) => void }): T {
  const { patchSink, ...previewCtx } = params;
  const writable = createWriteProxy(base, patchSink);
  return createPreviewProxy<T>(writable, previewCtx);
}
