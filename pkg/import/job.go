// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg

// DataTransferJobRecord is the async domain DTO (maps to lean task.DataTransferJob).
// Queue status lives on task.Job, not here.
type DataTransferJobRecord struct {
	Profile          Profile `json:"profile"`
	Policy           Policy  `json:"policy"`
	DryRun           bool    `json:"dry_run"`
	TargetModel      string  `json:"target_model,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
	TaskJobID        string  `json:"task_job_id,omitempty"`
	CompanyID        string  `json:"company_id,omitempty"`
	SpecSnapshotJSON []byte  `json:"spec_snapshot_json,omitempty"`
	Direction        string  `json:"direction,omitempty"` // import | export
}
