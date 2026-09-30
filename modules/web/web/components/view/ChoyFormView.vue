<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ViewContainer :showHeader="resolvedShowHeader">
    <template #header>
      <div class="form-view__action-bar flex items-center justify-between gap-3 border-b border-border pb-1 min-h-10 max-md:flex-col max-md:items-stretch">
        <div class="form-view__actions flex flex-1 items-center gap-4 max-md:flex-col max-md:items-stretch max-md:gap-2" v-if="resolvedShowActions">
          <div class="form-view__system-actions flex items-center gap-2">
            <slot name="system-actions">
              <template v-if="viewMode === 'display' && effectiveRecordId">
                <ChoyButton v-if="resolvedCreateAction && canCreate" size="sm" variant="default" @click="handleCreate">
                  <Plus class="size-4" />
                  {{ _t('New') }}
                </ChoyButton>
                <ChoyButton v-if="canEdit" size="sm" variant="outline" @click="handleEdit">
                  <Pencil class="size-4" />
                  {{ _t('Edit') }}
                </ChoyButton>
                <ChoyButton v-if="canRefresh" size="sm" variant="outline" @click="handleRefresh">
                  <RefreshCw class="size-4" />
                  {{ _t('Refresh') }}
                </ChoyButton>
                <ChoyButton v-if="canCopy" size="sm" variant="ghost" @click="handleCopy">
                  <Copy class="size-4" />
                  {{ _t('Copy') }}
                </ChoyButton>
                <ChoyButton
                  v-if="canDelete"
                  size="sm"
                  variant="outline"
                  class="border-danger/40 text-danger hover:bg-danger-subtle"
                  @click="handleDelete"
                >
                  <Trash2 class="size-4" />
                  {{ _t('Delete') }}
                </ChoyButton>
              </template>
              <template v-if="viewMode === 'edit'">
                <ChoyButton size="sm" variant="default" @click="handleSubmit" :disabled="loading">
                  <Check class="size-4" />
                  {{ saveLabel }}
                </ChoyButton>
                <ChoyButton size="sm" variant="outline" @click="handleCancel" :disabled="loading">
                  <X class="size-4" />
                  {{ _t('Cancel') }}
                </ChoyButton>
                <ChoyButton size="sm" variant="ghost" @click="handleReset" :disabled="loading">
                  <RotateCcw class="size-4" />
                  {{ _t('Reset') }}
                </ChoyButton>
              </template>
              <template v-if="viewMode === 'create'">
                <ChoyButton size="sm" variant="default" @click="handleSubmit" :disabled="loading">
                  <Check class="size-4" />
                  {{ saveLabel }}
                </ChoyButton>
                <ChoyButton size="sm" variant="outline" @click="handleCancel" :disabled="loading">
                  <X class="size-4" />
                  {{ _t('Cancel') }}
                </ChoyButton>
                <ChoyButton size="sm" variant="ghost" @click="handleReset" :disabled="loading">
                  <RotateCcw class="size-4" />
                  {{ _t('Reset') }}
                </ChoyButton>
              </template>
            </slot>
          </div>
          <div class="form-view__user-actions flex items-center gap-2 border-l border-border pl-4 max-md:border-l-0 max-md:border-t max-md:border-border max-md:pl-0 max-md:pt-2">
            <slot name="user-actions"> </slot>
          </div>
        </div>
        <div class="form-view__header-right flex items-center justify-end gap-3 max-md:justify-center">
          <slot name="statusbar" />
          <slot name="button-box" />
          <slot name="header-right"> </slot>
        </div>
      </div>
    </template>

    <!-- Always render the form; busy state is local (no Element Plus v-loading). -->
    <div
      class="form-view__content relative py-3"
      :class="{ 'form-view__content--busy pointer-events-none opacity-65': loading }"
      :aria-busy="loading || undefined"
    >      <form ref="formRef" @submit.prevent>
        <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
      </form>
    </div>
  </ViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
// =============================
// Section 1: Imports
// =============================
import { ref, computed, provide, inject, watch, toRaw, nextTick, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { FIELD_CLIENT_VALIDATORS_KEY } from '@/web/web/composables/fieldClientValidation';
import { confirmChoyAction, confirmChoyChoice } from '../../composables/confirmChoyAction';
import type { ClientModel, BaseModel, Updateable, Insertable } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';
import { deepClonePreserve } from '@/core/utils/clone';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { useResolvedCreateAction } from '@/web/web/composables/resolveCreateRoute';

import { provideOnchange } from '@/web/web/composables/useOnchange';
import type { ViewMode, ViewContainer as ViewContainerKind } from '@/web/web/components/view/ChoyViewScope.vue';
import { createFormController } from '@/web/web/controllers/formController';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import { useOnchangeAggregation } from '@/web/web/composables/useOnchangeAggregation';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import ViewContainer from '@/web/web/components/view/ViewContainer.vue';
import { nextLocalToken } from '@/web/web/components/view/localToken';
import { createTranslate } from '@/web/web/i18n';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { Plus, Pencil, RefreshCw, Trash2, Check, X, Copy, RotateCcw } from 'lucide-vue-next';
import type {
  FormSubmitMode,
  FormSubmitHandlerContext,
  FormSubmitHandlerResult,
  FormSubmitHandler,
  FormSubmitFailureReason,
  FormSubmitOutcome,
  FormChildSubmitApi,
  FormChildSubmitApiRegistration,
  FormChildSubmitApiRegister
} from '@/web/web/components/view/formViewTypes';

defineOptions({ name: 'ChoyFormView' });

/**
 * Store-only form engine. Requires :store or a page-provided store
 * (resolvePageStore); there is no chrome-only dual-mode path.
 */
const { _t, _lt } = createTranslate('web', { scope: 'web/components/view/FormView' });
const detailsTitle = _lt('Details');

const FORM_CHILD_SUBMIT_API_REGISTER_KEY = 'form-child-submit-api-register';
const FORM_EMBEDDED_CONTEXT_KEY = 'form-embedded-context';

// =============================
// Section 2: Router & basic utilities
// =============================
const router = useRouter();
const route = useRoute();

// =============================
// Section 3: Props & defaults
// =============================
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    recordId?: string;
    initialValues?: Partial<T>;
    viewMode?: ViewMode;
    embedded?: boolean;
    showHeader?: boolean;
    showActions?: boolean;
    createAction?: string | RouteLocationRaw;
    actionIds?: ActionIdMap;
    hasAction?: (actionId: string | undefined) => boolean;
    showMessages?: boolean;
    onchangeSessionId?: string;
    onchangeDebounceMs?: number;
    onchangeImmediateFirst?: boolean;
    submitHandler?: FormSubmitHandler<T>;
    resolveRecordIdFromRoute?: boolean;
  }>(),
  {
    createAction: undefined,
    onchangeSessionId: undefined,
    onchangeDebounceMs: 150,
    onchangeImmediateFirst: false,
    submitHandler: undefined,
  }
);

const store = resolvePageStore(props.store, 'ChoyFormView');

const rawPropKeys = new Set(Object.keys(((getCurrentInstance()?.vnode.props as Record<string, unknown> | null) || {}) as Record<string, unknown>));
const hasExplicitProp = (...keys: string[]) => keys.some(key => rawPropKeys.has(key));
const hasEmbeddedProp = hasExplicitProp('embedded');
const hasShowHeaderProp = hasExplicitProp('showHeader', 'show-header');
const hasShowActionsProp = hasExplicitProp('showActions', 'show-actions');
const hasShowMessagesProp = hasExplicitProp('showMessages', 'show-messages');
const hasResolveRecordIdFromRouteProp = hasExplicitProp('resolveRecordIdFromRoute', 'resolve-record-id-from-route');

// =============================
// Section 4: View layout notes (no auto-height here)
// =============================
// Adaptive height is intentionally disabled for this view.

// =============================
// Section 5: Slots declaration
// =============================
defineSlots<{
  breadcrumb(): any;
  'system-actions'(): any;
  'user-actions'(): any;
  statusbar(): any;
  'button-box'(): any;
  'header-right'(): any;
  default(props: { formData: Partial<ClientModel<T>>; viewMode: ViewMode; loading: boolean }): any;
}>();

// =============================
// Section 6: Emits definition
// =============================
const emit = defineEmits<{
  (e: 'before-load', payload: { confirm: () => void; cancel: () => void }): void;
  (e: 'before-submit', payload: { mode: 'create' | 'edit'; data: Insertable<T> | Updateable<T>; confirm: () => void; cancel: () => void }): void;
  (e: 'before-delete', payload: { id: string; confirm: () => void; cancel: () => void }): void;
  (e: 'before-refresh', payload: { confirm: () => void; cancel: () => void }): void;
  (e: 'load-success', payload: { record: ClientModel<T> | null }): void;
  (e: 'create-success', payload: { record: ClientModel<T>; preventDefault: () => void }): void;
  (e: 'update-success', payload: { record: ClientModel<T> }): void;
  (e: 'delete-success', payload: { id: string; preventDefault: () => void }): void;
  (e: 'refresh-success', payload: { record: ClientModel<T> | null }): void;
  (e: 'mode-change', payload: { mode: ViewMode }): void;
  (e: 'change', payload: { formData: Partial<ClientModel<T>> }): void;
  (e: 'reset', payload: { formData: Partial<ClientModel<T>> }): void;
  (e: 'copy', payload: { formData: Partial<ClientModel<T>> }): void;
  (e: 'action-error', payload: { action: 'load' | 'create' | 'update' | 'delete' | 'refresh'; error: Error }): void;
}>();

// =============================
// Section 7: Cancelable emit helper
// =============================
const { emitCancelable } = useCancelableEmit(emit as any);

// =============================
// Section 8: Controller & view context provisioning
// =============================
const formRef = ref<HTMLFormElement | null>(null);
const viewContainer = ref<ViewContainerKind>('Form');
const canCreate = computed(() => canShowAction(props.actionIds?.create, props.hasAction));
const canEdit = computed(() => canShowAction(props.actionIds?.edit, props.hasAction));
const canDelete = computed(() => canShowAction(props.actionIds?.delete, props.hasAction));
const canCopy = computed(() => canShowAction(props.actionIds?.copy, props.hasAction));
const canRefresh = computed(() => canShowAction(props.actionIds?.refresh, props.hasAction));
// Form controller.
const controller = createFormController(store as any);
controller.provideToChildren();
const viewMode = computed<ViewMode>(() => controller.vm.mode as ViewMode);
const loading = computed<boolean>(() => !!controller.vm.loading);
const saveLabel = computed(() => (loading.value ? _t('Saving...') : _t('Save')));
const registerChildSubmitApi = inject<FormChildSubmitApiRegister | null>(FORM_CHILD_SUBMIT_API_REGISTER_KEY, null);
const embeddedFromHost = inject<boolean | null>(FORM_EMBEDDED_CONTEXT_KEY, null);
const childSubmitRegistrationToken = nextLocalToken('form-view');

const isEmbedded = computed<boolean>(() => {
  if (hasEmbeddedProp) return props.embedded === true;
  if (typeof embeddedFromHost === 'boolean') return embeddedFromHost;
  return false;
});
const resolvedShowHeader = computed<boolean>(() => (hasShowHeaderProp ? props.showHeader === true : !isEmbedded.value));
const resolvedShowActions = computed<boolean>(() => (hasShowActionsProp ? props.showActions === true : !isEmbedded.value));
const resolvedShowMessages = computed<boolean>(() => (hasShowMessagesProp ? props.showMessages === true : !isEmbedded.value));
const resolvedResolveRecordIdFromRoute = computed<boolean>(() =>
  hasResolveRecordIdFromRouteProp ? props.resolveRecordIdFromRoute === true : !isEmbedded.value
);
const resolvedCreateAction = useResolvedCreateAction(() => props.createAction, {
  enabled: () => !isEmbedded.value,
});

// Guard the current submit channel from being captured by deeper nested FormView instances.
// Deeper nesting should re-provide its own registration entry from the nearest container.
provide<FormChildSubmitApiRegister>(FORM_CHILD_SUBMIT_API_REGISTER_KEY, (_registration: FormChildSubmitApiRegistration) => {});

// =============================
// Section 9: Onchange aggregation (field errors + messages)
// =============================
// Aggregate onchange field errors, candidate updates, and global messages.
const { lastOnchangeResult, fieldErrors, afterFlushHandler, reset: resetOnchangeAgg } = useOnchangeAggregation({ showMessages: resolvedShowMessages.value });

// =============================
// Section 10: Provide view environment (mode/container/errors)
// =============================
// Provide the view environment.
provide('view-mode', viewMode);
provide('view-container', viewContainer);
provide('field-errors', fieldErrors);

// Client-side RuleItem validators registered by FieldBase (replaces el-form-item).
const fieldClientValidators = new Map<string, () => Promise<string>>();
provide(FIELD_CLIENT_VALIDATORS_KEY, fieldClientValidators);

// =============================
// Section 11: Onchange controller (session scoped)
// =============================
// Provide the session-scoped onchange controller.
const localSessionId = props.onchangeSessionId || nextLocalToken(`FormView:${props.recordId ?? 'new'}`);
const onchangeCtrl = provideOnchange(store, localSessionId, {
  debounceMs: props.onchangeDebounceMs,
  immediateFirst: props.onchangeImmediateFirst,
  // Inject root record access so the composable stays decoupled from store internals.
  getRoot: () => controller.vm.draft,
  onPatch: (value: any) => {
    const r = controller.vm.draft as any;
    if (r && typeof r === 'object' && value && typeof value === 'object') {
      Object.assign(r, value);
    }
  },
});

// =============================
// Section 12: Provide reactive data for child fields
// =============================
// Provide readonly onchange results to child field components.
provide('lastOnchangeResult', lastOnchangeResult);

// =============================
// Section 13: Lifecycle hooks
// =============================
onMounted(() => {
  onchangeCtrl.registerAfterFlush(afterFlushHandler);
  registerChildSubmitApi?.({
    token: childSubmitRegistrationToken,
    api: {
      submit: handleSubmit as () => Promise<unknown>,
      getFormData: () => toRaw(exposedFormData.value) as any,
    },
  });
});
onBeforeUnmount(() => {
  onchangeCtrl.unregisterAfterFlush(afterFlushHandler);
  registerChildSubmitApi?.({
    token: childSubmitRegistrationToken,
    api: null,
  });
});

// =============================
// Section 14: Computed exposed form data
// =============================
const exposedFormData = computed<Partial<ClientModel<T>>>(() => (controller.vm.draft as any) || {});

// Reserved extension point for future form-layout composables.

// =============================
// Section 15: Initialization & mode switching
// =============================
function normalizeRecordId(raw: unknown): string | undefined {
  const s = String(raw ?? '').trim();
  if (!s || s === 'undefined' || s === 'null') return undefined;
  return s;
}

/** Route param/query id used when resolveRecordIdFromRoute is enabled. */
function peekRouteRecordId(): string | undefined {
  if (!resolvedResolveRecordIdFromRoute.value) {
    return undefined;
  }
  return normalizeRecordId(
    route.params.recordId ?? route.params.id ?? route.params.Id ?? route.query.recordId ?? route.query.id ?? route.query.Id
  );
}

/** Prop record id when set; otherwise the route-resolved detail id. */
const effectiveRecordId = computed(() => normalizeRecordId(props.recordId) ?? peekRouteRecordId());

/** Monotonic token so overlapping initializeForm runs discard stale completions. */
let initializeSeq = 0;

async function initializeForm() {
  const seq = ++initializeSeq;
  const isStale = () => seq !== initializeSeq;
  try {
    const ok = await emitCancelable('before-load');
    if (!ok || isStale()) return;
    onchangeCtrl.reset();
    resetOnchangeAgg();
    // Resolve the effective record id from props first, then from common route keys.
    const normId = effectiveRecordId.value;
    if (normId) {
      await controller.beginDisplay(normId);
      if (isStale()) return;
      emit('mode-change', { mode: controller.vm.mode as ViewMode });
      emit('load-success', { record: (controller.vm.original as any) || null });
    } else {
      await controller.beginCreate(deepClonePreserve((props.initialValues || {}) as any));
      if (isStale()) return;
      if ((props.viewMode as any) === 'display') {
        // Support read-only preview flows driven only by initialValues in nested forms.
        (controller.vm as any).mode = 'display';
      }
      emit('mode-change', { mode: controller.vm.mode as ViewMode });
      emit('load-success', { record: null as any });
    }
    if (isStale()) return;
    emit('change', { formData: toRaw(exposedFormData.value) as any });
  } catch (e: any) {
    if (isStale()) return;
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'load', error: err });
  }
}

// =============================
// Section 16: Validation pipeline
// =============================
async function validateForm() {
  // Block submission when server-side field errors are still present.
  if (fieldErrors.value.size > 0) {
    if (resolvedShowMessages.value) ChoyMessage.error(_t('Please fix the errors in the form first'));
    return false;
  }
  // Run FieldBase-registered client RuleItem validators (required/format/range).
  for (const validate of fieldClientValidators.values()) {
    const message = await validate();
    if (message) {
      if (resolvedShowMessages.value) ChoyMessage.error(_t('Please fix the errors in the form first'));
      return false;
    }
  }
  return true;
}

// =============================
// Section 17: Mode & draft management handlers
// =============================
function handleEdit() {
  controller.beginEdit();
  onchangeCtrl.reset();
  resetOnchangeAgg();
  emit('mode-change', { mode: 'edit' });
}

function handleCancel() {
  onchangeCtrl.reset();
  resetOnchangeAgg();
  if (viewMode.value === 'create') {
    router.back();
  } else {
    const id = effectiveRecordId.value;
    if (id) controller.beginDisplay(id);
    emit('mode-change', { mode: 'display' });
  }
}

function handleReset() {
  resetOnchangeAgg();
  controller.reset();
  onchangeCtrl.reset();
  emit('reset', { formData: toRaw(exposedFormData.value) as any });
}

// =============================
// Section 18: Submit handler
// =============================
async function handleSubmit(): Promise<FormSubmitOutcome<T>> {
  const modeForEmit: FormSubmitMode = (viewMode.value as any) === 'create' ? 'create' : 'edit';
  const currentFormData = () => (toRaw(exposedFormData.value) as Partial<ClientModel<T>>) || null;

  if (loading.value) {
    return {
      ok: false,
      mode: modeForEmit,
      handledByHandler: false,
      record: null,
      formData: currentFormData(),
      reason: 'loading',
    };
  }
  // Pause automatic onchange flushing during submit.
  onchangeCtrl.pause();
  try {
    try {
      (document.activeElement as HTMLElement | null)?.blur?.();
    } catch {}
    await nextTick();
    // Run only client-side validation and keep the existing validation flow.
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
    // Fire before-submit.
    const payloadForEmit = toRaw(exposedFormData.value) as any;
    const ok = await emitCancelable('before-submit', { mode: modeForEmit, data: payloadForEmit });
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

    const runDefaultSubmit = async (): Promise<Partial<ClientModel<T>> | null> => {
      await controller.submit();
      return (controller.vm.original as any) || null;
    };

    let handledByHandler = false;
    let handlerRecord: Partial<ClientModel<T>> | null = null;
    let handlerSuccessMessage = '';
    let handlerSkipSuccessMessage = false;

    if (props.submitHandler) {
      const handlerResult = await props.submitHandler({
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
      if (resolvedShowMessages.value) ChoyMessage.success(viewMode.value === 'create' ? _t('Created successfully') : _t('Saved successfully'));
      // Emit external success events.
      if (modeForEmit === 'create') {
        let defaultPrevented = false;
        emit('create-success', {
          record: controller.vm.original as any,
          preventDefault: () => {
            defaultPrevented = true;
          },
        });
        if (!defaultPrevented) {
          const currentRoute = router.currentRoute.value;
          if (currentRoute.path.endsWith('/new')) {
            const newId = (controller.vm.original as any).Id;
            if (newId) {
              const newPath = currentRoute.path.replace(/\/new$/, `/${newId}`);

              // Replace the current "new" breadcrumb with the detail path before navigation.
              // This keeps the guard on the same breadcrumb node so it updates instead of appending.
              try {
                const breadcrumbStore = useBreadcrumbStore();
                const stack = breadcrumbStore.breadcrumbStack;
                if (stack.length > 0 && stack[stack.length - 1].path === currentRoute.path) {
                  stack[stack.length - 1].path = newPath;
                  // Seed a temporary title until the route guard applies the final one.
                  Object.assign(stack[stack.length - 1], {
                    title: detailsTitle.src,
                    titleText: { ...detailsTitle },
                  });
                }
              } catch (e) {
                console.warn('Failed to update breadcrumb', e);
              }

              router.replace(newPath);
            }
          }
        }
      } else emit('update-success', { record: controller.vm.original as any });

      return {
        ok: true,
        mode: modeForEmit,
        handledByHandler: false,
        record: (controller.vm.original as any) || null,
        formData: currentFormData(),
      };
    } else {
      const handledRecord = (handlerRecord as any) || (toRaw(exposedFormData.value) as any) || null;
      const msg = handlerSuccessMessage || (modeForEmit === 'create' ? _t('Created successfully') : _t('Saved successfully'));
      if (resolvedShowMessages.value && !handlerSkipSuccessMessage) ChoyMessage.success(msg);

      if (modeForEmit === 'create') {
        emit('create-success', {
          record: handledRecord as any,
          preventDefault: () => {},
        });
      } else {
        emit('update-success', { record: handledRecord as any });
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
    if (resolvedShowMessages.value) ChoyMessage.error(e?.message || _t('Operation failed'));
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: viewMode.value === 'create' ? 'create' : 'update', error: err });
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
    onchangeCtrl.reset();
    resetOnchangeAgg();
    onchangeCtrl.resume();
  }
}

// =============================
// Section 19: Ancillary operations (create/refresh/copy/delete)
// =============================
async function handleCreate() {
  if (!resolvedCreateAction.value) return;
  try {
    await router.push(resolvedCreateAction.value);
  } catch {}
}

async function handleRefresh() {
  const ok = await emitCancelable('before-refresh');
  if (!ok) return;

  const id = effectiveRecordId.value;
  if (!id) {
    emit('refresh-success', { record: null });
    return;
  }
  try {
    await controller.beginDisplay(id);
    if (resolvedShowMessages.value) ChoyMessage.success(_t('Refreshed'));
    emit('refresh-success', { record: (controller.vm.original || null) as any });
  } catch (e: any) {
    if (resolvedShowMessages.value) ChoyMessage.error(_t('Refresh failed'));
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'refresh', error: err });
  }
}

async function handleCopy() {
  if (!(controller.vm.original as any)) return;
  const cp = deepClonePreserve(controller.vm.original as any);
  delete (cp as any).Id;
  await controller.beginCreate(cp);
  resetOnchangeAgg();
  emit('mode-change', { mode: 'create' });
  onchangeCtrl.reset();
  emit('copy', { formData: toRaw(exposedFormData.value) as any });
}

function handleDelete() {
  const currId = (controller.vm.original as any)?.Id;
  if (!currId) return;
  confirmChoyAction(_t('Are you sure you want to delete the current record? This action cannot be undone.'), _t('Confirm delete'), {
    confirmText: _t('Delete'),
    cancelText: _t('Cancel'),
    destructive: true,
  })
    .then(async () => {
      const id = String(currId);
      const ok = await emitCancelable('before-delete', { id });
      if (!ok) return;
      await controller.delete();
      resetOnchangeAgg();
      emit('mode-change', { mode: 'display' });
      ChoyMessage.success(_t('Record deleted'));
      let defaultPrevented = false;
      emit('delete-success', {
        id,
        preventDefault: () => {
          defaultPrevented = true;
        },
      });

      if (!defaultPrevented) {
        const currentRoute = router.currentRoute.value;
        // Infer the list route by removing the last path segment, which is assumed to be the record id.
        const pathSegments = currentRoute.path.split('/').filter(Boolean);
        if (pathSegments.length > 0) {
          const listPath = '/' + pathSegments.slice(0, -1).join('/');

          // Remove the current breadcrumb when this page is the stack top.
          try {
            const breadcrumbStore = useBreadcrumbStore();
            const stack = breadcrumbStore.breadcrumbStack;
            if (stack.length > 0 && stack[stack.length - 1].path === currentRoute.path) {
              stack.pop();
            }
          } catch (e) {
            console.warn('Failed to update breadcrumb', e);
          }

          router.replace(listPath);
        }
      }
    })
    .catch((e: unknown) => {
      if (e === 'cancel' || e === 'dismiss') return;
      ChoyMessage.error(_t('Delete failed'));
      const err = e instanceof Error ? e : new Error(String(e));
      emit('action-error', { action: 'delete', error: err });
    });
}

// =============================
// Section 20: Data change broadcasting (external consumers)
// =============================
watch(exposedFormData, v => emit('change', { formData: toRaw(v) as any }), { deep: true, flush: 'post' });

// =============================
// Section 21: Watchers (recordId/viewMode/session/route identity)
// =============================
/* Re-init when prop identity or (non-embedded) route name/id changes. */
watch(
  () =>
    [
      props.recordId,
      props.viewMode,
      props.onchangeSessionId,
      resolvedResolveRecordIdFromRoute.value ? String(route.name ?? '') : '',
      resolvedResolveRecordIdFromRoute.value ? peekRouteRecordId() ?? '' : '',
    ] as const,
  async () => {
    // Invalidate in-flight initializeForm before awaiting nextTick so a prior
    // beginDisplay cannot emit load-success for the superseded identity.
    initializeSeq += 1;
    await nextTick();
    await initializeForm();
    // Enter edit mode when requested by the external viewMode prop.
    if ((props.viewMode as any) === 'edit') controller.beginEdit();
  },
  { immediate: true, flush: 'sync' }
);

defineExpose({
  submit: handleSubmit,
  refresh: handleRefresh,
  reset: handleReset,
  copy: handleCopy,
  getFormData: () => toRaw(exposedFormData.value) as any,
  getViewMode: () => viewMode.value,
  isLoading: () => loading.value,
});
</script>

