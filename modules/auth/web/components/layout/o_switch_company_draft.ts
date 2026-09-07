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
