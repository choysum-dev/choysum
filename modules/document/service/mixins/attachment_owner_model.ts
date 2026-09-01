// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import type { BindReq, BindResp, UnbindReq, UnbindResp } from '../contracts';
import AttachmentBinding from '../models/attachment_binding';

/**
 * Opt-in mixin for business models with attachment owner fields.
 * Other apps extend this default-exported base:
 *
 * ```ts
 * import AttachmentOwnerMixin from '@/document/service/mixins/attachment_owner_model';
 * @Model('Partner', { application: 'partner' })
 * export default class Partner extends AttachmentOwnerMixin {}
 * ```
 *
 * Bind/unbind entry points dial document.AttachmentBinding — not on BaseModel.
 */
export default abstract class AttachmentOwnerMixin extends BaseModel {
  /** Bind finalized attachment content to an owner record field. */
  public static async AttachmentBind(req: BindReq): Promise<BindResp> {
    return AttachmentBinding.Bind(req);
  }

  /** Unbind an attachment from an owner record field. */
  public static async AttachmentUnbind(req: UnbindReq): Promise<UnbindResp> {
    return AttachmentBinding.Unbind(req);
  }
}
