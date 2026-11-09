// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** ChoyBadge variant for catalog status chips. */
export function moduleStatusBadgeVariant(
  status?: string,
  available?: boolean,
  updateAvailable?: boolean,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (available === false) return 'destructive';
  const val = String(status || '').toLowerCase();
  // Keep status chips quiet so card CTAs (Install / Upgrade) own the accent color.
  if ((val === 'installed' || val === 'succeeded') && updateAvailable === true) {
    return 'secondary';
  }
  if (val === 'installed' || val === 'succeeded') return 'outline';
  if (val === 'broken' || val === 'failed') return 'destructive';
  if (val === 'uninstalled') return 'secondary';
  return 'secondary';
}

/** Card summary text; empty when missing or duplicate of the module name. */
export function moduleCatalogCardSummary(rec: Record<string, unknown>): string {
  const name = String(rec.ModuleName || '').trim();
  const summary = String(rec.Summary || '').trim();
  if (!summary) return '';
  if (name && summary.toLowerCase() === name.toLowerCase()) return '';
  return summary;
}

/** Card application line; empty when missing (do not render a placeholder dash). */
export function moduleCatalogCardApplication(rec: Record<string, unknown>): string {
  return String(rec.Application || '').trim();
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

export type CatalogSyncResultSummary = {
  total: number;
  success: number;
  failed: number;
  catalogRows: number;
  origin?: string;
};

/** Parse PlanJson.syncResult from a MetaModuleOperation row. */
export function parseCatalogSyncResult(planJson: unknown): CatalogSyncResultSummary | null {
  let doc: Record<string, unknown> | null = null;
  if (typeof planJson === 'string' && planJson.trim()) {
    try {
      doc = JSON.parse(planJson) as Record<string, unknown>;
    } catch {
      return null;
    }
  } else if (planJson && typeof planJson === 'object') {
    doc = planJson as Record<string, unknown>;
  }
  if (!doc) return null;
  const raw = (doc.syncResult ?? doc.SyncResult) as Record<string, unknown> | undefined;
  if (!raw || typeof raw !== 'object') return null;
  const num = (v: unknown) => {
    const n = typeof v === 'number' ? v : Number(v);
    return Number.isFinite(n) ? Math.max(0, Math.floor(n)) : 0;
  };
  return {
    total: num(raw.total ?? raw.Total),
    success: num(raw.success ?? raw.Success),
    failed: num(raw.failed ?? raw.Failed),
    catalogRows: num(raw.catalogRows ?? raw.CatalogRows),
    origin: raw.origin != null || raw.Origin != null ? String(raw.origin ?? raw.Origin) : undefined,
  };
}

/** Condense an op summary payload into display text (non-i18n fallback). */
export function formatModuleOpSummary(summary: unknown): string {
  if (!summary) return '';
  if (typeof summary === 'string') return summary;
  if (typeof summary === 'object' && summary !== null) {
    const record = summary as Record<string, unknown>;
    const sync = record.syncResult as CatalogSyncResultSummary | undefined;
    if (sync && typeof sync === 'object') {
      if (sync.failed > 0) {
        return `Synced ${sync.success}/${sync.total} sources (${sync.failed} failed); catalog has ${sync.catalogRows} modules`;
      }
      return `Synced ${sync.total} sources; catalog has ${sync.catalogRows} modules`;
    }
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

/** True when summary carries a syncResult with failed > 0. */
export function catalogSyncSummaryPartial(summary: unknown): boolean {
  if (!summary || typeof summary !== 'object') return false;
  const sync = (summary as { syncResult?: CatalogSyncResultSummary }).syncResult;
  return !!sync && sync.failed > 0;
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
  // Dense catalog cards only emphasize Upgrade when a newer source version exists.
  if (rec.UpdateAvailable === true) return 'upgrade';
  return null;
}

/**
 * Primary CTA for Module Show.
 * Installed modules always offer Upgrade (Odoo-style re-entry), even when
 * UpdateAvailable is false; catalog cards stay gated separately.
 */
export function resolveModuleCatalogFormPrimaryAction(
  rec: Record<string, unknown>,
): 'install' | 'upgrade' | 'progress' | null {
  const activeOp = String(rec.ActiveOperationId || '').trim();
  if (activeOp) return 'progress';
  if (rec.Available === false) return null;
  if (rec.Installed !== true) return 'install';
  return 'upgrade';
}
