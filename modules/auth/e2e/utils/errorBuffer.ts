// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Page } from '@choysum/e2e';

/**
 * Installs window error buffers for specs that previously used page.on('pageerror'/'console').
 * Call before navigation; drain with drainPageErrorBuffer.
 */
export async function installPageErrorBuffer(page: Page): Promise<void> {
  await page.evaluate(() => {
    const w = window as any;
    if (w.__choysum_e2e_error_buffer_installed__) return;
    w.__choysum_e2e_error_buffer_installed__ = true;
    w.__choysum_e2e_errors__ = [];
    const push = (tag: string, text: string) => {
      w.__choysum_e2e_errors__.push(`[${tag}] ${text}`);
    };
    window.addEventListener('error', ev => {
      const msg = ev?.error?.message || ev?.message || String(ev);
      push('pageerror', msg);
    });
    window.addEventListener('unhandledrejection', ev => {
      const reason = (ev as PromiseRejectionEvent)?.reason;
      const msg = reason?.message || String(reason);
      push('unhandledrejection', msg);
    });
    const orig = console.error.bind(console);
    console.error = (...args: unknown[]) => {
      push('console.error', args.map(a => String(a)).join(' '));
      orig(...args);
    };
  });
}

/** Returns and clears buffered page/console errors. */
export async function drainPageErrorBuffer(page: Page): Promise<string[]> {
  return page.evaluate(() => {
    const w = window as any;
    const out = Array.isArray(w.__choysum_e2e_errors__) ? w.__choysum_e2e_errors__.slice() : [];
    w.__choysum_e2e_errors__ = [];
    return out;
  });
}

/** Filters buffered signals matching permission_denied / access denied / auth boot RPCs. */
export function filterDeniedSignals(signals: string[]): string[] {
  return signals.filter(s => /permission_denied|access denied|\/auth\.User\/(Browse|GetPermissionState)/i.test(s));
}
