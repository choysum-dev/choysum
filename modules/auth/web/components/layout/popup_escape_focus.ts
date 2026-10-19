// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Close an open popup on Escape and return keyboard focus to its trigger.
 * Returns true when the popup was open and dismissed.
 */
export function dismissPopupOnEscape(
  isOpen: boolean,
  close: () => void,
  trigger: { focus?: () => void } | null | undefined,
): boolean {
  if (!isOpen) return false;
  close();
  trigger?.focus?.();
  return true;
}
