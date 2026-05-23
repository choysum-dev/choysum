// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onUnmounted } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  getOperatorOptions as baseGetOperatorOptions,
  isNullOperator as baseIsNull,
  requiresValue as baseRequires,
  defaultValueFor as baseDefault,
} from '@/web/web/query/utils/filter/operators';
import type { OperatorOption } from '@/web/web/query/utils/filter/operators';

export function useFilterEditorBindings(store: WebModelStore<any>) {
  function metaTypeOf(field?: string): string {
    if (!field) return '';
    const md = (store as any)?.fieldsMetadata || {};
    return String(md[field]?.type || '').toLowerCase();
  }

  function isTreeModel(): boolean {
    const md = (store as any)?.fieldsMetadata || {};
    return !!md['ParentPath'];
  }

  function relationModelOf(field?: string): string | undefined {
    if (!field) return undefined;
    const md = (store as WebModelStore<any>).fieldsMetadata || {};
    const m = md[field] || {};
    return m.relationModel;
  }

  const relStoreCache = new Map<string, any>();

  // Destroy cached relation stores on unmount to avoid leaks.
  onUnmounted(() => {
    relStoreCache.forEach(s => {
      if (s && typeof s.destroy === 'function') {
        s.destroy();
      }
    });
    relStoreCache.clear();
  });

  function relationStoreOf(field?: string): any {
    if (!field) return undefined;
    const s: any = store as any;
    const md = (s?.fieldsMetadata || {}) as Record<string, any>;
    const meta = md[field];
    const modelName = relationModelOf(field) || 'Relation';

    const cacheKey = `${s.storeId}::${field}::Filter`;
    if (relStoreCache.has(cacheKey)) return relStoreCache.get(cacheKey);

    if (meta && meta.relationModel) {
      try {
        const rs = createStoreByModel(meta.relationModel, { scopeManager: undefined });
        if (rs) {
          relStoreCache.set(cacheKey, rs);
          return rs;
        }
      } catch {}
    }

    if (typeof s.getRelationStore === 'function') {
      const rs = s.getRelationStore(field);
      if (rs) {
        relStoreCache.set(cacheKey, rs);
        return rs;
      }
    }

    if (modelName) {
      const tryGet = (obj: any) => (typeof obj?.getStore === 'function' ? obj.getStore(modelName) : undefined);
      const rs = tryGet(s) || tryGet(s.rootStore) || tryGet(s.pool);
      if (rs) {
        relStoreCache.set(cacheKey, rs);
        return rs;
      }
    }
    return undefined;
  }

  function isTreeManyToOne(field?: string): boolean {
    if (!field) return false;
    const md = (store as any)?.fieldsMetadata || {};
    const meta = md[field] || {};
    return String(meta?.type || '').toLowerCase() === 'manytoone' && !!meta?.relationModelParentField;
  }

  function getOperatorOptionsForField(field?: string): OperatorOption[] {
    if (!field) return baseGetOperatorOptions('');
    if (field === 'Id' && isTreeModel()) {
      const base = baseGetOperatorOptions(metaTypeOf(field));
      const vals = new Set(base.map(o => o.value));
      for (const v of ['child_of', 'parent_of']) if (!vals.has(v)) base.push({ value: v, label: v } as any);
      return base;
    }
    const t = metaTypeOf(field);
    if (t === 'manytoone') {
      const base = baseGetOperatorOptions('manytoone');
      if (isTreeManyToOne(field)) {
        const vals = new Set(base.map(o => o.value));
        for (const v of ['child_of', 'parent_of']) if (!vals.has(v)) base.push({ value: v, label: v } as any);
      }
      return base;
    }
    return baseGetOperatorOptions(t);
  }

  return {
    metaTypeOf,
    isTreeModel,
    relationModelOf,
    relationStoreOf,
    isTreeManyToOne,
    getOperatorOptionsForField,
    isNullOperator: (op?: string) => baseIsNull(op),
    requiresValue: (op?: string) => baseRequires(op),
    defaultValueFor: (t?: string) => baseDefault(t),
  } as const;
}
