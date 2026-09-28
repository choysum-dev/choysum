// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isReactive, isReadonly } from 'vue';

/**
 * Keep only plain setup-state objects from a parent `setup` result.
 * A script-setup base returns a render function; spreading that would silently yield nothing.
 * Non-plain objects (Map/Set/class/Ref) are dropped so their internals cannot leak into setup.
 * Reactive/readonly proxies are dropped as well: spreading them copies unwrapped values,
 * which would silently lose reactivity for the merged shell.
 * Async parent setup (Promise or thenable) is rejected: flattening it would silently yield {}.
 */
export function reuseParentSetupState(result: unknown): Record<string, unknown> {
  if (result && typeof (result as { then?: unknown }).then === 'function') {
    throw new Error('reuseParentSetupState: async parent setup is not supported');
  }
  if (!result || typeof result !== 'object' || Array.isArray(result)) {
    return {};
  }
  if (isReactive(result) || isReadonly(result)) {
    return {};
  }
  const proto = Object.getPrototypeOf(result);
  // Realm-agnostic plain object: prototype is null or terminates immediately
  // (cross-realm Object.prototype !== local Object.prototype).
  if (proto !== null && Object.getPrototypeOf(proto) !== null) {
    return {};
  }
  // Shallow copy so later mutation of the returned record cannot alias parent state.
  return { ...(result as Record<string, unknown>) };
}
