// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Default land path after brand click / soft-land.
 * Stub until the W4 slice wires menu leaves + canNavigate + Module Board fallback.
 */
export function resolveDefaultLandPath(_opts?: {
  menuLeafPaths?: readonly string[];
  canNavigate?: (path: string) => boolean;
}): string {
  return '/';
}
