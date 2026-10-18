// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, nextTick, toRaw, type ComputedRef, type Ref } from 'vue';
import type { Router } from 'vue-router';
import { ChoyMessage } from './useChoyMessage';
import type { ClientModel, BaseModel } from '@/core/rpc';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import type {
  FormSubmitMode,
  FormSubmitHandler,
  FormSubmitOutcome,
} from '@/web/web/components/view/formViewTypes';
import type { createFormController } from '@/web/web/controllers/formController';
import type { useOnchangeAggregation } from '@/web/web/composables/useOnchangeAggregation';
import type { provideOnchange } from '@/web/web/composables/useOnchange';

type FormController = ReturnType<typeof createFormController>;
type OnchangeCtrl = ReturnType<typeof provideOnchange>;
type OnchangeAgg = ReturnType<typeof useOnchangeAggregation>;

export type UseFormViewSubmitOpts<T extends BaseModel> = {
  controller: FormController;
  viewMode: ComputedRef<ViewMode>;
  loading: ComputedRef<boolean>;
  exposedFormData: ComputedRef<Partial<ClientModel<T>>>;
  fieldClientValidators: Map<string, () => Promise<string>>;
  fieldErrors: Ref<Map<string, string>>;
  resolvedShowMessages: ComputedRef<boolean>;
  submitHandler?: FormSubmitHandler<T>;
  onchangeCtrl: OnchangeCtrl;
  resetOnchangeAgg: OnchangeAgg['reset'];
  emitCancelable: (event: string, payload?: any) => Promise<boolean>;
  emit: (event: string, payload?: any) => void;
  router: Router;
  detailsTitle: { src: string };
  _t: (key: string, ...args: any[]) => string;
};

/** Path to open after a successful create, including copy that still sits on the source record URL. */
export function formCreateSuccessPath(path: string, newId: string, currentRecordId?: string): string | null {
  const id = String(newId || '').trim();
  if (!id) return null;
  // Trailing slashes (e.g. /app/items/rec-1/) must not block the copy→saved replace.
  const normalized = String(path || '').replace(/\/+$/, '') || '/';
  if (normalized.endsWith('/new')) return normalized.slice(0, -'new'.length) + id;
  const current = String(currentRecordId || '').trim();
  if (!current || current === id) return null;
  if (!normalized.endsWith('/' + current)) return null;
  return normalized.slice(0, -current.length) + id;
}

function routeRecordId(route: { params?: Record<string, unknown>; query?: Record<string, unknown> }): string | undefined {
  const raw = route.params?.recordId ?? route.params?.id ?? route.params?.Id ?? route.query?.recordId ?? route.query?.id ?? route.query?.Id;
  const s = String(raw ?? '').trim();
  if (!s || s === 'undefined' || s === 'null') return undefined;
  return s;
}

export function useFormViewSubmit<T extends BaseModel>(opts: UseFormViewSubmitOpts<T>) {
  const submitting = ref(false);

  async function validateForm() {
    if (opts.fieldErrors.value.size > 0) {
      if (opts.resolvedShowMessages.value) ChoyMessage.error(opts._t('Please fix the errors in the form first'));
      return false;
    }
    for (const validate of opts.fieldClientValidators.values()) {
      const message = await validate();
      if (message) {
        if (opts.resolvedShowMessages.value) ChoyMessage.error(opts._t('Please fix the errors in the form first'));
        return false;
      }
    }
    return true;
  }

  async function handleSubmit(): Promise<FormSubmitOutcome<T>> {
    const modeForEmit: FormSubmitMode = (opts.viewMode.value as any) === 'create' ? 'create' : 'edit';
    const currentFormData = () => (toRaw(opts.exposedFormData.value) as Partial<ClientModel<T>>) || null;

    if (opts.viewMode.value === 'display') {
      return {
        ok: false,
        mode: modeForEmit,
        handledByHandler: false,
        record: null,
        formData: currentFormData(),
        reason: 'not-editable',
      };
    }
    if (opts.loading.value || submitting.value) {
      return {
        ok: false,
        mode: modeForEmit,
        handledByHandler: false,
        record: null,
        formData: currentFormData(),
        reason: 'loading',
      };
    }
    submitting.value = true;
    try {
      opts.onchangeCtrl.pause();
      try {
        (document.activeElement as HTMLElement | null)?.blur?.();
      } catch {}
      await nextTick();
      const okFrm = await validateForm();
      if (!okFrm) {
        return {
          ok: false,
          mode: modeForEmit,
          handledByHandler: false,
          record: null,
          formData: currentFormData(),
          reason: 'validate-failed',
        };
      }
      const payloadForEmit = toRaw(opts.exposedFormData.value) as any;
      const ok = await opts.emitCancelable('before-submit', { mode: modeForEmit, data: payloadForEmit });
      if (!ok) {
        return {
          ok: false,
          mode: modeForEmit,
          handledByHandler: false,
          record: null,
          formData: currentFormData(),
          reason: 'before-submit-canceled',
        };
      }

      // Memoize so a handler that calls defaultSubmit then returns false cannot double-save.
      let defaultSubmitPromise: Promise<Partial<ClientModel<T>> | null> | null = null;
      const runDefaultSubmit = (): Promise<Partial<ClientModel<T>> | null> => {
        if (!defaultSubmitPromise) {
          defaultSubmitPromise = opts.controller
            .submit()
            .then(() => (opts.controller.vm.original as any) || null);
        }
        return defaultSubmitPromise;
      };

      let handledByHandler = false;
      let handlerRecord: Partial<ClientModel<T>> | null = null;
      let handlerSuccessMessage = '';
      let handlerSkipSuccessMessage = false;

      if (opts.submitHandler) {
        const handlerResult = await opts.submitHandler({
          mode: modeForEmit,
          data: payloadForEmit,
          formData: payloadForEmit,
          defaultSubmit: runDefaultSubmit,
        });

        if (typeof handlerResult === 'boolean') {
          handledByHandler = handlerResult;
        } else if (handlerResult && typeof handlerResult === 'object') {
          handledByHandler = Boolean((handlerResult as any).handled);
          handlerRecord = ((handlerResult as any).record as Partial<ClientModel<T>> | null | undefined) ?? null;
          handlerSuccessMessage = String((handlerResult as any).successMessage || '').trim();
          handlerSkipSuccessMessage = Boolean((handlerResult as any).skipSuccessMessage);
        }
      }

      if (!handledByHandler) {
        await runDefaultSubmit();
        if (opts.resolvedShowMessages.value)
          ChoyMessage.success(opts.viewMode.value === 'create' ? opts._t('Created successfully') : opts._t('Saved successfully'));
        if (modeForEmit === 'create') {
          let defaultPrevented = false;
          opts.emit('create-success', {
            record: opts.controller.vm.original as any,
            preventDefault: () => {
              defaultPrevented = true;
            },
          });
          if (!defaultPrevented) {
            const currentRoute = opts.router.currentRoute.value;
            const newId = String((opts.controller.vm.original as any)?.Id ?? '').trim();
            const newPath = formCreateSuccessPath(currentRoute.path, newId, routeRecordId(currentRoute));
            if (newPath && newPath !== currentRoute.path) {
              (opts.controller.vm as { mode: string }).mode = 'display';
              try {
                const breadcrumbStore = useBreadcrumbStore();
                const stack = breadcrumbStore.breadcrumbStack;
                if (stack.length > 0 && stack[stack.length - 1].path === currentRoute.path) {
                  stack[stack.length - 1].path = newPath;
                  Object.assign(stack[stack.length - 1], {
                    title: opts.detailsTitle.src,
                    titleText: { ...opts.detailsTitle },
                  });
                }
              } catch (e) {
                console.warn('Failed to update breadcrumb', e);
              }
              opts.router.replace(newPath);
            }
          }
        } else opts.emit('update-success', { record: opts.controller.vm.original as any });

        return {
          ok: true,
          mode: modeForEmit,
          handledByHandler: false,
          record: (opts.controller.vm.original as any) || null,
          formData: currentFormData(),
        };
      } else {
        const handledRecord = (handlerRecord as any) || (toRaw(opts.exposedFormData.value) as any) || null;
        const msg = handlerSuccessMessage || (modeForEmit === 'create' ? opts._t('Created successfully') : opts._t('Saved successfully'));
        if (opts.resolvedShowMessages.value && !handlerSkipSuccessMessage) ChoyMessage.success(msg);

        if (modeForEmit === 'create') {
          opts.emit('create-success', {
            record: handledRecord as any,
            preventDefault: () => {},
          });
        } else {
          opts.emit('update-success', { record: handledRecord as any });
        }

        return {
          ok: true,
          mode: modeForEmit,
          handledByHandler: true,
          record: handledRecord as any,
          formData: currentFormData(),
        };
      }
    } catch (e: any) {
      if (opts.resolvedShowMessages.value) ChoyMessage.error(e?.message || opts._t('Operation failed'));
      const err = e instanceof Error ? e : new Error(String(e));
      opts.emit('action-error', { action: opts.viewMode.value === 'create' ? 'create' : 'update', error: err });
      return {
        ok: false,
        mode: modeForEmit,
        handledByHandler: false,
        record: null,
        formData: currentFormData(),
        reason: 'error',
        error: err,
      };
    } finally {
      submitting.value = false;
      opts.onchangeCtrl.reset();
      opts.resetOnchangeAgg();
      opts.onchangeCtrl.resume();
    }
  }

  function onNativeSubmit() {
    void handleSubmit();
  }

  return {
    submitting,
    handleSubmit,
    onNativeSubmit,
    validateForm,
  };
}
