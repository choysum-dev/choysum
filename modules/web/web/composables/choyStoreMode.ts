// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Detects whether a Choy field/view should host the store-bound O* engine.
 * Chrome / Dogfood paths omit store+prop and keep defineModel APIs.
 * `pageStore` covers fields under a store-backed ChoyPage that only pass `prop`.
 */
export function isChoyStoreFieldBinding(
  props: {
    store?: unknown;
    prop?: unknown;
    binding?: unknown;
  },
  pageStore?: unknown | null,
): boolean {
  if (props.binding != null) return true;
  const store = props.store ?? pageStore ?? null;
  return store != null && props.prop != null && props.prop !== '';
}

/**
 * True when an explicit store prop or page-provided store is available.
 */
export function hasChoyStoreEngine(
  propStore: unknown | null | undefined,
  pageStore: unknown | null | undefined,
): boolean {
  return propStore != null || pageStore != null;
}

/**
 * Split useAttrs() into non-listener bind props and v-on listener keys.
 * Attrs use Vue's onFoo form; v-on="listeners" runs toHandlers which prepends
 * on again, so listeners must be { foo: fn } (not { onFoo: fn }).
 */
export function splitChoyAttrsListeners(attrs: Record<string, unknown>): {
  bind: Record<string, unknown>;
  listeners: Record<string, unknown>;
} {
  const bind: Record<string, unknown> = {};
  const listeners: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith('on') && key.length > 2 && typeof value === 'function') {
      const rawEvent = key.startsWith('on-') ? key.slice(3) : key.slice(2);
      const eventName = rawEvent.charAt(0).toLowerCase() + rawEvent.slice(1);
      listeners[eventName] = value;
    } else {
      bind[key] = value;
    }
  }
  return { bind, listeners };
}
