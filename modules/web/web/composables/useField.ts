// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Provides unified field access, view mapping, array helpers, and field/metric registration.
 * - Depends only on WebModelStore identifiers and metadata.
 * - Registers field paths and metrics on mount and unregisters them on unmount.
 * - Prefers the injected form root and falls back to a read-only empty record outside forms.
 */

import { computed, inject, onMounted, onUnmounted, ref, watch, watchEffect, type ComputedRef, type Ref, type WritableComputedRef } from 'vue';
import { createStoreByModel, getStoreFactoryRegistryVersion } from '@/web/web/stores/registry';
import { getFieldMetadataView, type WebModelStore, type FieldMetadata } from '@/web/web/stores/modelStore';
import { registerFieldPath, unregisterFieldPath } from '@/web/web/query/utils/registry/field';
import { registerMetric, unregisterMetric } from '@/web/web/query/utils/registry/metric';
import type { MetricSpec } from '@/web/web/query/utils/registry/metric';
import { ComponentScopeManager } from '@/web/web/stores/storeScopeManager/component';
import type { ViewMode, ViewContainer } from '@/web/web/components/view/OViewScope.vue';

/* ===================== Environment and core helpers ===================== */

/**
 * Environment flags exposed by useField.
 */
export interface FieldEnv {
  readonly isForm: boolean;
  readonly isEditMode: boolean;
  readonly viewMode: ViewMode;
  readonly fieldPrefix: string | null;
}

const getByPath = (obj: any, path: string) =>
  String(path)
    .split('.')
    .filter(Boolean)
    .reduce((a, k) => (a == null ? a : a[k]), obj);

const setByPath = (obj: any, path: string, v: any) => {
  if (!obj || typeof obj !== 'object') return;
  const segs = String(path).split('.').filter(Boolean);
  let cur = obj;
  for (let i = 0; i < segs.length - 1; i++) {
    const k = segs[i]!;
    if (!cur[k] || typeof cur[k] !== 'object') cur[k] = {};
    cur = cur[k];
  }
  cur[segs[segs.length - 1]!] = v;
};

/* ===================== Public contracts ===================== */

/**
 * Public API returned by useField.
 */
export interface UseField<T = any, V = any> {
  env: FieldEnv;
  prop: string;
  meta: FieldMetadata | undefined;

  // Field references.
  fieldRef(): WritableComputedRef<V>;
  fieldRefOf(row: T): WritableComputedRef<V>;

  // Record reference, preferring the form context when available.
  recordRef(): ComputedRef<T>;

  // Bulk-register extra field paths used by sibling or relation fields.
  registerFields(paths: string | string[]): void;

  // Relation model store when the field resolves to a relation.
  relationStore?: WebModelStore<any>;

  // Read-only store passthrough for callers that need the store reference.
  store?: WebModelStore<any>;

  // View mapping helpers.
  asView<View = V, From = View>(opts?: {
    toView?: (raw: V) => View;
    fromView?: (v: View) => From;
    getFieldRef?: () => WritableComputedRef<V>;
    getFieldRefOf?: (row: any) => WritableComputedRef<V>;
  }): {
    fieldValue: () => WritableComputedRef<View>;
    fieldValueOfRow: (row: any) => WritableComputedRef<View>;
  };

  // Mutable array helpers for collection-like fields.
  asMutableArray<Item = V extends (infer E)[] ? E : never>(): {
    insertItem: (item: Item, index?: number) => void;
    removeItemAt: (index: number) => void;
    moveItem: (from: number, to: number) => void;
    clearItems: () => void;
    getItems: () => Item[];
  };

  // Compatibility placeholder currently maintained by OFormView.
  emitOnchange?: (fieldPaths?: string | string[], opts?: { withCompute?: boolean; maxIterations?: number }) => Promise<null>;
}

/* ===================== Metric registration types and helpers ===================== */

/**
 * Aggregate functions shared with backend query execution.
 */
export type AggregateFunction = 'sum' | 'avg' | 'min' | 'max' | 'count' | 'count_distinct';

/**
 * Generic aggregate declaration accepted by field metrics.
 */
export type AggProp = AggregateFunction | { agg: AggregateFunction; alias?: string };

/** Narrowed aggregate declaration for a constrained aggregate set. */
export type NarrowAggProp<Allowed extends AggregateFunction> = Allowed | { agg: Allowed; alias?: string };
export type NumericAggFns = Exclude<AggregateFunction, 'count'>;
export type NonNumericAggFns = Extract<AggregateFunction, 'count_distinct'>;
export type TemporalAggFns = NonNumericAggFns;

// Generate a metric alias, preferring an explicit alias first.
function metricAliasOf(field: string, agg?: AggProp): string | null {
  if (!agg) return null;
  if (typeof agg === 'string') return agg === 'count' ? '__count' : `${field}__${agg}`;
  if (!agg.agg) return null;
  if (agg.agg === 'count') return '__count';
  const alias = typeof agg.alias === 'string' && agg.alias.trim() ? agg.alias.trim() : undefined;
  return alias || `${field}__${agg.agg}`;
}

/* ===================== Main entry ===================== */

/**
 * Creates field helpers for a store-backed field path.
 */
export function useField<T = any, P extends string = string, V = any>(opts: {
  store?: WebModelStore<any>;
  prop: P;
  autoRegister?: boolean;
  agg?: AggProp | AggProp[];
}): UseField<T, V> {
  const { store, prop, autoRegister = true, agg } = opts;
  const suppressFieldRegistration = inject<boolean>('suppress-field-registration', false);
  const altRegistrationStore = inject<any>('alt-field-registration-store', null);

  const resolveAltStore = () => {
    if (!altRegistrationStore) return null;
    return (altRegistrationStore as any).value ?? altRegistrationStore;
  };

  const allowFieldRegistration = () => suppressFieldRegistration !== true || !!resolveAltStore();
  const storeId = store?.storeId;
  const registeredPaths = new Set<string>();

  const currentTargetStore = () => (suppressFieldRegistration === true ? resolveAltStore() : store);
  const currentTargetStoreId = () => currentTargetStore()?.storeId;

  // View context provided by OViewScope or OFormView.
  const viewContainer = inject<Ref<ViewContainer>>('view-container', ref<ViewContainer>('Form'));
  const viewModeRef = inject<Ref<ViewMode>>('view-mode', ref<ViewMode>('display'));
  const fieldPrefixRef = inject<Ref<string | null>>('field-prefix', ref<string | null>(null));
  const formRoot = inject<any>('form-root', null); // { draft, getField, setField, ... }

  const env: FieldEnv = {
    get isForm() {
      return viewContainer.value === 'Form';
    },
    get isEditMode() {
      // Nested list editors remain editable while the surrounding form is active.
      const inFormContext = !!formRoot || viewContainer.value === 'Form';
      return inFormContext && (viewModeRef.value === 'edit' || viewModeRef.value === 'create');
    },
    get viewMode() {
      return viewModeRef.value;
    },
    get fieldPrefix() {
      return fieldPrefixRef.value;
    },
  };

  // Resolve metadata chains and relation stores, retrying when factories register late.
  const segs = String(prop).split('.').filter(Boolean);
  const relationScope = new ComponentScopeManager();
  const leafMetaRef = ref<FieldMetadata | undefined>(undefined);
  const relationStoreRef = ref<WebModelStore<any> | undefined>(undefined);
  const storeFactoryRegistryVersion = getStoreFactoryRegistryVersion();

  const resolveLeafMetaAndRelationStore = () => {
    let owningStore: WebModelStore<any> | undefined = store;
    let leafMeta: FieldMetadata | undefined;

    for (let i = 0; i < segs.length; i++) {
      const seg = segs[i]!;
      const meta = (owningStore?.fieldsMetadata as any)?.[seg] as FieldMetadata | undefined;
      const view = getFieldMetadataView(meta);
      leafMeta = meta;
      const isLeaf = i === segs.length - 1;
      if (!isLeaf && view.isRelation && view.relationModel) {
        try {
          owningStore = createStoreByModel(view.relationModel, { scopeManager: relationScope });
        } catch {
          // Stop current attempt; watchEffect will retry after factory registry updates.
          leafMeta = undefined;
          break;
        }
      }
    }

    let relationStore: WebModelStore<any> | undefined;
    const leafView = getFieldMetadataView(leafMeta);
    if (leafView.isRelation && leafView.relationModel) {
      try {
        relationStore = createStoreByModel(leafView.relationModel, { scopeManager: relationScope });
      } catch {
        relationStore = undefined;
      }
    }

    leafMetaRef.value = leafMeta;
    relationStoreRef.value = relationStore;
  };

  watchEffect(() => {
    void storeFactoryRegistryVersion.value;
    resolveLeafMetaAndRelationStore();
  });

  // Form root record reference that falls back to an empty read-only object.
  const recordRef = () =>
    computed<T>(() => {
      const d = formRoot?.draft;
      return (d ?? ({} as T)) as T;
    });

  // Field ref bound to the current form context.
  const fieldRefImpl = computed<V>({
    get() {
      if (formRoot?.getField) {
        return formRoot.getField(String(prop)) as V;
      }
      const rec = recordRef().value as any;
      return getByPath(rec, String(prop)) as V;
    },
    set(v: V) {
      // Outside form context the field becomes effectively read-only.
      if (formRoot?.setField) {
        formRoot.setField(String(prop), v);
      } else if (formRoot?.draft) {
        setByPath(formRoot.draft, String(prop), v);
      }
    },
  }) as WritableComputedRef<V>;

  // Field ref bound to an explicit row object.
  function fieldRefOfImpl(row: T) {
    // Strip the field prefix so row-level refs operate on relative paths.
    return computed<V>({
      get() {
        const prefix = env.fieldPrefix ? String(env.fieldPrefix) : null;
        let path = String(prop);
        if (prefix) {
          if (path.startsWith(prefix + '.')) path = path.slice(prefix.length + 1);
          else if (path === prefix) path = '';
        }
        return path ? (getByPath(row as any, path) as V) : (row as any as unknown as V);
      },
      set(v: V) {
        const prefix = env.fieldPrefix ? String(env.fieldPrefix) : null;
        let path = String(prop);
        if (prefix) {
          if (path.startsWith(prefix + '.')) path = path.slice(prefix.length + 1);
          else if (path === prefix) path = '';
        }
        if (path) setByPath(row as any, path, v);
      },
    }) as WritableComputedRef<V>;
  }

  // Register field paths for this field and any extra dependent paths.
  const normalizeRegistrationPath = (path: string) => {
    let p = String(path || '').trim();
    // When registration is redirected to an alternate store (e.g., relationStore inside m2m slots),
    // strip the field prefix so the target store receives relative field names.
    if (suppressFieldRegistration === true) {
      const prefix = fieldPrefixRef.value ? String(fieldPrefixRef.value) : '';
      if (prefix) {
        if (p === prefix) p = '';
        else if (p.startsWith(prefix + '.')) p = p.slice(prefix.length + 1);
      }
    }
    return p;
  };

  function doRegister(path: string) {
    if (!allowFieldRegistration()) return;
    const p = normalizeRegistrationPath(path);
    if (!p) return;
    registeredPaths.add(p);
    const targetStoreId = currentTargetStoreId();
    if (!targetStoreId) return;
    registerFieldPath(targetStoreId, p);
  }
  function doUnregister(path: string) {
    if (!allowFieldRegistration()) return;
    const p = normalizeRegistrationPath(path);
    if (!p) return;
    const targetStoreId = currentTargetStoreId();
    if (targetStoreId) unregisterFieldPath(targetStoreId, p);
    registeredPaths.delete(p);
  }
  function registerFields(paths: string | string[]) {
    if (!allowFieldRegistration()) return;
    const list = Array.isArray(paths) ? paths : [paths];
    list.forEach(doRegister);
  }

  // Register the primary field path.
  onMounted(() => {
    if (autoRegister && allowFieldRegistration()) doRegister(String(prop));
  });
  onUnmounted(() => {
    if (autoRegister && allowFieldRegistration()) doUnregister(String(prop));
  });

  // Re-sync registrations when the alternate target store becomes available late.
  watch(
    () => currentTargetStoreId(),
    (next: string | undefined, prev: string | undefined) => {
      if (!allowFieldRegistration()) return;
      if (next === prev) return;
      if (prev) {
        for (const p of registeredPaths) unregisterFieldPath(prev, p);
      }
      if (next) {
        for (const p of registeredPaths) registerFieldPath(next, p);
      }
    }
  );

  // Register requested metrics plus decimal scale helpers when needed.
  const registeredSpecs = ref<MetricSpec[]>([]);
  onMounted(() => {
    if (!storeId) return;

    // Register primary metrics, including array forms.
    if (agg) {
      const list = Array.isArray(agg) ? agg : [agg];
      for (const one of list) {
        const spec: MetricSpec = typeof one === 'string' ? { field: String(prop), agg: one } : { field: String(prop), agg: one.agg, alias: one.alias };
        registerMetric(storeId, spec);
        registeredSpecs.value.push(spec);
      }
    }

    // Add a max aggregation for decimal scale fields when available.
    try {
      const t = (leafMetaRef.value?.type || '').toLowerCase();
      const scaleField = (leafMetaRef.value as any)?.scaleField as string | undefined;
      if (t === 'decimal' && scaleField) {
        const segs2 = segs.slice();
        segs2[segs2.length - 1] = scaleField;
        const scalePath = segs2.join('.');
        const alias = `${scaleField}__max`;
        const scaleSpec: MetricSpec = { field: scalePath, agg: 'max', alias };
        registerMetric(storeId, scaleSpec);
        registeredSpecs.value.push(scaleSpec);
      }
    } catch {
      /* ignore */
    }
  });
  onUnmounted(() => {
    if (!storeId) return;
    for (const s of registeredSpecs.value) unregisterMetric(storeId, s);
    registeredSpecs.value = [];
  });

  // Build view-layer value mappings on top of the field ref.
  function asView<View = V, From = View>(m?: {
    toView?: (raw: V) => View;
    fromView?: (v: View) => From;
    getFieldRef?: () => WritableComputedRef<V>;
    getFieldRefOf?: (row: any) => WritableComputedRef<V>;
  }) {
    const getFieldRef = m?.getFieldRef ?? (() => fieldRefImpl);
    const getFieldRefOf = m?.getFieldRefOf ?? fieldRefOfImpl;
    const toView = m?.toView ?? ((r: V) => r as unknown as View);
    const fromView = m?.fromView ?? ((v: View) => v as unknown as From);

    const fieldValueRef = computed<View>({
      get: () => toView(getFieldRef().value),
      set: v => {
        const next = fromView(v) as unknown as V;
        getFieldRef().value = next;
      },
    }) as WritableComputedRef<View>;

    const fieldValue = () => fieldValueRef;

    const fieldValueOfRow = (row: T) =>
      computed<View>({
        get: () => toView(getFieldRefOf(row).value),
        set: v => {
          const next = fromView(v) as unknown as V;
          getFieldRefOf(row).value = next;
        },
      }) as WritableComputedRef<View>;

    return { fieldValue, fieldValueOfRow };
  }

  // Build convenience helpers for collection-like fields.
  function asMutableArray<Item = V extends (infer E)[] ? E : never>() {
    const itemsRef = computed<Item[]>(() => {
      const cur = fieldRefImpl.value as any;
      return Array.isArray(cur) ? (cur as Item[]) : ([] as Item[]);
    });

    function ensureArray() {
      if (!Array.isArray(fieldRefImpl.value as any)) {
        fieldRefImpl.value = [] as unknown as V;
      }
    }

    function insertItem(item: Item, index?: number) {
      ensureArray();
      const arr = fieldRefImpl.value as unknown as Item[];
      const pos = typeof index === 'number' && index >= 0 ? Math.min(index, arr.length) : arr.length;
      arr.splice(pos, 0, item);
      fieldRefImpl.value = arr as unknown as V;
    }

    function removeItemAt(index: number) {
      const arr = fieldRefImpl.value as unknown as Item[];
      if (!Array.isArray(arr)) return;
      if (index < 0 || index >= arr.length) return;
      arr.splice(index, 1);
      fieldRefImpl.value = arr as unknown as V;
    }

    function moveItem(from: number, to: number) {
      const arr = fieldRefImpl.value as unknown as Item[];
      if (!Array.isArray(arr)) return;
      if (from < 0 || from >= arr.length || to < 0 || to >= arr.length) return;
      const [it] = arr.splice(from, 1);
      arr.splice(to, 0, it);
      fieldRefImpl.value = arr as unknown as V;
    }

    function clearItems() {
      fieldRefImpl.value = [] as unknown as V;
    }

    return {
      insertItem,
      removeItemAt,
      moveItem,
      clearItems,
      getItems: () => itemsRef.value,
    };
  }

  return {
    env,
    prop: String(prop),
    get meta() {
      return leafMetaRef.value;
    },
    fieldRef: () => fieldRefImpl,
    fieldRefOf: fieldRefOfImpl,
    recordRef,
    registerFields,
    get relationStore() {
      return relationStoreRef.value;
    },
    asView,
    asMutableArray,
    store,
    emitOnchange: async () => null,
  } as UseField<T, V>;
}

/* ---------- Standalone fields without a backing store ---------- */

/**
 * Creates field helpers around a standalone ref without a store.
 */
export function useStandaloneField<V, T = any>(opts: {
  value: Ref<V>;
  meta?: Partial<FieldMetadata>;
  prop?: string;
  env?: Partial<FieldEnv>;
  record?: T;
}): UseField<T, V> {
  const prop = String(opts.prop ?? 'value');

  const env: FieldEnv = {
    isForm: false,
    isEditMode: true,
    viewMode: 'edit',
    fieldPrefix: null,
    ...(opts.env || {}),
  };

  const fieldRefImpl = computed<V>({
    get: () => opts.value.value as unknown as V,
    set: v => {
      (opts.value as Ref<V>).value = v as V;
    },
  }) as WritableComputedRef<V>;

  const fieldRef = () => fieldRefImpl;
  function fieldRefOf(_row: T) {
    return fieldRefImpl;
  }

  const _recordRef = computed<T>(() => (opts.record as T) ?? ({} as T));
  const recordRef = () => _recordRef;

  function asView<View = V, From = View>(m?: {
    toView?: (raw: V) => View;
    fromView?: (v: View) => From;
    getFieldRef?: () => WritableComputedRef<V>;
    getFieldRefOf?: (row: any) => WritableComputedRef<V>;
  }) {
    const getFieldRef = m?.getFieldRef ?? (() => fieldRefImpl);
    const getFieldRefOf = m?.getFieldRefOf ?? fieldRefOf;
    const toView = m?.toView ?? ((r: V) => r as unknown as View);
    const fromView = m?.fromView ?? ((v: View) => v as unknown as From);

    const fieldValueRef = computed<View>({
      get: () => toView(getFieldRef().value),
      set: v => {
        const next = fromView(v) as unknown as V;
        getFieldRef().value = next;
      },
    }) as WritableComputedRef<View>;

    const fieldValue = () => fieldValueRef;

    const fieldValueOfRow = (_row: T) =>
      computed<View>({
        get: () => toView(getFieldRefOf(_row).value),
        set: v => {
          const next = fromView(v) as unknown as V;
          getFieldRefOf(_row).value = next;
        },
      }) as WritableComputedRef<View>;

    return { fieldValue, fieldValueOfRow };
  }

  function asMutableArray<Item = V extends (infer E)[] ? E : never>() {
    const itemsRef = computed<Item[]>(() => {
      const cur = fieldRefImpl.value as unknown as any[];
      return Array.isArray(cur) ? (cur as Item[]) : ([] as Item[]);
    });

    function ensureArray() {
      const cur = fieldRefImpl.value as unknown as any[];
      if (!Array.isArray(cur)) {
        (fieldRefImpl as WritableComputedRef<any>).value = [] as Item[];
      }
      return fieldRefImpl.value as unknown as Item[];
    }

    const insertItem = (item: Item, index?: number) => {
      const arr = ensureArray();
      if (index == null || index < 0 || index > arr.length) arr.push(item);
      else arr.splice(index, 0, item);
    };
    const removeItemAt = (index: number) => {
      const arr = ensureArray();
      if (index >= 0 && index < arr.length) arr.splice(index, 1);
    };
    const moveItem = (from: number, to: number) => {
      const arr = ensureArray();
      if (from === to || from < 0 || from >= arr.length || to < 0 || to >= arr.length) return;
      const [it] = arr.splice(from, 1);
      arr.splice(to, 0, it);
    };
    const clearItems = () => {
      const arr = ensureArray();
      arr.splice(0, arr.length);
    };

    return {
      insertItem,
      removeItemAt,
      moveItem,
      clearItems,
      getItems: () => itemsRef.value,
    };
  }

  return {
    env,
    prop,
    meta: (opts.meta as FieldMetadata | undefined) ?? undefined,
    fieldRef,
    fieldRefOf,
    recordRef,
    registerFields: () => {},
    asView,
    asMutableArray,
    relationStore: undefined,
    emitOnchange: async () => null,
  } as UseField<T, V>;
}
