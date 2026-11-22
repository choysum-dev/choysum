// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Optional bridge from RPC lifecycle to product connectivity (web store).
 * Core must not import Pinia; web registers handlers after Pinia is ready.
 */
export type ConnectivityHooks = {
  onTransportSuccess?: () => void;
  onTransportOutage?: () => void;
};

let hooks: ConnectivityHooks = {};

export function setConnectivityHooks(next: ConnectivityHooks | null | undefined): void {
  hooks = next && typeof next === 'object' ? { ...next } : {};
}

export function getConnectivityHooks(): ConnectivityHooks {
  return hooks;
}

/** Test helper: clear registered hooks. */
export function resetConnectivityHooksForTests(): void {
  hooks = {};
}
