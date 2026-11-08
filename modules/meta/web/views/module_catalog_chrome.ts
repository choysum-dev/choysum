// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** ChoyBadge variant for catalog status chips. */
export function moduleStatusBadgeVariant(
  status?: string,
  available?: boolean,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (available === false) return 'destructive';
  const val = String(status || '').toLowerCase();
  if (val === 'installed' || val === 'succeeded') return 'default';
  if (val === 'broken' || val === 'failed') return 'destructive';
  if (val === 'uninstalled') return 'outline';
  return 'secondary';
}

/** True when InstalledStatus / Installed means the module is installed. */
export function isModuleInstalled(status?: string): boolean {
  return String(status || '').toLowerCase() === 'installed';
}

/** Normalize catalog Installed / InstalledStatus into a status token. */
export function catalogInstalledStatus(rec: Record<string, unknown>): string {
  if (rec.InstalledStatus) return String(rec.InstalledStatus);
  if (rec.Installed === true) return 'installed';
  if (rec.Installed === false) return 'uninstalled';
  return '';
}

/** Format module timestamps for card metadata (local YYYY-MM-DD HH:mm). */
export function formatModuleCatalogDate(dt?: unknown): string {
  if (dt === undefined || dt === null) return '';
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

/** Capture the focused element before opening a modal (null when none). */
export function captureDialogFocusTarget(
  doc: Pick<Document, 'activeElement'> = document,
): HTMLElement | null {
  const el = doc.activeElement as { focus?: () => void } | null;
  if (!el || typeof el.focus !== 'function') return null;
  return el as HTMLElement;
}

/**
 * Sequence gate so a superseded Plan response cannot overwrite a newer drawer session.
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

/** Version line for catalog cards (installed → catalog when an update is pending). */
export function formatModuleCatalogVersionLine(rec: Record<string, unknown>): string {
  const installed = String(rec.InstalledVersion || '').trim();
  const catalog = String(rec.LocalVersion || rec.RegistryVersion || '').trim();
  if (rec.Installed === true && rec.UpdateAvailable === true && installed && catalog) {
    return `${installed} → ${catalog}`;
  }
  if (rec.Installed === true) return installed || catalog || '—';
  return catalog || '—';
}

/** Primary CTA for a catalog row (null = no primary button). */
export function resolveModuleCatalogPrimaryAction(
  rec: Record<string, unknown>,
): 'install' | 'upgrade' | 'progress' | null {
  const activeOp = String(rec.ActiveOperationId || '').trim();
  if (activeOp) return 'progress';
  if (rec.Available === false) return null;
  if (rec.Installed !== true) return 'install';
  if (rec.UpdateAvailable === true) return 'upgrade';
  return null;
}
