// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

// GarbageCollector is the stable lifecycle seam for task retention cleanup.
//
// Alternate runtimes may provide their own collector to enforce different
// retention policies, storage-specific purge strategies, or remote cleanup
// orchestration while preserving the host lifecycle contract.
type GarbageCollector interface {
	Start()
	Stop()
}
