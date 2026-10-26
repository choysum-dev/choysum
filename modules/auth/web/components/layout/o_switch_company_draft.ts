// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Sync company-switch drafts from JWT metadata when the panel is closed.
 * While the panel is open, drafts stay user-owned and JWT refreshes are ignored.
 */
export function syncCompanyDraftsFromJwt(opts: {
  panelVisible: boolean;
  activeCompanyId: string;
  enabledCompanyIds: string[];
  apply: (activeCompanyId: string, enabledCompanyIds: string[]) => void;
}): void {
  if (opts.panelVisible) return;
  opts.apply(opts.activeCompanyId, opts.enabledCompanyIds);
}

/**
 * The active company must stay in the enabled set; lock its checkbox so users
 * cannot uncheck a value that `ensureActiveInEnabled` would immediately restore.
 */
export function isActiveCompanyEnabledLocked(
  companyId: string,
  draftActiveCompanyId: string,
): boolean {
  const active = String(draftActiveCompanyId ?? '').trim();
  if (!active) return false;
  return String(companyId ?? '').trim() === active;
}

/**
 * Toggle one company in the enabled draft. The active company stays locked in
 * and is always included after the toggle.
 */
export function toggleEnabledCompanyId(opts: {
  companyId: string;
  on: boolean | 'indeterminate';
  draftActiveCompanyId: string;
  draftEnabledCompanyIds: readonly string[];
}): string[] {
  const id = String(opts.companyId ?? '').trim();
  const active = String(opts.draftActiveCompanyId ?? '').trim();
  let next = Array.from(
    new Set(opts.draftEnabledCompanyIds.map(x => String(x ?? '').trim()).filter(Boolean)),
  );
  if (opts.on === true) {
    if (id && !next.includes(id)) next = [...next, id];
  } else if (id && !isActiveCompanyEnabledLocked(id, active)) {
    next = next.filter(x => x !== id);
  }
  if (active && !next.includes(active)) {
    next = [active, ...next];
  }
  return next;
}
