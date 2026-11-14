<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer
    class="gap-5"
    :data-anchor="isSettingsPresentation ? 'choy.settings-view' : undefined"
    :data-testid="isSettingsPresentation ? 'choy-settings-view' : undefined"
  >
    <template v-if="resolvedShowHeader" #header>
      <div :data-anchor="isSettingsPresentation ? 'choy.settings.view-chrome' : 'choy.form.view-chrome'">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="resolvedShowActions || chromeTitle || showTitleOverflow" #system-actions>
            <!--
              Left chrome contract (List / Kanban / Form):
              [primary] [page title][⋮] [secondary…]
              create/edit: primary=Save/Cancel, Reset in ⋮ (no trailing secondary).
              display: primary=New, secondary=Edit/Copy/Delete after the title cluster.
              settings: primary=Save/Discard only (no CRUD / Reset overflow).
            -->
            <!-- Settings chrome: Save/Discard; dirty gates enablement. -->
            <template v-if="isSettingsPresentation && resolvedShowActions">
              <ChoyButton
                type="button"
                size="default"
                variant="default"
                data-testid="settings-save"
                :disabled="controlsLocked || !settingsDirty"
                @click="onSettingsSave"
              >
                <Check class="size-4" aria-hidden="true" />
                {{ settingsSaveLabel }}
              </ChoyButton>
              <ChoyButton
                type="button"
                size="default"
                variant="outline"
                data-testid="settings-discard"
                :disabled="controlsLocked || !settingsDirty"
                @click="onSettingsDiscard"
              >
                <X class="size-4" aria-hidden="true" />
                {{ settingsDiscardLabel }}
              </ChoyButton>
            </template>
            <!-- Primary: Save/Cancel while editing; New (or #system-actions) in display. -->
            <template v-else-if="isEditing && resolvedShowActions">
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
              v-if="chromeTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              :class="
                isSettingsPresentation
                  ? 'choy-settings-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]'
                  : 'choy-form-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]'
              "
            >
              {{ chromeTitle }}
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
            <template v-if="!isSettingsPresentation && resolvedShowActions && viewMode === 'display' && effectiveRecordId">
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
            <!-- Same tray as title/⋮ (List/Kanban secondary-actions parity) — no ActionBar separator. -->
            <slot
              v-if="!isSettingsPresentation && resolvedShowActions && viewMode === 'display'"
              name="secondary-actions"
            />
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
            <template v-if="isSettingsPresentation && settingsViewSwitcherOptions.length > 1">
              <ChoyViewSwitcher
                :active="settingsActiveView"
                :options="settingsViewSwitcherOptions"
                :aria-label="_t('View switcher')"
                @select="onSettingsViewSelect"
              />
            </template>
            <slot v-else name="statusbar" />
            <slot v-if="!isSettingsPresentation" name="button-box" />
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <ChoyViewBusyHost
      :busy="overlayBusy"
      :mode="formBusyMode"
      class="min-h-0 flex-1 overflow-y-auto"
      :data-anchor="isSettingsPresentation ? 'choy.settings.sheet' : 'choy.form-view'"
    >
      <form
        ref="formRef"
        :class="isSettingsPresentation ? 'choy-settings-view__sheet flex min-h-0 min-w-0 flex-1 flex-col gap-8' : undefined"
        :inert="(isSettingsPresentation ? overlayBusy : submitting) || undefined"
        @submit.self.prevent="isSettingsPresentation ? onSettingsSave() : onNativeSubmit()"
      >
        <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
      </form>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<style scoped>
/*
 * Settings rows often carry longer CJK labels than dense Form columns.
 * Raise the horizontal label track only under settings presentation.
 */
.choy-settings-view__sheet {
  --choy-field-label-max: 12rem;
}
</style>

<script setup lang="ts" generic="T extends BaseModel">
// =============================
// Section 1: Imports
// =============================
import { ref, computed, provide, inject, watch, toRaw, onMounted, onBeforeUnmount, getCurrentInstance, useSlots, reactive } from 'vue';
import { useControlBusy } from '@/core/web/client/loadingRegistry';
import { FIELD_CLIENT_VALIDATORS_KEY } from '@/web/web/composables/fieldClientValidation';
import { FIELD_ERROR_SPACE_KEY } from '@/web/web/composables/fieldErrorSpace';
import type { ClientModel, BaseModel, Updateable, Insertable } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';
import {
  canShowAction,
  canShowDeclaredAction,
  type ActionIdMap,
} from '@/web/web/components/view/actionVisibility';
import { resolvePageStore, useHostPageTitleInView, useRegisterPageActionTarget } from '@/web/web/composables/usePageContext';
import { useResolvedCreateAction } from '@/web/web/composables/resolveCreateRoute';
import { getDraftField, setDraftField } from '@/web/web/composables/listRowEdit';

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
import type { LocalFormStore } from '@/web/web/stores/localFormStore';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyPageTitleActions from '@/web/web/components/layout/ChoyPageTitleActions.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import ChoyViewSwitcher, { type ChoyViewSwitcherOption } from '@/web/web/components/view/ChoyViewSwitcher.vue';
import DropdownMenuItem from '@/web/web/components/vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import { Plus, Pencil, RefreshCw, Trash2, Check, X, Copy, RotateCcw } from 'lucide-vue-next';
import type {
  FormSubmitHandler,
  FormChildSubmitApiRegistration,
  FormChildSubmitApiRegister,
} from '@/web/web/components/view/formViewTypes';

export type FormPresentation = 'record' | 'settings';

defineOptions({ name: 'ChoyFormView' });

/**
 * Store-backed form engine. Requires :store or a page-provided store
 * (resolvePageStore). `presentation=settings` adds Settings chrome (Save/Discard,
 * ViewSwitcher) while reusing form-root / leave / BusyHost / control submit.
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
    /**
     * Optional cancel override (after dirty confirm). Dialog / non-route embedded
     * hosts use this instead of create→router.back or edit→beginDisplay.
     */
    cancelHandler?: () => void | Promise<void>;
    resolveRecordIdFromRoute?: boolean;
    /** Opt out of page IO action-target registration (embedded views pass false). */
    registerActionTarget?: boolean;
    /** `settings` = Settings chrome on this Form engine (Save/Discard, no CRUD). */
    presentation?: FormPresentation;
    /**
     * Settings / wizard draft bag when fields bind via form-root without a LocalFormStore.
     * Ignored for record presentation.
     */
    draft?: Record<string, unknown>;
    /**
     * Optional LocalFormStore for settings form-root (curated keys as field names).
     * When set, takes precedence over `draft` for getField/setField.
     */
    localStore?: LocalFormStore | null;
    /**
     * Page-level dirty override (Settings). When the prop is passed, leave/Cancel and
     * Save/Discard enablement use this instead of controller draft vs original.
     */
    dirty?: boolean;
    /** Settings first-load / refresh surface busy (skeleton). */
    contentLoading?: boolean;
    /** Settings Save in flight (control); locks chrome without masking the sheet. */
    controlBusy?: boolean;
    /** Optional title override (Settings mounts without relying solely on ChoyPage). */
    pageTitle?: string;
    saveLabel?: string;
    discardLabel?: string;
    activeView?: string;
    viewSwitcherOptions?: ChoyViewSwitcherOption[];
  }>(),
  {
    createAction: undefined,
    onchangeSessionId: undefined,
    onchangeDebounceMs: 150,
    onchangeImmediateFirst: false,
    submitHandler: undefined,
    cancelHandler: undefined,
    // Keep omitted as undefined (not false) so matching page-store views auto-register.
    registerActionTarget: undefined,
    presentation: 'record',
    localStore: null,
    contentLoading: false,
    controlBusy: false,
    activeView: 'settings',
    viewSwitcherOptions: () => [],
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
const hasDirtyProp = hasExplicitProp('dirty');

const isSettingsPresentation = computed(() => props.presentation === 'settings');

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
  /**
   * Display-only CTAs after the title/⋮ cluster (and after Edit/Copy/Delete when shown).
   * Same tray as List/Kanban `#secondary-actions` — no ActionBar separator.
   */
  'secondary-actions'(): any;
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
  /** Settings presentation: Save clicked (host persists). */
  (e: 'save'): void;
  /** Settings presentation: Discard confirmed (host reloads baseline). */
  (e: 'discard'): void;
  /** Settings presentation: ViewSwitcher select after dirty confirm. */
  (e: 'view-select', value: string): void;
}>();

// =============================
// Section 7: Cancelable emit helper
// =============================
const { emitCancelable } = useCancelableEmit(emit as any);

// =============================
// Section 8: Controller & view context provisioning
// =============================
const formRef = ref<HTMLFormElement | null>(null);
// Mutating slots are opt-in (empty action id → hidden). Refresh stays ungated by default.
const canCreate = computed(() => canShowDeclaredAction(props.actionIds?.create, props.hasAction));
const canEdit = computed(() => canShowDeclaredAction(props.actionIds?.edit, props.hasAction));
const canDelete = computed(() => canShowDeclaredAction(props.actionIds?.delete, props.hasAction));
const canCopy = computed(() => canShowDeclaredAction(props.actionIds?.copy, props.hasAction));
const canRefresh = computed(() => canShowAction(props.actionIds?.refresh, props.hasAction));
// Form controller.
const controller = createFormController(store as any);
const viewMode = computed<ViewMode>(() => {
  if (isSettingsPresentation.value) return 'edit';
  return controller.vm.mode as ViewMode;
});
const loading = computed<boolean>(() => !!controller.vm.loading);
useProvideViewScope({
  kind: computed(() => (isSettingsPresentation.value ? 'Settings' : 'Form')),
  mode: viewMode,
});

/** Mutable bag used when settings presentation passes a plain `draft` object. */
const settingsDraftBag = reactive<Record<string, unknown>>({});
watch(
  // Reference only: setField writes back into props.draft, so deep:true would loop.
  () => props.draft,
  (next) => {
    if (!isSettingsPresentation.value) return;
    if (!next || typeof next !== 'object') return;
    for (const key of Object.keys(settingsDraftBag)) {
      if (!(key in next)) delete settingsDraftBag[key];
    }
    Object.assign(settingsDraftBag, next);
  },
  { immediate: true },
);

/**
 * Settings chrome: form-root binds LocalFormStore or draft bag so curated Settings
 * fields share one write path. Record Form keeps controller.provideToChildren().
 */
if (isSettingsPresentation.value) {
  provide('form-root', {
    get mode() {
      return 'edit' as const;
    },
    get draft() {
      if (props.localStore) return props.localStore.getValues();
      if (props.draft && typeof props.draft === 'object') return settingsDraftBag;
      return controller.vm.draft;
    },
    getField(path: string) {
      if (props.localStore) return props.localStore.getField(path);
      if (props.draft && typeof props.draft === 'object') return getDraftField(settingsDraftBag, path);
      return controller.vm.draft ? (controller.vm.draft as any)[path] : undefined;
    },
    setField(path: string, value: unknown) {
      if (props.localStore) {
        props.localStore.setField(path, value);
        return;
      }
      if (props.draft && typeof props.draft === 'object') {
        setDraftField(settingsDraftBag, path, value);
        setDraftField(props.draft, path, value);
        return;
      }
      if (!controller.vm.draft) (controller.vm as any).draft = {};
      (controller.vm.draft as any)[path] = value;
    },
    onChange(_cb: (nextDraft: unknown) => void) {},
  });
} else {
  controller.provideToChildren();
}

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
  enabled: () => !isEmbedded.value && !isSettingsPresentation.value,
});
const slots = useSlots();
const { pageTitle: hostedPageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(resolvedShowHeader);
const chromeTitle = computed(() => {
  const override = String(props.pageTitle || '').trim();
  if (override) return override;
  return String(hostedPageTitle.value || '').trim();
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

// Client-side RuleItem validators registered by ChoyFieldBase.
const fieldClientValidators = new Map<string, () => Promise<string>>();
provide(FIELD_CLIENT_VALIDATORS_KEY, fieldClientValidators);
// Multi-column form + display/edit: absolute errors avoid row jump; fields may override.
provide(FIELD_ERROR_SPACE_KEY, 'overlay');

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
const exposedFormData = computed<Partial<ClientModel<T>>>(() => {
  if (isSettingsPresentation.value) {
    if (props.localStore) return props.localStore.getValues() as Partial<ClientModel<T>>;
    if (props.draft && typeof props.draft === 'object') {
      return settingsDraftBag as Partial<ClientModel<T>>;
    }
  }
  return (controller.vm.draft as any) || {};
});

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
  isFormDirty,
  confirmDiscardIfDirty,
  runWhileLeaving,
  handleEdit,
  handleCancel,
  handleReset,
  handleCreate,
  handleRefresh,
  handleCopy,
  handleDelete,
  watchFormIdentity,
} = useFormViewLifecycle({
  ...lifecycleDeps,
  get cancelHandler() {
    return props.cancelHandler;
  },
  externalDirty: () => (hasDirtyProp ? props.dirty === true : undefined),
  leaveBlocked: () => isSettingsPresentation.value && (props.controlBusy === true || submitting.value),
});

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

/** Binary/Image fields use this to show Attachment state=uploading during Prepare/PUT/Finalize. */
provide('form-submitting', submitting);

const formBusyTarget = computed(() => `form:${String((store as any)?.storeId ?? (store as any)?.model ?? 'default')}`);
const formControlBusy = useControlBusy(formBusyTarget);
const settingsControlBusy = computed(
  () => props.controlBusy === true || submitting.value || formControlBusy.value.busy,
);
/** Surface load only — submit/delete/settings Save are control (no content spinner mask). */
const overlayBusy = computed<boolean>(() => {
  if (isSettingsPresentation.value) {
    return props.contentLoading === true;
  }
  return loading.value && !submitting.value && !formControlBusy.value.busy;
});
const formBusyMode = computed<'stale' | 'skeleton' | 'block'>(() => {
  // Prefer block (Loader2) for first paint; stale when a record frame already exists.
  if (isSettingsPresentation.value) return 'block';
  return controller.vm.draft || controller.vm.original ? 'stale' : 'block';
});
const controlsLocked = computed<boolean>(() => {
  if (isSettingsPresentation.value) return settingsControlBusy.value;
  return loading.value || submitting.value || formControlBusy.value.busy;
});
const saveLabel = computed(() => (submitting.value ? _t('Saving...') : _t('Save')));
// Explicit dirty=false stays false; omitted dirty falls back to controller draft dirty.
const settingsDirty = computed(() => (hasDirtyProp ? props.dirty === true : isFormDirty()));
const settingsSaveLabel = computed(() => {
  if (props.saveLabel) return props.saveLabel;
  return settingsControlBusy.value ? _t('Saving...') : _t('Save');
});
const settingsDiscardLabel = computed(() => props.discardLabel || _t('Discard'));
const settingsActiveView = computed(() => props.activeView || 'settings');
const settingsViewSwitcherOptions = computed(() => {
  const opts = props.viewSwitcherOptions || [];
  if (!controlsLocked.value) return opts;
  return opts.map((opt) => ({ ...opt, disabled: true }));
});

function onSettingsSave() {
  if (controlsLocked.value || !settingsDirty.value) return;
  emit('save');
}

async function onSettingsDiscard() {
  if (controlsLocked.value || !settingsDirty.value) return;
  // cancelHandler path (ChoySettingsView) emits discard itself — avoid a second emit.
  if (props.cancelHandler) {
    await handleCancel();
    return;
  }
  if (!(await confirmDiscardIfDirty())) return;
  emit('discard');
}

async function onSettingsViewSelect(value: string) {
  if (controlsLocked.value) return;
  // Same view: do not open leave-guard / 1s fallback for a no-op switch.
  if (value === settingsActiveView.value) return;
  if (!(await confirmDiscardIfDirty())) return;
  // Skip leave-guard while the host pushes the sibling view route.
  await runWhileLeaving(async () => {
    // No router: do not hold leaving for the 1s fallback with nothing to await.
    if (!router?.afterEach) {
      emit('view-select', value);
      return;
    }
    await new Promise<void>((resolve) => {
      let settled = false;
      let removeAfter: (() => void) | undefined;
      let fallbackTimer: ReturnType<typeof setTimeout> | undefined;
      const finish = () => {
        if (settled) return;
        settled = true;
        removeAfter?.();
        if (fallbackTimer != null) clearTimeout(fallbackTimer);
        resolve();
      };
      // Attach before emit so a synchronous host navigation still settles leaving.
      removeAfter = router.afterEach(() => finish());
      fallbackTimer = setTimeout(finish, 1000);
      emit('view-select', value);
    });
  });
}

const isEditing = computed(() => viewMode.value === 'edit' || viewMode.value === 'create');
const showRefreshInOverflow = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    resolvedShowHeader.value &&
    canRefresh.value &&
    viewMode.value === 'display' &&
    !!effectiveRecordId.value,
);
/** Reset lives in ⋮ during create/edit so the left bar stays [Save][Cancel][title][⋮]. */
const showResetInOverflow = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    resolvedShowHeader.value &&
    isEditing.value,
);
const showTitleOverflow = computed(
  () =>
    !isSettingsPresentation.value &&
    (!!pageTitleActionsIo.value ||
      showRefreshInOverflow.value ||
      showResetInOverflow.value ||
      !!slots['overflow-actions']),
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

/**
 * Runs every mounted field's client rules (revealing their messages) without submitting.
 * Resolves false when any rule fails or a server field error is still shown.
 */
async function validate(): Promise<boolean> {
  let ok = fieldErrors.value.size === 0;
  for (const run of fieldClientValidators.values()) {
    if (await run()) ok = false;
  }
  return ok;
}

defineExpose({
  submit: handleSubmit,
  validate,
  /** Runs dirty confirmation, then cancelHandler / route back / display reload. */
  cancel: handleCancel,
  refresh: handleRefresh,
  reset: handleReset,
  copy: handleCopy,
  getFormData: () => toRaw(exposedFormData.value) as any,
  getViewMode: () => viewMode.value,
  isLoading: () => loading.value,
});
</script>

