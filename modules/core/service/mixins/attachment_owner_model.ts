// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../orm/model/model';
import { dial } from '../orm/model/model_pool';
import type {
  AttachmentOwnerBindReq,
  AttachmentOwnerBindResp,
  AttachmentOwnerUnbindReq,
  AttachmentOwnerUnbindResp,
} from './attachment_owner_contracts';

type AttachmentBindingService = {
  Bind(req: AttachmentOwnerBindReq): Promise<AttachmentOwnerBindResp>;
  Unbind(req: AttachmentOwnerUnbindReq): Promise<AttachmentOwnerUnbindResp>;
};

/**
 * Opt-in dial facade for business models with attachment owner fields.
 *
 * Lives in core so business apps may `extends` it under the hard rule that
 * cross-Application value imports are forbidden except from core. Runtime calls
 * dial `document.AttachmentBinding` — not BaseModel, not a document model import.
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
    return dial<AttachmentBindingService>('document.AttachmentBinding').Bind(req);
  }

  /** Unbind an attachment from an owner record field. */
  public static async AttachmentUnbind(req: AttachmentOwnerUnbindReq): Promise<AttachmentOwnerUnbindResp> {
    return dial<AttachmentBindingService>('document.AttachmentBinding').Unbind(req);
  }
}
