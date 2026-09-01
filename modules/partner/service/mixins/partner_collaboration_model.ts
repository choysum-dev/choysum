// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import AttachmentOwnerMixin from '@/core/service/mixins/attachment_owner_model';
import MessageThreadModel from '@/core/service/mixins/message_thread_model';
import type {
  AttachmentOwnerBindReq,
  AttachmentOwnerBindResp,
  AttachmentOwnerUnbindReq,
  AttachmentOwnerUnbindResp,
} from '@/core/service/mixins';

/**
 * Partner stacks message-thread and attachment-owner dial facades under TS
 * single inheritance. Both facades live in core; runtime still dials
 * `message.*` / `document.AttachmentBinding` (depends listed in package.json).
 *
 * Extends must use the mixin module default export (parser contract).
 */
export default abstract class PartnerCollaborationModel extends MessageThreadModel {
  /** Bind finalized attachment content to a partner field. */
  public static async AttachmentBind(req: AttachmentOwnerBindReq): Promise<AttachmentOwnerBindResp> {
    return AttachmentOwnerMixin.AttachmentBind(req);
  }

  /** Unbind an attachment from a partner field. */
  public static async AttachmentUnbind(req: AttachmentOwnerUnbindReq): Promise<AttachmentOwnerUnbindResp> {
    return AttachmentOwnerMixin.AttachmentUnbind(req);
  }
}
