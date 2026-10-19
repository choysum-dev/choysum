// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type CommandPaletteHotkeyEvent = {
  key: string
  metaKey?: boolean
  ctrlKey?: boolean
  repeat?: boolean
  defaultPrevented?: boolean
  target?: EventTarget | { tagName?: string; isContentEditable?: boolean } | null
}

/**
 * Returns true when Ctrl/Cmd+K should toggle the command palette.
 * Skips repeats, already-handled events, and editable targets.
 */
export function shouldToggleCommandPalette(event: CommandPaletteHotkeyEvent): boolean {
  if (event.key !== 'k' && event.key !== 'K') return false
  if (!(event.metaKey || event.ctrlKey)) return false
  if (event.repeat || event.defaultPrevented) return false
  const target = event.target as { tagName?: string; isContentEditable?: boolean } | null | undefined
  const tag = target?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return false
  return true
}
