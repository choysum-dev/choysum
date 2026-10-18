<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer class="gap-5">
    <template v-if="resolvedShowHeader" #header>
      <div data-anchor="choy.form.view-chrome">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="resolvedShowActions || pageTitle || showTitleOverflow" #system-actions>
            <!--
              Left chrome contract (List / Kanban / Form):
              [primary] [page title][⋮] [secondary…]
              create/edit: primary=Save/Cancel, Reset in ⋮ (no trailing secondary).
              display: primary=New, secondary=Edit/Copy/Delete after the title cluster.
            -->
            <!-- Primary: Save/Cancel while editing; New (or #system-actions) in display. -->
            <template v-if="isEditing">
              <ChoyButton type="button" size="default" variant="default" :disabled="controlsLocked" @click="handleSubmit">
                <Check class="size-4" aria-hidden="true" />
                {{ saveLabel }}
              </ChoyButton>
              <ChoyButton type="button" size="default" variant="outline" :disabled="controlsLocked" @click="handleCancel">
                <X class="size-4" aria-hidden="true" />
                {{ _t('Cancel') }}
              </ChoyButton>
            </template>
            <slot v-else-if="resolvedShowActions" name="system-actions">
              <ChoyButton
                v-if="viewMode === 'display' && effectiveRecordId && resolvedCreateAction && canCreate"
                type="button"
                size="default"
                variant="default"
                data-testid="form-primary-new"
                @click="handleCreate"
              >
                <Plus class="size-4" aria-hidden="true" />
                {{ _t('New') }}
              </ChoyButton>
            </slot>

            <h1
              v-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              class="choy-form-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold tracking-tight text-foreground sm:max-w-[14rem]"
            >
              {{ pageTitle }}
            </h1>
            <ChoyPageTitleActions
              v-if="showTitleOverflow"
              :has-io-menu="!!pageTitleActionsIo"
              :action-import="pageTitleActionsIo?.actionImport ?? false"
              :action-export="pageTitleActionsIo?.actionExport ?? false"
              :action-import-upload-hint="pageTitleActionsIo?.actionImportUploadHint"
              :action-import-column-mapping="pageTitleActionsIo?.actionImportColumnMapping"
              :action-list-ref="(pageTitleActionsIo?.actionListRef as any)"
              :action-company-id="pageTitleActionsIo?.actionCompanyId"
              :store="store"
            >
              <template
                v-if="showRefreshInOverflow || showResetInOverflow || $slots['overflow-actions']"
                #overflow
              >
                <DropdownMenuItem
                  v-if="showRefreshInOverflow"
                  data-testid="form-overflow-refresh"
                  @select="handleRefresh"
                >
                  <RefreshCw class="size-4" aria-hidden="true" />
                  {{ _t('Refresh') }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  v-if="showResetInOverflow"
                  data-testid="form-overflow-reset"
                  :disabled="controlsLocked"
                  @select="handleReset"
                >
                  <RotateCcw class="size-4" aria-hidden="true" />
                  {{ _t('Reset') }}
                </DropdownMenuItem>
                <slot name="overflow-actions" />
              </template>
            </ChoyPageTitleActions>

            <!-- Secondary (display only): Edit/Copy/Delete after the title cluster. -->
            <template v-if="resolvedShowActions && viewMode === 'display' && effectiveRecordId">
              <ChoyButton
                v-if="canEdit"
                type="button"
                size="default"
                variant="outline"
                data-testid="form-secondary-edit"
                @click="handleEdit"
              >
                <Pencil class="size-4" aria-hidden="true" />
                {{ _t('Edit') }}
              </ChoyButton>
              <ChoyButton
                v-if="canCopy"
                type="button"
                size="default"
                variant="outline"
                data-testid="form-secondary-copy"
                @click="handleCopy"
              >
                <Copy class="size-4" aria-hidden="true" />
                {{ _t('Copy') }}
              </ChoyButton>
              <ChoyButton
                v-if="canDelete"
                type="button"
                size="default"
                variant="outline-destructive"
                data-testid="form-secondary-delete"
                @click="handleDelete"
              >
                <Trash2 class="size-4" aria-hidden="true" />
                {{ _t('Delete') }}
              </ChoyButton>
            </template>
          </template>
          <!-- Only register when filled — empty tray would paint a dangling ActionBar separator. -->
          <template v-if="resolvedShowActions && !!slots['user-actions']" #user-actions>
            <slot name="user-actions" />
          </template>
          <!-- Keep #center registered so the thirds grid does not collapse. -->
          <template #center>
            <div class="min-w-0" />
          </template>
          <template #header-right>
            <slot name="statusbar" />
            <slot name="button-box" />
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <ChoyViewBusyHost
      :busy="overlayBusy"
      class="min-h-0 flex-1 overflow-y-auto p-5"
      data-anchor="choy.form-view"
    >
      <form ref="formRef" :inert="submitting || undefined" @submit.self.prevent="onNativeSubmit">
        <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
      </form>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
// =============================
// Section 1: Imports
// =============================
import { ref, computed, provide, inject, watch, toRaw, onMounted, onBeforeUnmount, getCurrentInstance, useSlots } from 'vue';
import { FIELD_CLIENT_VALIDATORS_KEY } from '@/web/web/composables/fieldClientValidation';
import type { ClientModel, BaseModel, Updateable, Insertable } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';
import { resolvePageStore, useHostPageTitleInView, useRegisterPageActionTarget } from '@/web/web/composables/usePageContext';
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
import ChoyPageTitleActions from '@/web/web/components/layout/ChoyPageTitleActions.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import DropdownMenuItem from '@/web/web/components/vendor/ui/dropdown-menu/DropdownMenuItem.vue';
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
    /** Opt out of page IO action-target registration (embedded views pass false). */
    registerActionTarget?: boolean;
  }>(),
  {
    createAction: undefined,
    onchangeSessionId: undefined,
    onchangeDebounceMs: 150,
    onchangeImmediateFirst: false,
    submitHandler: undefined,
    // Keep omitted as undefined (not false) so matching page-store views auto-register.
    registerActionTarget: undefined,
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
// Section 4: View layout notes
// =============================
// The shell fills the workspace. Field, notebook, and chatter content scroll
// inside the form body rather than the page shell.

// =============================
// Section 5: Slots declaration
// =============================
defineSlots<{
  breadcrumb(): any;
  /** Primary actions before the page title (default: New in display). */
  'system-actions'(): any;
  'user-actions'(): any;
  statusbar(): any;
  'button-box'(): any;
  'header-right'(): any;
  'overflow-actions'(): any;
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
const slots = useSlots();
const { pageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(resolvedShowHeader);

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
  fieldsMetadata: () => (store as { fieldsMetadata?: Record<string, { isReadonly?: boolean }> }).fieldsMetadata,
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

const overlayBusy = computed<boolean>(() => loading.value && !submitting.value);
const controlsLocked = computed<boolean>(() => loading.value || submitting.value);
const saveLabel = computed(() => (submitting.value ? _t('Saving...') : _t('Save')));
const isEditing = computed(() => viewMode.value === 'edit' || viewMode.value === 'create');
const showRefreshInOverflow = computed(
  () => resolvedShowActions.value && resolvedShowHeader.value && canRefresh.value && viewMode.value === 'display' && !!effectiveRecordId.value,
);
/** Reset lives in ⋮ during create/edit so the left bar stays [Save][Cancel][title][⋮]. */
const showResetInOverflow = computed(
  () => resolvedShowActions.value && resolvedShowHeader.value && isEditing.value,
);
const showTitleOverflow = computed(
  () =>
    !!pageTitleActionsIo.value ||
    showRefreshInOverflow.value ||
    showResetInOverflow.value ||
    !!slots['overflow-actions'],
);

useRegisterPageActionTarget({
  store,
  enabled: () => (isEmbedded.value ? false : props.registerActionTarget),
  target: {
    get selectedItems() {
      const id = effectiveRecordId.value || (controller.vm.original as { Id?: string } | null)?.Id;
      return id ? [{ Id: String(id) }] : [];
    },
    refresh: () => handleRefresh(),
  },
});

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

