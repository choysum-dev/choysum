// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type ComputedRef, type Ref } from 'vue';
import type { RouteLocationRaw, Router } from 'vue-router';
import { ChoyMessage } from './useChoyMessage';
import { confirmChoyAction } from './confirmChoyAction';
import type { BaseModel, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { createListController } from '@/web/web/controllers/listController';

type ListController = ReturnType<typeof createListController>;

export type UseListViewActionsOpts<T extends BaseModel> = {
  store: WebModelStore<T>;
  controller: ListController;
  router: Router;
  resolvedCreateAction: ComputedRef<RouteLocationRaw | string | undefined>;
  selectedItems: Ref<any[]>;
  items: ComputedRef<any[]>;
  selection: { clear: () => void };
  emitCancelable: (event: string, payload?: any) => Promise<boolean>;
  emit: (event: string, payload?: any) => void;
  loadData: () => Promise<void>;
  afterLayoutRecompute: () => Promise<unknown>;
  _t: (key: string, ...args: any[]) => string;
};

export function useListViewActions<T extends BaseModel>(opts: UseListViewActionsOpts<T>) {
  const deleteLoading = ref(false);

  async function handleRefresh() {
    const ok = await opts.emitCancelable('before-refresh');
    if (!ok) return;

    try {
      await opts.controller.apply();
      ChoyMessage.success(opts._t('List data refreshed'));
      const total = Number(((opts.store.state as any).result?.total ?? 0) as any) || 0;
      opts.emit('refresh-success', { items: (opts.items.value as any) || [], total });
      await opts.afterLayoutRecompute();
    } catch (e: unknown) {
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: 'refresh', error: err });
      ChoyMessage.error(opts._t('Failed to refresh list'));
    }
  }

  async function handleCreate() {
    if (!opts.resolvedCreateAction.value) return;
    try {
      await opts.router.push(opts.resolvedCreateAction.value);
    } catch (e) {
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: 'create', error: err });
    }
  }

  async function handleDelete() {
    if (opts.selectedItems.value.length === 0) return;
    const count = opts.selectedItems.value.length;
    try {
      await confirmChoyAction(
        opts._t('Are you sure you want to delete the selected %s record(s)? This action cannot be undone.', count),
        opts._t('Confirm delete'),
        {
          confirmText: opts._t('Delete'),
          cancelText: opts._t('Cancel'),
          destructive: true,
        }
      );
      const ids = (opts.selectedItems.value as any[])
        .map((r: any) => {
          if (r?.type === 'record') return r?.record?.Id;
          if (r?.kind === 'record') return r?.payload?.Id;
          const rec = r?.payload ?? r?.record ?? r;
          return rec?.Id;
        })
        .filter((id: any) => id != null) as string[];

      if (ids.length === 0) {
        ChoyMessage.warning(opts._t('Selected records are missing valid IDs'));
        return;
      }

      const ok = await opts.emitCancelable('before-delete', { ids });
      if (!ok) return;

      deleteLoading.value = true;

      await opts.store.Delete(['Id', 'in', ids] as QueryCondition<T>);
      ChoyMessage.success(opts._t('Successfully deleted %s record(s)', ids.length));
      opts.selection.clear();
      opts.selectedItems.value = [];
      await opts.loadData();
      opts.emit('delete-success', { ids });
    } catch (e) {
      if (e === 'cancel' || e === 'dismiss') return;
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: 'delete', error: err });
      ChoyMessage.error(opts._t('Delete failed'));
    } finally {
      deleteLoading.value = false;
    }
  }

  return {
    deleteLoading,
    handleRefresh,
    handleCreate,
    handleDelete,
  };
}
