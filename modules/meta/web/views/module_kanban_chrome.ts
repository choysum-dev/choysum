// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveListRowRecordId } from './list_row_nav';

/**
 * Status badge Tailwind classes for module kanban cards and op dialogs.
 */
export function moduleStatusBadgeClass(status?: string, available?: boolean): string {
  if (available === false) return 'bg-destructive/15 text-destructive';
  const val = String(status || '').toLowerCase();
  if (val === 'installed' || val === 'succeeded') {
    return 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-200';
  }
  if (val === 'uninstalled') return 'bg-muted text-foreground/70';
  if (val === 'disabled' || val === 'dispatching' || val === 'queued') {
    return 'bg-amber-500/15 text-amber-900 dark:text-amber-100';
  }
  if (val === 'broken' || val === 'failed') return 'bg-destructive/15 text-destructive';
  return 'bg-muted text-foreground/70';
}

/** True when InstalledStatus means the module is installed. */
export function isModuleInstalled(status?: string): boolean {
  return String(status || '').toLowerCase() === 'installed';
}

/** Format module timestamps for card metadata (local YYYY-MM-DD HH:mm). */
export function formatModuleKanbanDate(dt?: unknown): string {
  if (dt === undefined || dt === null) return '';
  // Non-finite numbers stringify to "NaN"/"Infinity", which the card would render verbatim.
  if (typeof dt === 'number' && !Number.isFinite(dt)) return '';
  try {
    const d =
      typeof dt === 'number'
        ? new Date(dt)
        : typeof dt === 'string'
          ? new Date(dt)
          : dt instanceof Date
            ? dt
            : new Date(String(dt));
    if (!(d instanceof Date) || Number.isNaN(d.getTime())) {
      // Keep raw string fragments; avoid rendering "[object Object]" for structured payloads.
      return typeof dt === 'string' ? dt.slice(0, 19) : '';
    }
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    return `${y}-${m}-${day} ${hh}:${mm}`;
  } catch {
    try {
      return String(dt).slice(0, 19);
    } catch {
      return '';
    }
  }
}

/** Condense an op summary payload into display text. */
export function formatModuleOpSummary(summary: unknown): string {
  if (!summary) return '';
  if (typeof summary === 'string') return summary;
  if (typeof summary === 'object' && summary !== null) {
    const record = summary as Record<string, unknown>;
    for (const key of ['message', 'code'] as const) {
      const value = record[key];
      if (typeof value === 'string' && value.trim() !== '') return value;
      if (typeof value === 'number' && Number.isFinite(value)) return String(value);
    }
  }
  try {
    return JSON.stringify(summary) ?? String(summary);
  } catch {
    try {
      return String(summary);
    } catch {
      return '';
    }
  }
}

/** Short description from a module manifest object. */
export function manifestSummaryText(raw: unknown): string {
  if (!raw || typeof raw !== 'object') return '';
  const obj = raw as Record<string, unknown>;
  const text = obj.short_desc || obj.shortDesc || obj.summary || obj.description || obj.name || '';
  return typeof text === 'string' ? text : '';
}

/**
 * After a superseded search apply finishes last, recover when nothing else is in flight.
 */
export { shouldRecoverStaleKanbanSearch } from '@/web/web/components/view/kanbanStoreHelpers';

/** Capture the focused element before opening a modal (null when none). */
export function captureDialogFocusTarget(
  doc: Pick<Document, 'activeElement'> = document,
): HTMLElement | null {
  const el = doc.activeElement as { focus?: () => void } | null;
  if (!el || typeof el.focus !== 'function') return null;
  return el as HTMLElement;
}

/**
 * Sequence gate so a superseded PlanOperation response cannot overwrite a newer dialog.
 */
export function createPlanDialogSessionGate() {
  let seq = 0;
  return {
    begin(): number {
      seq += 1;
      return seq;
    },
    invalidate(): void {
      seq += 1;
    },
    isCurrent(requestSeq: number): boolean {
      return requestSeq === seq;
    },
  };
}

/**
 * Coalesce overlapping lane syncs: waiters resume after the in-flight pass (and any
 * follow-up resync) finishes, instead of resolving immediately.
 */
export { createLaneSyncGate } from '@/web/web/components/view/kanbanStoreHelpers';

/**
 * Navigable record id from a kanban card payload. Blank ids fail closed.
 * Shares scalar fail-closed rules with list row navigation.
 */
export const resolveModuleKanbanCardId = resolveListRowRecordId;

/**
 * Stable kanban card key: prefer fail-closed record id, then ModuleName, else lane+index.
 */
export function resolveModuleKanbanCardKey(
  payload: unknown,
  laneKey: string,
  index: number,
): string {
  const id = resolveListRowRecordId(payload);
  if (id) return id;
  if (payload && typeof payload === 'object') {
    const bag = payload as { row?: unknown; ModuleName?: unknown };
    const record =
      bag.row && typeof bag.row === 'object' ? (bag.row as { ModuleName?: unknown }) : bag;
    const name = record.ModuleName;
    if (typeof name === 'string' && name.trim() !== '') return name.trim();
  }
  return `${laneKey}-${index}`;
}
