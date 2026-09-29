// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computed,
  inject,
  onBeforeUnmount,
  onMounted,
  provide,
  shallowRef,
  toValue,
  type ComputedRef,
  type InjectionKey,
  type MaybeRefOrGetter,
  type ShallowRef,
} from 'vue';

/** List/kanban surface used by page title Import/Export actions. */
export type PageActionTarget = {
  selectedItems?: { value?: Array<{ Id?: string }> } | Array<{ Id?: string }> | null;
  refresh?: () => Promise<void> | void;
};

/** Default screen-level store and optional primary view action target from `Page`. */
export type PageContext = {
  store: ComputedRef<unknown | null>;
  actionTarget: ShallowRef<PageActionTarget | null>;
  registerActionTarget: (target: PageActionTarget) => void;
  unregisterActionTarget: (target: PageActionTarget) => void;
};

/** Stable across `vi.resetModules()` so provide/inject still match in unit tests. */
export const PageContextKey: InjectionKey<PageContext> = Symbol.for('choysum.pageContext');

export function providePageContext(options: { store: MaybeRefOrGetter<unknown | null | undefined> }) {
  const store = computed(() => (toValue(options.store) ?? null) as unknown | null);
  const actionTarget = shallowRef<PageActionTarget | null>(null);

  function registerActionTarget(target: PageActionTarget) {
    actionTarget.value = target;
  }

  function unregisterActionTarget(target: PageActionTarget) {
    if (actionTarget.value === target) {
      actionTarget.value = null;
    }
  }

  const ctx: PageContext = {
    store,
    actionTarget,
    registerActionTarget,
    unregisterActionTarget,
  };
  provide(PageContextKey, ctx);
  return ctx;
}

export function usePageContext(): PageContext | null {
  return inject(PageContextKey, null);
}

/** Reactive page store (null when Page did not provide one). */
export function useOptionalPageStore<T = unknown>(): ComputedRef<T | null> {
  const ctx = inject(PageContextKey, null);
  return computed(() => (ctx?.store.value ?? null) as T | null);
}

/**
 * Resolve store from an explicit prop, else from `Page` provided store.
 * Prefer for setup-time bindings used throughout a view.
 */
export function resolvePageStore<T>(propStore: T | null | undefined, label = 'component'): T {
  if (propStore !== undefined && propStore !== null) {
    return propStore;
  }
  const pageStore = inject(PageContextKey, null)?.store.value as T | null | undefined;
  if (pageStore !== undefined && pageStore !== null) {
    return pageStore;
  }
  throw new Error(`${label} requires a store: pass :store or set Page :store`);
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

/**
 * Register a list/kanban as the page IO action target when it shares the page store.
 * Pass `enabled: false` to opt out (embedded views); `enabled: true` forces registration.
 */
export function useRegisterPageActionTarget(options: {
  store: unknown;
  target: PageActionTarget;
  enabled?: MaybeRefOrGetter<boolean | undefined>;
}) {
  const ctx = inject(PageContextKey, null);
  let registered = false;

  onMounted(() => {
    if (!ctx) {
      return;
    }
    const flag = toValue(options.enabled);
    const pageStore = ctx.store.value;
    const matches = pageStore != null && options.store === pageStore;
    const should = flag === true || (flag !== false && matches);
    if (!should) {
      return;
    }
    ctx.registerActionTarget(options.target);
    registered = true;
  });

  onBeforeUnmount(() => {
    if (registered && ctx) {
      ctx.unregisterActionTarget(options.target);
      registered = false;
    }
  });
}
