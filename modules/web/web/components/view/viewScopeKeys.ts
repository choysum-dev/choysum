// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computed,
  inject,
  isRef,
  provide,
  ref,
  unref,
  type ComputedRef,
  type InjectionKey,
  type Ref,
} from 'vue';
import {
  getViewKind,
  registerBuiltinViewKinds,
  type FieldPresentation,
  type ViewKindDef,
  type ViewKindId,
} from './viewKinds';

export type ViewMode = 'display' | 'edit' | 'create';

/** Alias for ViewKindId (historical name used in some call sites). */
export type ViewContainer = ViewKindId;

export type ViewScopeValue = {
  kind: Ref<ViewKindId>;
  mode: Ref<ViewMode>;
  fieldPrefix: Ref<string | null | undefined>;
  kindDef: ComputedRef<ViewKindDef>;
  fieldPresentation: ComputedRef<FieldPresentation>;
};

export const VIEW_SCOPE_KEY: InjectionKey<ViewScopeValue> = Symbol('choy.view-scope');

export type ProvideViewScopeOptions = {
  kind: Ref<ViewKindId> | ViewKindId;
  mode?: Ref<ViewMode> | ViewMode;
  fieldPrefix?: Ref<string | null | undefined> | string | null;
};

function asRef<T>(value: Ref<T> | T): Ref<T> {
  return isRef(value) ? value : (ref(value) as Ref<T>);
}

const FALLBACK_KIND_DEF: ViewKindDef = {
  id: 'unknown',
  fieldPresentation: 'none',
  defaultMode: 'display',
};

/**
 * Provide view environment for descendants (Form/List/Kanban/… or ChoyViewScope).
 * Ensures builtin kinds are registered once.
 * When `fieldPrefix` is omitted, inherits the parent scope prefix (if any).
 */
export function useProvideViewScope(opts: ProvideViewScopeOptions): ViewScopeValue {
  registerBuiltinViewKinds();
  const parent = inject(VIEW_SCOPE_KEY, null);
  const kind = asRef(opts.kind) as Ref<ViewKindId>;
  const mode = (opts.mode !== undefined
    ? asRef(opts.mode)
    : computed(() => getViewKind(kind.value).defaultMode || 'display')) as Ref<ViewMode>;
  const fieldPrefix = computed(() => {
    if (opts.fieldPrefix !== undefined) {
      const v = unref(opts.fieldPrefix as Ref<string | null | undefined> | string | null);
      // Explicit undefined (optional prop not set) still inherits the parent prefix.
      if (v === undefined) return parent?.fieldPrefix.value ?? null;
      return v;
    }
    return parent?.fieldPrefix.value ?? null;
  }) as Ref<string | null | undefined>;

  const kindDef = computed(() => getViewKind(kind.value));
  const fieldPresentation = computed(() => kindDef.value.fieldPresentation);

  const scope: ViewScopeValue = {
    kind,
    mode,
    fieldPrefix,
    kindDef,
    fieldPresentation,
  };
  provide(VIEW_SCOPE_KEY, scope);
  return scope;
}

/**
 * Inject the nearest view scope. When missing, returns a display/none fallback
 * without warning on the synthetic "unknown" kind id.
 */
export function useViewScope(): ViewScopeValue {
  registerBuiltinViewKinds();
  const injected = inject(VIEW_SCOPE_KEY, null);
  if (injected) return injected;

  const kind = ref<ViewKindId>('unknown');
  const mode = ref<ViewMode>('display');
  const fieldPrefix = ref<string | null>(null);
  const kindDef = computed(() => FALLBACK_KIND_DEF);
  const fieldPresentation = computed(() => FALLBACK_KIND_DEF.fieldPresentation);
  return { kind, mode, fieldPrefix, kindDef, fieldPresentation };
}
