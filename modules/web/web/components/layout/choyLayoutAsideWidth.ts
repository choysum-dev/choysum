// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** CSS width for the nav rail from density tokens. */
export function choyLayoutAsideWidth(collapsed: boolean): string {
  return collapsed
    ? 'var(--choy-layout-sidebar-collapsed-width, 3.5rem)'
    : 'var(--choy-layout-sidebar-width, 15rem)';
}
