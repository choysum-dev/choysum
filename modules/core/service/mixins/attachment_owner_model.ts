// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../orm/model/model';
import { dial } from '../orm/model/model_pool';
import { ChoysumError, GrpcCode } from '../error';
import type { ModelConstructor } from '../../rpc/types';
import type {
  AttachmentOwnerBindReq,
  AttachmentOwnerBindResp,
  AttachmentOwnerUnbindReq,
  AttachmentOwnerUnbindResp,
} from './attachment_owner_contracts';

/** Typing stub: core must not import document.AttachmentBinding. */
type DocumentAttachmentBindingStub = ModelConstructor & {
  Bind(req: AttachmentOwnerBindReq): Promise<AttachmentOwnerBindResp>;
  Unbind(req: AttachmentOwnerUnbindReq): Promise<AttachmentOwnerUnbindResp>;
};

/** Stable code when `document.AttachmentBinding` is not registered (module not installed). */
export const ATTACHMENT_OWNER_SERVICE_FACTORY_NOT_FOUND = 'SERVICE_FACTORY_NOT_FOUND';

function dialAttachmentBinding() {
  try {
    return dial<DocumentAttachmentBindingStub>('document.AttachmentBinding');
  } catch (err) {
    const cause = err instanceof Error ? err : new Error(String(err ?? 'unknown'));
    throw new ChoysumError({
      domain: 'core',
      code: ATTACHMENT_OWNER_SERVICE_FACTORY_NOT_FOUND,
      message: cause.message || `Service factory for model 'document.AttachmentBinding' not found.`,
    })
      .withGrpcCode(GrpcCode.FailedPrecondition)
      .withCause(cause);
  }
}

/**
 * Opt-in dial facade for business models with attachment owner fields.
 *
 * Lives in core so business apps may `extends` it under the hard rule that
 * cross-Application value imports are forbidden except from core. Runtime calls
 * dial `document.AttachmentBinding` — not BaseModel, not a document model import.
 *
 * Soft dependency on document: if `document.AttachmentBinding` is not
 * registered, bind/unbind throw {@link ChoysumError} with code
 * {@link ATTACHMENT_OWNER_SERVICE_FACTORY_NOT_FOUND} (FailedPrecondition).
 * Callers / UI must treat bind/unbind as unavailable until document is installed.
 *
 * ```ts
 * import AttachmentOwnerMixin from '@/core/service/mixins/attachment_owner_model';
 * @Model('Partner', { application: 'partner' })
 * export default class Partner extends AttachmentOwnerMixin {}
 * ```
 */
export default abstract class AttachmentOwnerMixin extends BaseModel {
  /** Bind finalized attachment content to an owner record field. */
  public static async AttachmentBind(req: AttachmentOwnerBindReq): Promise<AttachmentOwnerBindResp> {
    return dialAttachmentBinding().Bind(req);
  }

  /** Unbind an attachment from an owner record field. */
  public static async AttachmentUnbind(req: AttachmentOwnerUnbindReq): Promise<AttachmentOwnerUnbindResp> {
    return dialAttachmentBinding().Unbind(req);
  }
}
