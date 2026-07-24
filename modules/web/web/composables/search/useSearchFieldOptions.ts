// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, type ComputedRef } from 'vue';
import { getFieldMetadataView, type WebFieldMetadata, type WebModelStore } from '@/web/web/stores/modelStore';
import { getGlobalComposer } from '@/web/web/i18n';
import { resolveFieldLabel } from '@/web/web/composables/resolveFieldLabel';

export type SearchFieldOption = {
  prop: string;
  label: string;
  id?: string;
  type?: string;
};

function labelFor(store: WebModelStore<any>, prop: string, meta: Partial<WebFieldMetadata> | undefined): string {
  return resolveFieldLabel({
    prop,
    meta: meta as any,
    composer: getGlobalComposer(),
    fieldsGetTranslatedString: store.getFieldsGetTranslatedString?.(prop),
  });
}

export function sortSearchFieldOptions<T extends { prop: string; label: string; id?: string }>(
  items: T[],
  idOf?: (prop: string) => string
): T[] {
  return [...items].sort((a, b) => {
    const idA = a.id ?? idOf?.(a.prop) ?? '';
    const idB = b.id ?? idOf?.(b.prop) ?? '';
    const cmp = String(idA).localeCompare(String(idB), 'en', { sensitivity: 'base' });
    if (cmp !== 0) return cmp;
    return a.label.localeCompare(b.label, 'en', { sensitivity: 'base' });
  });
}

/** Fields shown in the custom-filter field picker (scalar + many-to-one / ref). */
export function isFilterableSearchField(prop: string, meta: Partial<WebFieldMetadata> | undefined): boolean {
  if (prop === 'DeletedAt') return false;
  if (prop === 'Id') return true;
  const view = getFieldMetadataView(meta as any);
  if (!view.isRelation) return true;
  const lowerType = String(meta?.type ?? '').toLowerCase();
  return lowerType === 'manytoone' || lowerType === 'manytooneref';
}

/** Fields eligible for Group by (excludes collections / json blobs). */
export function isGroupableSearchField(prop: string, meta: Partial<WebFieldMetadata> | undefined): boolean {
  if (prop === 'DeletedAt') return false;
  const t = String(meta?.type || '').toLowerCase();
  if (t === 'onetomany' || t === 'manytomany' || t === 'jsonobject') return false;
  return true;
}

export function listSearchFieldOptions(
  store: WebModelStore<any>,
  predicate: (prop: string, meta: Partial<WebFieldMetadata> | undefined) => boolean
): SearchFieldOption[] {
  const md = (store.fieldsMetadata ?? {}) as Record<string, Partial<WebFieldMetadata>>;
  const items: SearchFieldOption[] = [];
  for (const [prop, meta] of Object.entries(md)) {
    if (!predicate(prop, meta)) continue;
    items.push({
      prop,
      label: labelFor(store, prop, meta),
      id: meta?.id != null ? String(meta.id) : undefined,
      type: meta?.type != null ? String(meta.type) : undefined,
    });
  }
  return sortSearchFieldOptions(items);
}

export function useFilterableSearchFields(store: WebModelStore<any>): ComputedRef<Array<{ prop: string; label: string }>> {
  return computed(() =>
    listSearchFieldOptions(store, isFilterableSearchField).map(({ prop, label }) => ({ prop, label }))
  );
}

export function resolveSearchFieldLabel(store: WebModelStore<any> | undefined, prop: string): string {
  if (!store || !prop) return prop;
  const meta = (store.fieldsMetadata as any)?.[prop];
  return labelFor(store, prop, meta);
}
