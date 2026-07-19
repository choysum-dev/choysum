// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * D11: component hint from Scope path heuristics (no DB field).
 * Examples:
 *   web/pages/Login@title → web/pages/Login
 *   web/components/layout/OHeader@brand → web/components/layout/OHeader
 *   game.rescue → game.rescue
 */
export function componentHintFromScope(scope: string): string {
  const raw = String(scope || '').trim();
  if (!raw) {
    return '';
  }
  const at = raw.lastIndexOf('@');
  if (at > 0) {
    return raw.slice(0, at);
  }
  return raw;
}
