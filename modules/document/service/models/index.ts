// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Document layout exemplar: upload/binding workflow lives on AttachmentContent / AttachmentBinding;
// codec/GC/owner-auth stay as adjacent `_*.ts` helpers. Owner models extend AttachmentOwnerMixin.
export { default as AttachmentContent } from './attachment_object';
export { default as AttachmentBinding } from './attachment_binding';
export { default as AttachmentUploadSession } from './upload_session';
export { default as AttachmentMutationLedger } from './attachment_mutation_ledger';
export { default as StoredContent } from './stored_content';
