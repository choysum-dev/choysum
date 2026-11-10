// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

// TopicDispatchWakeup accelerates the task dispatcher. String value is frozen;
// hybrid poll remains the multi-instance reliability path.
const TopicDispatchWakeup = "task.dispatch.wakeup"

// TopicMessageThreadChanged notifies a Form Chatter thread that messages changed.
const TopicMessageThreadChanged = "message.thread.changed"

// TopicMessageNotificationUser notifies one user's inbox that notifications changed.
const TopicMessageNotificationUser = "message.notification.user"

// TopicAuditFieldChangeAppended notifies a Form Chatter thread that field-change
// history was appended for a business record (best-effort; Unary remains authoritative).
const TopicAuditFieldChangeAppended = "audit.field_change.appended"

// TopicMetaModuleOpChanged notifies that a Meta module operation row changed.
// Tip is best-effort and located by operationId (jobId may be attached for
// diagnostics). Authoritative status is meta_module_operation, not tip payload.
const TopicMetaModuleOpChanged = "meta.module_op.changed"
