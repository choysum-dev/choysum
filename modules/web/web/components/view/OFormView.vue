<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OViewContainer :showHeader="resolvedShowHeader">
    <template #header>
      <div class="form-view__action-bar">
        <div class="form-view__actions" v-if="resolvedShowActions">
          <div class="form-view__system-actions">
            <slot name="system-actions">
              <template v-if="viewMode === 'display' && recordId">
                <el-button v-if="createAction && canCreate" size="small" plain type="primary" @click="handleCreate">
                  <el-icon><Plus /></el-icon>
                  {{ _t('New') }}
                </el-button>
                <el-button v-if="canEdit" size="small" plain type="primary" @click="handleEdit">
                  <el-icon><Edit /></el-icon>
                  {{ _t('Edit') }}
                </el-button>
                <el-button v-if="canRefresh" size="small" plain @click="handleRefresh">
                  <el-icon><Refresh /></el-icon>
                  {{ _t('Refresh') }}
                </el-button>
                <el-button v-if="canCopy" size="small" plain @click="handleCopy">
                  <el-icon><ContentCopyFilled /></el-icon>
                  {{ _t('Copy') }}
                </el-button>
                <el-button v-if="canDelete" size="small" plain type="danger" @click="handleDelete">
                  <el-icon><Delete /></el-icon>
                  {{ _t('Delete') }}
                </el-button>
              </template>
              <template v-if="viewMode === 'edit'">
                <el-button size="small" plain type="success" @click="handleSubmit" :loading="loading">
                  <el-icon><Check /></el-icon>
                  {{ saveLabel }}
                </el-button>
                <el-button size="small" plain @click="handleCancel" :disabled="loading">
                  <el-icon><Close /></el-icon>
                  {{ _t('Cancel') }}
                </el-button>
                <el-button size="small" plain @click="handleReset" :disabled="loading">
                  <el-icon><RestoreOutlined /></el-icon>
                  {{ _t('Reset') }}
                </el-button>
              </template>
              <template v-if="viewMode === 'create'">
                <el-button size="small" plain type="success" @click="handleSubmit" :loading="loading">
                  <el-icon><Check /></el-icon>
                  {{ saveLabel }}
                </el-button>
                <el-button size="small" plain @click="handleCancel" :disabled="loading">
                  <el-icon><Close /></el-icon>
                  {{ _t('Cancel') }}
                </el-button>
                <el-button size="small" plain @click="handleReset" :disabled="loading">
                  <el-icon><RestoreOutlined /></el-icon>
                  {{ _t('Reset') }}
                </el-button>
              </template>
            </slot>
          </div>
          <div class="form-view__user-actions">
            <slot name="user-actions"> </slot>
          </div>
        </div>
        <div class="form-view__header-right">
          <slot name="statusbar" />
          <slot name="button-box" />
          <slot name="header-right"> </slot>
        </div>
      </div>
    </template>

    <!-- Always render the form and rely on v-loading for the overlay. -->
    <div class="form-view__content" v-loading="loading">
      <el-form ref="formRef" :model="exposedFormData as Record<string, any>" :hide-required-asterisk="viewMode == 'display' ? true : false">
        <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
      </el-form>
    </div>
  </OViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
// =============================
// Section 1: Imports
// =============================
import { ref, computed, provide, inject, watch, toRaw, nextTick, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue';
import { ElForm, ElMessageBox, ElMessage } from 'element-plus';
import type { ClientModel, BaseModel, Updateable, Insertable } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { Edit, Check, Close, Plus, Refresh, Delete } from '@element-plus/icons-vue';
import { ContentCopyFilled, RestoreOutlined } from '@vicons/material';
import type { RouteLocationRaw } from 'vue-router';
import { useRouter } from 'vue-router';
import { deepClonePreserve } from '@/core/utils/clone';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';

import { provideOnchange } from '@/web/web/composables/useOnchange';
import type { ViewMode, ViewContainer } from '@/web/web/components/view/OViewScope.vue';
import { createFormController } from '@/web/web/controllers/formController';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import { useOnchangeAggregation } from '@/web/web/composables/useOnchangeAggregation';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
import { nextLocalToken } from '@/web/web/components/view/localToken';
import { createTranslate } from '@/web/web/i18n';
import type {
  OFormSubmitMode,
  OFormSubmitHandlerContext,
  OFormSubmitHandlerResult,
  OFormSubmitHandler,
  OFormSubmitFailureReason,
  OFormSubmitOutcome,
  OFormChildSubmitApi,
  OFormChildSubmitApiRegistration,
  OFormChildSubmitApiRegister,
} from '@/web/web/components/view/formViewTypes';

const { _t, _lt } = createTranslate('web', { scope: 'web/components/view/OFormView' });
const detailsTitle = _lt('Details');

const O_FORM_CHILD_SUBMIT_API_REGISTER_KEY = 'o-form-child-submit-api-register';
const O_FORM_EMBEDDED_CONTEXT_KEY = 'o-form-embedded-context';

// =============================
// Section 2: Router & basic utilities
// =============================
const router = useRouter();

// =============================
// Section 3: Props & defaults
// =============================
const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
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
    submitHandler?: OFormSubmitHandler<T>;
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
const store = props.store;
const formRef = ref<InstanceType<typeof ElForm>>();
const viewContainer = ref<ViewContainer>('Form');
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
const registerChildSubmitApi = inject<OFormChildSubmitApiRegister | null>(O_FORM_CHILD_SUBMIT_API_REGISTER_KEY, null);
const embeddedFromHost = inject<boolean | null>(O_FORM_EMBEDDED_CONTEXT_KEY, null);
const childSubmitRegistrationToken = nextLocalToken('o-form-view');

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

// Guard the current submit channel from being captured by deeper nested OFormView instances.
// Deeper nesting should re-provide its own registration entry from the nearest container.
provide<OFormChildSubmitApiRegister>(O_FORM_CHILD_SUBMIT_API_REGISTER_KEY, (_registration: OFormChildSubmitApiRegistration) => {});

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
async function initializeForm() {
  try {
    const ok = await emitCancelable('before-load');
    if (!ok) return;
    onchangeCtrl.reset();
    resetOnchangeAgg();
    // Resolve the effective record id from props first, then from common route keys.
    const route = router.currentRoute.value;
    const routeRecordId = resolvedResolveRecordIdFromRoute.value
      ? (route.params.recordId ?? route.params.id ?? route.params.Id ?? route.query.recordId ?? route.query.id ?? route.query.Id)
      : undefined;
    const rawId: any = props.recordId ?? routeRecordId;
    const normId = (() => {
      const s = String(rawId ?? '').trim();
      if (!s || s === 'undefined' || s === 'null') return undefined;
      return s;
    })();
    if (normId) {
      await controller.beginDisplay(normId);
      emit('mode-change', { mode: controller.vm.mode as ViewMode });
      emit('load-success', { record: (controller.vm.original as any) || null });
    } else {
      await controller.beginCreate(deepClonePreserve((props.initialValues || {}) as any));
      if ((props.viewMode as any) === 'display') {
        // Support read-only preview flows driven only by initialValues in nested forms.
        (controller.vm as any).mode = 'display';
      }
      emit('mode-change', { mode: controller.vm.mode as ViewMode });
      emit('load-success', { record: null as any });
    }
    emit('change', { formData: toRaw(exposedFormData.value) as any });
  } catch (e: any) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'load', error: err });
  }
}

// =============================
// Section 16: Validation pipeline
// =============================
async function validateForm() {
  if (!formRef.value) return true;

  // Block submission when server-side field errors are still present.
  if (fieldErrors.value.size > 0) {
    if (resolvedShowMessages.value) ElMessage.error(_t('Please fix the errors in the form first'));
    return false;
  }

  try {
    await formRef.value.validate();

    // Recheck field errors in case validation triggered new server-side issues.
    if (fieldErrors.value.size > 0) {
      if (resolvedShowMessages.value) ElMessage.error(_t('Please fix the errors in the form first'));
      return false;
    }
    return true;
  } catch {
    return false;
  }
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
  formRef.value?.clearValidate();
  onchangeCtrl.reset();
  resetOnchangeAgg();
  if (viewMode.value === 'create') {
    router.back();
  } else {
    if (props.recordId) controller.beginDisplay(props.recordId);
    emit('mode-change', { mode: 'display' });
  }
}

function handleReset() {
  formRef.value?.clearValidate();
  resetOnchangeAgg();
  controller.reset();
  onchangeCtrl.reset();
  emit('reset', { formData: toRaw(exposedFormData.value) as any });
}

// =============================
// Section 18: Submit handler
// =============================
async function handleSubmit(): Promise<OFormSubmitOutcome<T>> {
  const modeForEmit: OFormSubmitMode = (viewMode.value as any) === 'create' ? 'create' : 'edit';
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
      if (resolvedShowMessages.value) ElMessage.success(viewMode.value === 'create' ? _t('Created successfully') : _t('Saved successfully'));
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
      if (resolvedShowMessages.value && !handlerSkipSuccessMessage) ElMessage.success(msg);

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
    if (resolvedShowMessages.value) ElMessage.error(e?.message || _t('Operation failed'));
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
  if (!props.createAction) return;
  try {
    await router.push(props.createAction);
  } catch {}
}

async function handleRefresh() {
  const ok = await emitCancelable('before-refresh');
  if (!ok) return;

  if (!props.recordId) {
    emit('refresh-success', { record: null });
    return;
  }
  try {
    await controller.beginDisplay(props.recordId);
    if (resolvedShowMessages.value) ElMessage.success(_t('Refreshed'));
    emit('refresh-success', { record: (controller.vm.original || null) as any });
  } catch (e: any) {
    if (resolvedShowMessages.value) ElMessage.error(_t('Refresh failed'));
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
  const currId = (controller.vm.original as any)?.Id ?? (controller.vm.original as any)?.id;
  if (!currId) return;
  ElMessageBox.confirm(_t('Are you sure you want to delete the current record? This action cannot be undone.'), _t('Confirm delete'), {
    confirmButtonText: _t('Delete'),
    cancelButtonText: _t('Cancel'),
    type: 'warning',
    confirmButtonClass: 'el-button--danger',
  })
    .then(async () => {
      const id = String(currId);
      const ok = await emitCancelable('before-delete', { id });
      if (!ok) return;
      await controller.delete();
      resetOnchangeAgg();
      emit('mode-change', { mode: 'display' });
      ElMessage.success(_t('Record deleted'));
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
      if (e !== 'cancel') {
        ElMessage.error(_t('Delete failed'));
        const err = e instanceof Error ? e : new Error(String(e));
        emit('action-error', { action: 'delete', error: err });
      }
    });
}

// =============================
// Section 20: Data change broadcasting (external consumers)
// =============================
watch(exposedFormData, v => emit('change', { formData: toRaw(v) as any }), { deep: true, flush: 'post' });

// =============================
// Section 21: Watchers (recordId/viewMode/session)
// =============================
/* Watch props.recordId, viewMode, and sessionId changes. */
watch(
  () => [props.recordId, props.viewMode, props.onchangeSessionId] as const,
  async () => {
    await nextTick();
    await initializeForm();
    // Enter edit mode when requested by the external viewMode prop.
    if ((props.viewMode as any) === 'edit') controller.beginEdit();
  },
  { immediate: true }
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

<style lang="scss" scoped>
.form-view__action-bar {
  display: flex;
  justify-content: space-between;
  padding-bottom: 4px;
  align-items: center;
  border-bottom: 1px solid var(--el-border-color-light);
  min-height: 40px;
}
.form-view__actions {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
}
.form-view__system-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.form-view__user-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 16px;
  border-left: 1px solid var(--el-border-color-light);
}
.form-view__header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.form-view__content {
  padding: 16px 0;
  :deep(.el-form-item__label) {
    width: 120px;
  }
}

@media (max-width: 768px) {
  .form-view__action-bar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  .form-view__header-right {
    justify-content: center;
  }
  .form-view__actions {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .form-view__user-actions {
    padding-left: 0;
    border-left: none;
    border-top: 1px solid var(--el-border-color-light);
    padding-top: 8px;
  }
}
</style>
