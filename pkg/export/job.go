// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

// DataTransferJobRecord is the async export domain DTO (maps to lean task.DataTransferJob).
// Queue status lives on task.Job, not here. Direction is always "export" for this package.
type DataTransferJobRecord struct {
	Profile          Profile `json:"profile"`
	TargetModel      string  `json:"target_model,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
	TaskJobID        string  `json:"task_job_id,omitempty"`
	CompanyID        string  `json:"company_id,omitempty"`
	SpecSnapshotJSON []byte  `json:"spec_snapshot_json,omitempty"`
	Direction        string  `json:"direction,omitempty"` // always export for this DTO
}
