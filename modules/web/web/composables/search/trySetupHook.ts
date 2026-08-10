// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Call an inject()/setup-only hook; return null when the caller is outside setup. */
export function trySetupHook<T>(fn: () => T): T | null {
  try {
    return fn();
  } catch {
    return null;
  }
}
