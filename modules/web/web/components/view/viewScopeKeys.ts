// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  computed,
  inject,
  isRef,
  provide,
  ref,
  toRef,
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

/**
 * Provide view environment for descendants (Form/List/Kanban/… or ChoyViewScope).
 * Ensures builtin kinds are registered once.
 */
export function useProvideViewScope(opts: ProvideViewScopeOptions): ViewScopeValue {
  registerBuiltinViewKinds();
  const kind = asRef(opts.kind) as Ref<ViewKindId>;
  const mode = (opts.mode !== undefined
    ? asRef(opts.mode)
    : computed(() => getViewKind(kind.value).defaultMode || 'display')) as Ref<ViewMode>;
  const fieldPrefix = (opts.fieldPrefix !== undefined
    ? asRef(opts.fieldPrefix)
    : toRef(() => null as string | null)) as Ref<string | null | undefined>;

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
 * (no silent Form layout).
 */
export function useViewScope(): ViewScopeValue {
  registerBuiltinViewKinds();
  const injected = inject(VIEW_SCOPE_KEY, null);
  if (injected) return injected;

  const kind = toRef(() => 'unknown' as ViewKindId);
  const mode = toRef(() => 'display' as ViewMode);
  const fieldPrefix = toRef(() => null as string | null);
  const kindDef = computed(() => getViewKind(kind.value));
  const fieldPresentation = computed(() => kindDef.value.fieldPresentation);
  return { kind, mode, fieldPrefix, kindDef, fieldPresentation };
}
