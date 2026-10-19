// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Header menu button: prefer layoutStore so ChoySidebarBridge stays the
 * single writer to kit open state; fall back to the kit toggle otherwise.
 */
export function runShellMenuToggle(
  layoutStore: { toggleSidebar: () => void } | null | undefined,
  toggleSidebar: () => void,
): void {
  if (layoutStore) {
    layoutStore.toggleSidebar();
    return;
  }
  toggleSidebar();
}
