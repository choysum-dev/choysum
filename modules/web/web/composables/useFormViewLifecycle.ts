// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, nextTick, toRaw, watch, type ComputedRef, type Ref } from 'vue';
import type { RouteLocationNormalizedLoaded, RouteLocationRaw, Router } from 'vue-router';
import { ChoyMessage } from './useChoyMessage';
import { confirmChoyAction } from './confirmChoyAction';
import { deepClonePreserve } from '@/core/utils/clone';
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import type { createFormController } from '@/web/web/controllers/formController';
import type { provideOnchange } from '@/web/web/composables/useOnchange';
import type { useOnchangeAggregation } from '@/web/web/composables/useOnchangeAggregation';

type FormController = ReturnType<typeof createFormController>;
type OnchangeCtrl = ReturnType<typeof provideOnchange>;
type OnchangeAgg = ReturnType<typeof useOnchangeAggregation>;

export type FormViewIdentityProps<T extends BaseModel> = Readonly<{
  recordId?: string;
  viewMode?: ViewMode;
  initialValues?: Partial<T>;
  onchangeSessionId?: string;
}>;

export type UseFormViewLifecycleOpts<T extends BaseModel> = {
  identity: FormViewIdentityProps<T>;
  route: RouteLocationNormalizedLoaded;
  router: Router;
  controller: FormController;
  resolvedResolveRecordIdFromRoute: ComputedRef<boolean>;
  resolvedShowMessages: ComputedRef<boolean>;
  resolvedCreateAction: ComputedRef<RouteLocationRaw | string | undefined>;
  viewMode: ComputedRef<ViewMode>;
  exposedFormData: ComputedRef<Partial<ClientModel<T>>>;
  onchangeCtrl: OnchangeCtrl;
  resetOnchangeAgg: OnchangeAgg['reset'];
  emitCancelable: (event: string, payload?: any) => Promise<boolean>;
  emit: (event: string, payload?: any) => void;
  _t: (key: string, ...args: any[]) => string;
};

function normalizeRecordId(raw: unknown): string | undefined {
  const s = String(raw ?? '').trim();
  if (!s || s === 'undefined' || s === 'null') return undefined;
  return s;
}

export function useFormViewLifecycle<T extends BaseModel>(opts: UseFormViewLifecycleOpts<T>) {
  function peekRouteRecordId(): string | undefined {
    if (!opts.resolvedResolveRecordIdFromRoute.value) {
      return undefined;
    }
    const route = opts.route;
    return normalizeRecordId(
      route.params.recordId ?? route.params.id ?? route.params.Id ?? route.query.recordId ?? route.query.id ?? route.query.Id
    );
  }

  const effectiveRecordId = computed(() => normalizeRecordId(opts.identity.recordId) ?? peekRouteRecordId());

  let initializeSeq = 0;

  async function initializeForm() {
    const seq = ++initializeSeq;
    const isStale = () => seq !== initializeSeq;
    try {
      const ok = await opts.emitCancelable('before-load');
      if (!ok || isStale()) return;
      opts.onchangeCtrl.reset();
      opts.resetOnchangeAgg();
      const normId = effectiveRecordId.value;
      if (normId) {
        await opts.controller.beginDisplay(normId);
        if (isStale()) return;
        opts.emit('mode-change', { mode: opts.controller.vm.mode as ViewMode });
        opts.emit('load-success', { record: (opts.controller.vm.original as any) || null });
      } else {
        await opts.controller.beginCreate(deepClonePreserve((opts.identity.initialValues || {}) as any));
        if (isStale()) return;
        if ((opts.identity.viewMode as any) === 'display') {
          (opts.controller.vm as any).mode = 'display';
        }
        opts.emit('mode-change', { mode: opts.controller.vm.mode as ViewMode });
        opts.emit('load-success', { record: null as any });
      }
      if (isStale()) return;
      opts.emit('change', { formData: toRaw(opts.exposedFormData.value) as any });
    } catch (e: any) {
      if (isStale()) return;
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: 'load', error: err });
    }
  }

  function bumpInitializeSeq() {
    initializeSeq += 1;
  }

  function handleEdit() {
    opts.controller.beginEdit();
    opts.onchangeCtrl.reset();
    opts.resetOnchangeAgg();
    opts.emit('mode-change', { mode: 'edit' });
  }

  async function handleCancel() {
    opts.onchangeCtrl.reset();
    opts.resetOnchangeAgg();
    if (opts.viewMode.value === 'create') {
      opts.router.back();
      return;
    }
    const id = effectiveRecordId.value;
    if (id) {
      try {
        await opts.controller.beginDisplay(id);
      } catch (e: any) {
        const err = e instanceof Error ? e : new Error(String(e));
        opts.emit('action-error', { action: 'load', error: err });
      }
    }
    opts.emit('mode-change', { mode: 'display' });
  }

  function handleReset() {
    opts.resetOnchangeAgg();
    opts.controller.reset();
    opts.onchangeCtrl.reset();
    opts.emit('reset', { formData: toRaw(opts.exposedFormData.value) as any });
  }

  async function handleCreate() {
    if (!opts.resolvedCreateAction.value) return;
    try {
      await opts.router.push(opts.resolvedCreateAction.value);
    } catch {}
  }

  async function handleRefresh() {
    const ok = await opts.emitCancelable('before-refresh');
    if (!ok) return;

    const id = effectiveRecordId.value;
    if (!id) {
      opts.emit('refresh-success', { record: null });
      return;
    }
    try {
      await opts.controller.beginDisplay(id);
      if (opts.resolvedShowMessages.value) ChoyMessage.success(opts._t('Refreshed'));
      opts.emit('refresh-success', { record: (opts.controller.vm.original || null) as any });
    } catch (e: any) {
      if (opts.resolvedShowMessages.value) ChoyMessage.error(opts._t('Refresh failed'));
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: 'refresh', error: err });
    }
  }

  async function handleCopy() {
    if (!(opts.controller.vm.original as any)) return;
    const cp = deepClonePreserve(opts.controller.vm.original as any);
    delete (cp as any).Id;
    await opts.controller.beginCreate(cp);
    opts.resetOnchangeAgg();
    opts.emit('mode-change', { mode: 'create' });
    opts.onchangeCtrl.reset();
    opts.emit('copy', { formData: toRaw(opts.exposedFormData.value) as any });
  }

  function handleDelete() {
    const currId = (opts.controller.vm.original as any)?.Id;
    if (!currId) return;
    confirmChoyAction(opts._t('Are you sure you want to delete the current record? This action cannot be undone.'), opts._t('Confirm delete'), {
      confirmText: opts._t('Delete'),
      cancelText: opts._t('Cancel'),
      destructive: true,
    })
      .then(async () => {
        const id = String(currId);
        const ok = await opts.emitCancelable('before-delete', { id });
        if (!ok) return;
        await opts.controller.delete();
        opts.resetOnchangeAgg();
        opts.emit('mode-change', { mode: 'display' });
        ChoyMessage.success(opts._t('Record deleted'));
        let defaultPrevented = false;
        opts.emit('delete-success', {
          id,
          preventDefault: () => {
            defaultPrevented = true;
          },
        });

        if (!defaultPrevented) {
          const currentRoute = opts.router.currentRoute.value;
          const pathSegments = currentRoute.path.split('/').filter(Boolean);
          if (pathSegments.length > 0) {
            const listPath = '/' + pathSegments.slice(0, -1).join('/');
            try {
              const breadcrumbStore = useBreadcrumbStore();
              const stack = breadcrumbStore.breadcrumbStack;
              if (stack.length > 0 && stack[stack.length - 1].path === currentRoute.path) {
                stack.pop();
              }
            } catch (e) {
              console.warn('Failed to update breadcrumb', e);
            }
            opts.router.replace(listPath);
          }
        }
      })
      .catch((e: unknown) => {
        if (e === 'cancel' || e === 'dismiss') return;
        ChoyMessage.error(opts._t('Delete failed'));
        const err = e instanceof Error ? e : new Error(String(e));
        opts.emit('action-error', { action: 'delete', error: err });
      });
  }

  function watchFormIdentity(onAfterInit?: () => void) {
    watch(
      () =>
        [
          opts.identity.recordId,
          opts.identity.viewMode,
          opts.identity.onchangeSessionId,
          opts.resolvedResolveRecordIdFromRoute.value ? String(opts.route.name ?? '') : '',
          opts.resolvedResolveRecordIdFromRoute.value ? peekRouteRecordId() ?? '' : '',
        ] as const,
      async () => {
        bumpInitializeSeq();
        await nextTick();
        await initializeForm();
        if ((opts.identity.viewMode as any) === 'edit') opts.controller.beginEdit();
        onAfterInit?.();
      },
      { immediate: true, flush: 'sync' }
    );
  }

  return {
    effectiveRecordId,
    initializeForm,
    bumpInitializeSeq,
    handleEdit,
    handleCancel,
    handleReset,
    handleCreate,
    handleRefresh,
    handleCopy,
    handleDelete,
    watchFormIdentity,
  };
}
