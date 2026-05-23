// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package task defines the stable task queue, schedule, and event bus
// contracts shared by the default runtime and future alternate
// implementations.
//
// The package models the minimum semantics needed by the current task runtime:
// queued jobs can be enqueued, claimed, retried, completed, or cancelled;
// schedules can advance and trigger jobs; and event delivery uses a small
// envelope that is transport-agnostic.
package task
