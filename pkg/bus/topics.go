// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bus

// TopicDispatchWakeup accelerates the task dispatcher. String value is frozen;
// hybrid poll remains the multi-instance reliability path.
const TopicDispatchWakeup = "task.dispatch.wakeup"
