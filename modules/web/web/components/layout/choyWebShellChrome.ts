// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Short label for collapsed rail attribution (version abbrev). */
export function shortAppVersion(version: string | undefined | null): string {
  const v = String(version ?? '').trim() || 'dev';
  if (v.length <= 6) return v;
  return v.slice(0, 6);
}

/** One-line page footer: Powered by Choysum v0.1.0 */
export function shellPoweredByLine(version: string | undefined | null): string {
  const raw = String(version ?? '').trim() || 'dev';
  const label = /^v/i.test(raw) ? raw : `v${raw}`;
  return `Powered by Choysum ${label}`;
}

/** Aria / tooltip label for the shell menu trigger. */
export function shellMenuTriggerLabel(opts: {
  isMobile: boolean;
  railCollapsed: boolean;
  t: (key: string) => string;
}): string {
  if (opts.isMobile) return opts.t('layout.header.menu');
  return opts.railCollapsed ? opts.t('layout.sidebar.expand') : opts.t('layout.sidebar.collapse');
}

/** Whether Escape should dismiss the mobile nav drawer. */
export function shouldCloseDrawerOnEscape(event: {
  key: string;
  defaultPrevented?: boolean;
}, drawerOpen: boolean): boolean {
  if (event.defaultPrevented) return false;
  return event.key === 'Escape' && drawerOpen;
}

/** Prior body overflow captured while the drawer owns the scroll lock. */
let savedBodyOverflow: string | null = null;

/**
 * Lock or unlock document body scroll while the mobile drawer is open.
 * Restores any overflow value that was present before this helper locked.
 */
export function setDrawerBodyOverflow(locked: boolean): void {
  if (locked) {
    if (savedBodyOverflow === null) {
      savedBodyOverflow = document.body.style.overflow;
    }
    document.body.style.overflow = 'hidden';
    return;
  }
  if (savedBodyOverflow !== null) {
    document.body.style.overflow = savedBodyOverflow;
    savedBodyOverflow = null;
  }
}
