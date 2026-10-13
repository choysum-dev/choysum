// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Label for SidebarMenuButton tooltip on collapsed company/user triggers.
 * Returns undefined while the popover/dropdown is open so tip and panel do not stack.
 * Kit TooltipContent already hides when the sidebar is expanded or mobile.
 */
export function collapsedIdentityTooltip(
  menuOpen: boolean,
  label: string,
): string | undefined {
  if (menuOpen) return undefined;
  const text = String(label ?? '').trim();
  return text || undefined;
}
