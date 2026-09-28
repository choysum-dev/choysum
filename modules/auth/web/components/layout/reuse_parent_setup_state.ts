// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Keep only plain setup-state objects from a parent `setup` result.
 * A script-setup base returns a render function; spreading that would silently yield nothing.
 * Non-plain objects (Map/Set/class/Ref) are dropped so their internals cannot leak into setup.
 * Async parent setup is rejected: flattening a Promise would silently yield {} and break markup.
 */
export function reuseParentSetupState(result: unknown): Record<string, unknown> {
  if (result instanceof Promise) {
    throw new Error('reuseParentSetupState: async parent setup is not supported');
  }
  if (!result || typeof result !== 'object' || Array.isArray(result)) {
    return {};
  }
  const proto = Object.getPrototypeOf(result);
  if (proto !== null && proto !== Object.prototype) {
    return {};
  }
  return result as Record<string, unknown>;
}
