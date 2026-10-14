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
  watch,
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

/** Props for page Import/Export menu when a view hosts the title row. */
export type PageTitleActionsIo = {
  actionImport: boolean;
  actionExport: boolean;
  actionImportUploadHint?: string;
  actionImportColumnMapping?: Record<string, string>;
  actionListRef?: unknown;
  actionCompanyId?: string;
};

/** Default screen-level store and optional primary view action target from `Page`. */
export type PageContext = {
  store: ComputedRef<unknown | null>;
  actionTarget: ShallowRef<PageActionTarget | null>;
  registerActionTarget: (target: PageActionTarget) => void;
  unregisterActionTarget: (target: PageActionTarget) => void;
  /** Page title from ChoyPage (trimmed; empty when unset). */
  title: ComputedRef<string>;
  /** DOM id for the page title heading (aria-labelledby). */
  titleId: string;
  /** Import/Export menu props when the page enables IO; null otherwise. */
  titleActionsIo: ComputedRef<PageTitleActionsIo | null>;
  /** True while List/Kanban view chrome hosts the page title (+ IO). */
  titleHostedByView: ShallowRef<boolean>;
  claimPageTitle: () => void;
  releasePageTitle: () => void;
};

/** Stable across `vi.resetModules()` so provide/inject still match in unit tests. */
export const PageContextKey: InjectionKey<PageContext> = Symbol.for('choysum.pageContext');

export function providePageContext(options: {
  store: MaybeRefOrGetter<unknown | null | undefined>;
  title?: MaybeRefOrGetter<string | undefined>;
  titleId?: string;
  titleActionsIo?: MaybeRefOrGetter<PageTitleActionsIo | null | undefined>;
}) {
  const store = computed(() => (toValue(options.store) ?? null) as unknown | null);
  const actionTarget = shallowRef<PageActionTarget | null>(null);
  const title = computed(() => String(toValue(options.title) ?? '').trim());
  const titleId = options.titleId ?? '';
  const titleActionsIo = computed(() => (toValue(options.titleActionsIo) ?? null) as PageTitleActionsIo | null);
  const titleHostedByView = shallowRef(false);

  function registerActionTarget(target: PageActionTarget) {
    actionTarget.value = target;
  }

  function unregisterActionTarget(target: PageActionTarget) {
    if (actionTarget.value === target) {
      actionTarget.value = null;
    }
  }

  function claimPageTitle() {
    titleHostedByView.value = true;
  }

  function releasePageTitle() {
    titleHostedByView.value = false;
  }

  const ctx: PageContext = {
    store,
    actionTarget,
    registerActionTarget,
    unregisterActionTarget,
    title,
    titleId,
    titleActionsIo,
    titleHostedByView,
    claimPageTitle,
    releasePageTitle,
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

/**
 * List/Kanban view chrome hosts the page title (+ IO menu) so Page does not
 * render a separate title band above the action bar.
 */
export function useHostPageTitleInView(enabled: MaybeRefOrGetter<boolean>) {
  const ctx = inject(PageContextKey, null);
  let claimed = false;

  function shouldHost(): boolean {
    if (!ctx || !toValue(enabled)) return false;
    return !!ctx.title.value || !!ctx.titleActionsIo.value;
  }

  function sync() {
    if (!ctx) return;
    const on = shouldHost();
    if (on && !claimed) {
      ctx.claimPageTitle();
      claimed = true;
    } else if (!on && claimed) {
      ctx.releasePageTitle();
      claimed = false;
    }
  }

  // Claim during setup (parent Page already provided ctx) so the first paint
  // hides the page title band and drops top inset — not only after mount.
  sync();
  onMounted(sync);
  watch(
    () => [toValue(enabled), ctx?.title.value, ctx?.titleActionsIo.value] as const,
    sync,
  );

  onBeforeUnmount(() => {
    if (claimed && ctx) {
      ctx.releasePageTitle();
      claimed = false;
    }
  });

  return {
    pageTitle: computed(() => (shouldHost() ? (ctx?.title.value ?? '') : '')),
    pageTitleId: computed(() => ctx?.titleId ?? ''),
    pageTitleActionsIo: computed(() => (shouldHost() ? (ctx?.titleActionsIo.value ?? null) : null)),
  };
}
