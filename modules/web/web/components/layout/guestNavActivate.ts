// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * True for a plain primary-button click with no modifier keys.
 * Modified clicks (Ctrl/Cmd/Shift/Alt or non-left button) should keep native
 * anchor navigation so "open in new tab" works.
 */
export function isUnmodifiedPrimaryClick(e: MouseEvent): boolean {
  // Programmatic HTMLElement.click() often omits button; treat missing as primary.
  const button = e.button ?? 0
  return !(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || button !== 0)
}
