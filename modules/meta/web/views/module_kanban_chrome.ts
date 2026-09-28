// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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
  if (!dt) return '';
  try {
    const d =
      typeof dt === 'number'
        ? new Date(dt)
        : typeof dt === 'string'
          ? new Date(dt)
          : dt instanceof Date
            ? dt
            : new Date(String(dt));
    if (!(d instanceof Date) || isNaN(d.getTime())) return String(dt).slice(0, 19);
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
    const message = record.message;
    if (message !== undefined && message !== null && String(message).trim() !== '') {
      return String(message);
    }
    const code = record.code;
    if (code !== undefined && code !== null && String(code).trim() !== '') {
      return String(code);
    }
  }
  try {
    return JSON.stringify(summary);
  } catch {
    return String(summary);
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
export function shouldRecoverStaleKanbanSearch(args: {
  completedSeq: number;
  latestSeq: number;
  inFlight: number;
}): boolean {
  return args.completedSeq !== args.latestSeq && args.inFlight === 0;
}

/** Capture the focused element before opening a modal (null when none). */
export function captureDialogFocusTarget(
  doc: Pick<Document, 'activeElement'> = document,
): HTMLElement | null {
  const el = doc.activeElement as { focus?: () => void } | null;
  if (!el || typeof el.focus !== 'function') return null;
  return el as HTMLElement;
}

/**
 * Navigable record id from a kanban card payload. Blank ids fail closed.
 */
export function resolveModuleKanbanCardId(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return '';
  return String((payload as { Id?: unknown }).Id ?? '').trim();
}
