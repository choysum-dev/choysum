// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Avatar image upload needs document.AttachmentBinding (ChoyImageField dial).
 * Hide the field when the document module is not installed.
 */
export function isUserAvatarFieldVisible(caps: { document?: boolean } | null | undefined): boolean {
  return caps?.document === true;
}
