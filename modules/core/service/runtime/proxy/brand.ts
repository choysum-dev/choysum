// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Runtime proxy identity brands used to reject invalid Service Wrapper `thisArg`
 * values (draft / bridge / hydrate proxies must not bind into conventional RPC).
 *
 * See `.dev/docs/core/service/record_lifecycle_proxy_wrapper_boundary_plan20260715.md`.
 */
export type ChoysumProxyKind = 'onchange-write' | 'onchange-preview' | 'constraint-draft' | 'bridge-execution' | 'model-hydrate';

const PROXY_KINDS = new WeakMap<object, ChoysumProxyKind>();

export function markProxyKind(proxy: object, kind: ChoysumProxyKind): void {
  PROXY_KINDS.set(proxy, kind);
}

export function getProxyKind(value: unknown): ChoysumProxyKind | undefined {
  if (!value || (typeof value !== 'object' && typeof value !== 'function')) {
    return undefined;
  }
  return PROXY_KINDS.get(value as object);
}

export function isBrandedProxy(value: unknown): boolean {
  return getProxyKind(value) !== undefined;
}
