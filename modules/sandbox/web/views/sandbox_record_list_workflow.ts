// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type SandboxWorkflowState = 'draft' | 'open' | 'done';

/** Unwrap List selection row wrappers to a field map. */
export function unwrapSandboxListRecord(row: unknown): Record<string, unknown> | null {
  if (!row || typeof row !== 'object') return null;
  const r = row as Record<string, unknown>;
  if (r.type === 'record' && r.record && typeof r.record === 'object') {
    return r.record as Record<string, unknown>;
  }
  if (r.kind === 'record' && r.payload && typeof r.payload === 'object') {
    return r.payload as Record<string, unknown>;
  }
  if (r.payload && typeof r.payload === 'object') return r.payload as Record<string, unknown>;
  return r;
}

export function resolveSandboxWorkflowState(row: Record<string, unknown>): SandboxWorkflowState {
  const raw = String(row.State ?? row.Progress ?? 'draft').trim() || 'draft';
  if (raw === 'open' || raw === 'done') return raw;
  return 'draft';
}

/** Source State for a Mark Open / Done / Reset Draft transition. */
export function sandboxWorkflowSourceState(next: SandboxWorkflowState): SandboxWorkflowState {
  if (next === 'open') return 'draft';
  if (next === 'done') return 'open';
  return 'done';
}

/** Ids among selection whose State matches the transition source. */
export function selectedSandboxIdsFor(
  from: SandboxWorkflowState,
  selectedItems: unknown,
): string[] {
  const rows = Array.isArray(selectedItems) ? selectedItems : [];
  const ids: string[] = [];
  for (const row of rows) {
    const rec = unwrapSandboxListRecord(row);
    if (!rec) continue;
    if (resolveSandboxWorkflowState(rec) !== from) continue;
    const id = String(rec.Id ?? '').trim();
    if (id) ids.push(id);
  }
  return ids;
}
