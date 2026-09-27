// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Detects whether a Choy field/view should host the store-bound O* engine.
 * Chrome / Dogfood paths omit store+prop and keep defineModel APIs.
 */
export function isChoyStoreFieldBinding(props: {
  store?: unknown;
  prop?: unknown;
  binding?: unknown;
}): boolean {
  if (props.binding != null) return true;
  return props.store != null && props.prop != null && props.prop !== '';
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
