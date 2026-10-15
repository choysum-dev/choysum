<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer>
    <template v-if="resolvedShowHeader" #header>
      <div data-anchor="choy.form.view-chrome">
        <ChoyViewActionBar
          layout="split"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="resolvedShowActions" #system-actions>
            <slot name="system-actions">
              <template v-if="viewMode === 'display' && effectiveRecordId">
                <ChoyButton
                  v-if="resolvedCreateAction && canCreate"
                  type="button"
                  size="default"
                  variant="default"
                  @click="handleCreate"
                >
                  <Plus class="size-4" />
                  {{ _t('New') }}
                </ChoyButton>
                <ChoyButton v-if="canEdit" type="button" size="default" variant="outline" @click="handleEdit">
                  <Pencil class="size-4" />
                  {{ _t('Edit') }}
                </ChoyButton>
                <ChoyButton v-if="canRefresh" type="button" size="default" variant="outline" @click="handleRefresh">
                  <RefreshCw class="size-4" />
                  {{ _t('Refresh') }}
                </ChoyButton>
                <ChoyButton v-if="canCopy" type="button" size="default" variant="ghost" @click="handleCopy">
                  <Copy class="size-4" />
                  {{ _t('Copy') }}
                </ChoyButton>
                <ChoyButton
                  v-if="canDelete"
                  type="button"
                  size="default"
                  variant="outline"
                  class="border-danger/40 text-danger hover:bg-danger-subtle"
                  @click="handleDelete"
                >
                  <Trash2 class="size-4" />
                  {{ _t('Delete') }}
                </ChoyButton>
              </template>
              <template v-if="viewMode === 'edit'">
                <ChoyButton type="button" size="default" variant="default" @click="handleSubmit" :disabled="formBusy">
                  <Check class="size-4" />
                  {{ saveLabel }}
                </ChoyButton>
                <ChoyButton type="button" size="default" variant="outline" @click="handleCancel" :disabled="formBusy">
                  <X class="size-4" />
                  {{ _t('Cancel') }}
                </ChoyButton>
                <ChoyButton type="button" size="default" variant="ghost" @click="handleReset" :disabled="formBusy">
                  <RotateCcw class="size-4" />
                  {{ _t('Reset') }}
                </ChoyButton>
              </template>
              <template v-if="viewMode === 'create'">
                <ChoyButton type="button" size="default" variant="default" @click="handleSubmit" :disabled="formBusy">
                  <Check class="size-4" />
                  {{ saveLabel }}
                </ChoyButton>
                <ChoyButton type="button" size="default" variant="outline" @click="handleCancel" :disabled="formBusy">
                  <X class="size-4" />
                  {{ _t('Cancel') }}
                </ChoyButton>
                <ChoyButton type="button" size="default" variant="ghost" @click="handleReset" :disabled="formBusy">
                  <RotateCcw class="size-4" />
                  {{ _t('Reset') }}
                </ChoyButton>
              </template>
            </slot>
          </template>
          <template v-if="resolvedShowActions" #user-actions>
            <slot name="user-actions" />
          </template>
          <template #header-right>
            <slot name="statusbar" />
            <slot name="button-box" />
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <!-- pt tighter than pb: ViewContainer gap already separates ActionBar hairline. -->
    <ChoyViewBusyHost :busy="formBusy" class="pt-2 pb-3" data-anchor="choy.form-view">
      <form ref="formRef" @submit.self.prevent="onNativeSubmit">
        <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
      </form>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
// =============================
// Section 1: Imports
// =============================
import { ref, computed, provide, inject, watch, toRaw, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue';
import { FIELD_CLIENT_VALIDATORS_KEY } from '@/web/web/composables/fieldClientValidation';
import type { ClientModel, BaseModel, Updateable, Insertable } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { useResolvedCreateAction } from '@/web/web/composables/resolveCreateRoute';

import { provideOnchange } from '@/web/web/composables/useOnchange';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { createFormController } from '@/web/web/controllers/formController';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import { useOnchangeAggregation } from '@/web/web/composables/useOnchangeAggregation';
import { useFormViewLifecycle } from '@/web/web/composables/useFormViewLifecycle';
import { useFormViewSubmit } from '@/web/web/composables/useFormViewSubmit';
import { nextLocalToken } from '@/web/web/components/view/localToken';
import { createTranslate } from '@/web/web/i18n';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import { Plus, Pencil, RefreshCw, Trash2, Check, X, Copy, RotateCcw } from 'lucide-vue-next';
import type {
  FormSubmitHandler,
  FormChildSubmitApiRegistration,
  FormChildSubmitApiRegister,
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
useProvideViewScope({ kind: 'Form', mode: viewMode });
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
// Section 10: Provide field errors / client validators
// =============================
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
// Section 13: Computed exposed form data
// =============================
const exposedFormData = computed<Partial<ClientModel<T>>>(() => (controller.vm.draft as any) || {});

const lifecycleDeps = {
  // Getters keep identity reactive when the host reuses the form without remounting.
  identity: {
    get recordId() {
      return props.recordId;
    },
    get viewMode() {
      return props.viewMode;
    },
    get initialValues() {
      return props.initialValues;
    },
    get onchangeSessionId() {
      return props.onchangeSessionId;
    },
  },
  route,
  router,
  controller,
  resolvedResolveRecordIdFromRoute,
  resolvedShowMessages,
  resolvedCreateAction,
  viewMode,
  exposedFormData,
  onchangeCtrl,
  resetOnchangeAgg,
  emitCancelable,
  emit: emit as (event: string, payload?: unknown) => void,
  _t,
};

const {
  effectiveRecordId,
  handleEdit,
  handleCancel,
  handleReset,
  handleCreate,
  handleRefresh,
  handleCopy,
  handleDelete,
  watchFormIdentity,
} = useFormViewLifecycle(lifecycleDeps);

const { submitting, handleSubmit, onNativeSubmit } = useFormViewSubmit({
  controller,
  viewMode,
  loading,
  exposedFormData,
  fieldClientValidators,
  fieldErrors,
  resolvedShowMessages,
  get submitHandler() {
    return props.submitHandler;
  },
  onchangeCtrl,
  resetOnchangeAgg,
  emitCancelable,
  emit: emit as (event: string, payload?: unknown) => void,
  router,
  detailsTitle,
  _t,
});

const formBusy = computed<boolean>(() => loading.value || submitting.value);
const saveLabel = computed(() => (formBusy.value ? _t('Saving...') : _t('Save')));

watch(exposedFormData, v => emit('change', { formData: toRaw(v) as any }), { deep: true, flush: 'post' });

watchFormIdentity();

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

