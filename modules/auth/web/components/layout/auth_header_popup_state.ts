// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** True when auth header popup refs should be cleared after logout / session loss. */
export function shouldResetAuthHeaderPopups(
  wasAuthenticated: boolean,
  isAuthenticated: boolean,
): boolean {
  return wasAuthenticated && !isAuthenticated;
}
