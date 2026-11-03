// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Avatar image upload needs document.AttachmentBinding (ChoyImageField dial).
 * `caps.document === true` means the document module (and that factory) is loaded;
 * hide the field otherwise so UI stays aligned with AttachmentOwnerMixin soft-dial.
 */
export function isUserAvatarFieldVisible(caps: { document?: boolean } | null | undefined): boolean {
  return caps?.document === true;
}
