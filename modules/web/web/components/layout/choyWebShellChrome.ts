// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Short label for collapsed rail attribution (version abbrev). */
export function shortAppVersion(version: string | undefined | null): string {
  const v = String(version ?? '').trim() || 'dev';
  if (v.length <= 6) return v;
  return v.slice(0, 6);
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

/** Lock or unlock document body scroll while the mobile drawer is open. */
export function setDrawerBodyOverflow(locked: boolean): void {
  document.body.style.overflow = locked ? 'hidden' : '';
}
