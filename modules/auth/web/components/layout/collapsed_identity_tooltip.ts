// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Label for collapsed company/user tip hosts.
 * Returns undefined while the popover/dropdown is open so tip and panel do not stack.
 * Tip hosts hide the content when the sidebar is expanded or mobile.
 */
export function collapsedIdentityTooltip(
  menuOpen: boolean,
  label: string,
): string | undefined {
  if (menuOpen) return undefined;
  const text = String(label ?? '').trim();
  return text || undefined;
}
