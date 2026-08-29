// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computed,
  inject,
  provide,
  type ComputedRef,
  type InjectionKey,
  type MaybeRefOrGetter,
  toValue,
} from 'vue';

/** Default screen-level store provided by `OPage`. */
export type OPageContext = {
  store: ComputedRef<unknown | null>;
};

/** Stable across `vi.resetModules()` so provide/inject still match in unit tests. */
export const OPageContextKey: InjectionKey<OPageContext> = Symbol.for('choysum.oPageContext');

export function provideOPageContext(options: { store: MaybeRefOrGetter<unknown | null | undefined> }) {
  const store = computed(() => (toValue(options.store) ?? null) as unknown | null);
  const ctx: OPageContext = { store };
  provide(OPageContextKey, ctx);
  return ctx;
}

export function useOPageContext(): OPageContext | null {
  return inject(OPageContextKey, null);
}

/** Reactive page store (null when OPage did not provide one). */
export function useOptionalPageStore<T = unknown>(): ComputedRef<T | null> {
  const ctx = inject(OPageContextKey, null);
  return computed(() => (ctx?.store.value ?? null) as T | null);
}

/**
 * Resolve store from an explicit prop, else from `OPage` provided store.
 * Prefer for setup-time bindings used throughout a view.
 */
export function resolvePageStore<T>(propStore: T | null | undefined, label = 'component'): T {
  if (propStore !== undefined && propStore !== null) {
    return propStore;
  }
  const pageStore = inject(OPageContextKey, null)?.store.value as T | null | undefined;
  if (pageStore !== undefined && pageStore !== null) {
    return pageStore;
  }
  throw new Error(`${label} requires a store: pass :store or set OPage :store`);
}

/**
 * Soft resolve for callers that can run without a store (e.g. export disabled).
 * Call inject-related helpers only during setup; wrap prop reads in computed at the call site.
 */
export function useResolvedOptionalPageStore<T>(
  propStore: MaybeRefOrGetter<T | null | undefined>,
): ComputedRef<T | null> {
  const pageStore = useOptionalPageStore<T>();
  return computed(() => {
    const fromProp = toValue(propStore);
    if (fromProp !== undefined && fromProp !== null) {
      return fromProp;
    }
    return pageStore.value;
  });
}
