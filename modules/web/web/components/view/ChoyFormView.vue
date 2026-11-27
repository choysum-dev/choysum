<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewBase
    :data-anchor="isSettingsPresentation ? 'choy.settings-view' : undefined"
    :data-testid="isSettingsPresentation ? 'choy-settings-view' : undefined"
    :chrome-anchor="isSettingsPresentation ? 'choy.settings.view-chrome' : 'choy.form.view-chrome'"
    :class="formViewBaseClass"
    :show-header="resolvedShowHeader"
    :show-actions="resolvedShowActions"
    :page-title="chromeTitle || undefined"
    :page-title-id="pageTitleId"
    :page-title-class="
      isSettingsPresentation ? 'choy-settings-view__page-title' : 'choy-form-view__page-title'
    "
    :page-title-actions-io="pageTitleActionsIo"
    :show-title-overflow="showTitleOverflow"
    :store="store"
    :busy="overlayBusy"
    :mode="formBusyMode"
    :sheet-surface="!isEmbedded"
    :sheet-pad="isEmbedded ? 'none' : 'form'"
    :busy-host-class="formSheetHostClass"
    :busy-host-attrs="{
      'data-anchor': isSettingsPresentation ? 'choy.settings.sheet' : 'choy.form-view',
      'data-testid': isSettingsPresentation ? 'settings-sheet' : 'form-sheet',
    }"
    :error="formSurfaceError"
    :has-frame="formHasFrame"
    :has-settled="formHasSettled"
    :error-fallback-title="_t('Failed to load')"
    @retry="onFormSurfaceRetry"
    @cancel="onFormSurfaceCancel"
  >
    <!--
      Action Bar contract:
      [identity: title] [主 → 次 → ⋯] | [找] | [看]
      create/edit: platform Save/Cancel; Reset/New/Copy/Delete in ⋮;
        statusbar-buttons hidden; statusbar-status ribbon stays when present.
      display: platform Edit; domain flow → optional statusbar (#statusbar-buttons | #statusbar-status).
      settings: Save/Discard only (no CRUD / Reset overflow / statusbar).
    -->
    <template v-if="resolvedShowActions" #system-actions>
      <!-- Settings chrome: Save/Discard; dirty gates enablement. -->
      <template v-if="isSettingsPresentation && resolvedShowActions">
        <ChoyActionButton
          type="button"
          size="default"
          variant="soft-primary"
          data-testid="settings-save"
          :disabled="!settingsDirty"
          :busy="controlsLocked"
          @execute="onSettingsSaveGated"
        >
          <Check class="size-4" aria-hidden="true" />
          {{ settingsSaveLabel }}
        </ChoyActionButton>
        <ChoyActionButton
          type="button"
          size="default"
          variant="ghost"
          data-testid="settings-discard"
          :requires-connectivity="false"
          :disabled="!settingsDirty"
          :busy="controlsLocked"
          @execute="onSettingsDiscard"
        >
          <X class="size-4" aria-hidden="true" />
          {{ settingsDiscardLabel }}
        </ChoyActionButton>
      </template>
      <!-- Editing: Save/Cancel own the platform tray; domain tray is hidden. -->
      <template v-else-if="isEditing && resolvedShowActions">
        <ChoyActionButton
          type="button"
          size="default"
          variant="soft-primary"
          :busy="controlsLocked"
          @execute="onSaveExecute"
        >
          <Check class="size-4" aria-hidden="true" />
          {{ saveLabel }}
        </ChoyActionButton>
        <ChoyActionButton
          type="button"
          size="default"
          variant="ghost"
          :requires-connectivity="false"
          :busy="controlsLocked"
          @execute="handleCancel"
        >
          <X class="size-4" aria-hidden="true" />
          {{ _t('Cancel') }}
        </ChoyActionButton>
      </template>
      <!-- Display: platform Edit stays soft-primary; contextual CTAs go to #center. -->
      <template v-else-if="resolvedShowActions">
        <ChoyActionButton
          v-if="viewMode === 'display' && effectiveRecordId && canEdit"
          type="button"
          size="default"
          variant="soft-primary"
          data-testid="form-primary-edit"
          :action-id="actionIds?.edit"
          :has-action="hasAction"
          @execute="handleEdit"
        >
          <Pencil class="size-4" aria-hidden="true" />
          {{ _t('Edit') }}
        </ChoyActionButton>
        <!-- Optional platform extras only — domain CTAs use #statusbar-buttons. -->
        <slot
          name="system-actions"
          :form-data="exposedFormData"
          :record-id="effectiveRecordId"
          :view-mode="viewMode"
          :loading="loading"
        />
      </template>
    </template>

    <template
      v-if="
        showRefreshInOverflow ||
        showResetInOverflow ||
        showFormLifecycleInOverflow ||
        $slots['overflow-actions']
      "
      #overflow-actions
    >
      <!-- Lifecycle first; platform tools (Reset / Refresh) last — same as List IO then Refresh. -->
      <ChoyActionMenuItem
        v-if="showOverflowNew"
        data-testid="form-overflow-new"
        :action-id="actionIds?.create"
        :has-action="hasAction"
        @execute="handleCreate"
      >
        <Plus class="size-4" aria-hidden="true" />
        {{ _t('New') }}
      </ChoyActionMenuItem>
      <ChoyActionMenuItem
        v-if="showOverflowCopy"
        data-testid="form-overflow-copy"
        :action-id="actionIds?.copy"
        :has-action="hasAction"
        @execute="handleCopy"
      >
        <Copy class="size-4" aria-hidden="true" />
        {{ _t('Copy') }}
      </ChoyActionMenuItem>
      <ChoyActionMenuItem
        v-if="showOverflowDelete"
        data-testid="form-overflow-delete"
        class="text-destructive focus:text-destructive"
        :action-id="actionIds?.delete"
        :has-action="hasAction"
        @execute="handleDelete"
      >
        <Trash2 class="size-4" aria-hidden="true" />
        {{ _t('Delete') }}
      </ChoyActionMenuItem>
      <slot name="overflow-actions" />
      <!-- Match List: separator before platform tools (Reset / Refresh). -->
      <DropdownMenuSeparator
        v-if="
          (showResetInOverflow || showRefreshInOverflow) &&
          (showFormLifecycleInOverflow || !!$slots['overflow-actions'])
        "
      />
      <ChoyActionMenuItem
        v-if="showResetInOverflow"
        data-testid="form-overflow-reset"
        :requires-connectivity="false"
        :disabled="controlsLocked"
        @execute="handleReset"
      >
        <RotateCcw class="size-4" aria-hidden="true" />
        {{ _t('Reset') }}
      </ChoyActionMenuItem>
      <ChoyViewOverflowRefreshItem
        v-if="showRefreshInOverflow"
        test-id="form-overflow-refresh"
        @select="handleRefresh"
      />
    </template>

    <!-- Leading secondary (rare); stays beside platform via ViewBase. -->
    <template
      v-if="
        !isSettingsPresentation &&
        resolvedShowActions &&
        !isEditing &&
        !!slots['secondary-actions']
      "
      #secondary-actions
    >
      <slot
        name="secondary-actions"
        :form-data="exposedFormData"
        :record-id="effectiveRecordId"
        :view-mode="viewMode"
        :loading="loading"
      />
    </template>

    <!--
      Optional Form statusbar under Action Bar:
      statusbar = #statusbar-buttons | #statusbar-status.
      Buttons: display only (side-path workflow). Status ribbon: stays across
      display/edit/create when slotted (identity); Progress stays mounted for
      cancel/refresh beginDisplay browse selection.
      Root hidden when editing and only buttons exist (no empty chrome).
      Omitted when settings or both sides empty.
      #contextual-actions / #user-actions → buttons (deprecated).
      #statusbar → status (deprecated).
    -->
    <template
      v-if="!isSettingsPresentation && (hasStatusbarButtons() || hasStatusbarStatus())"
      #header-extra
    >
      <div
        v-show="hasStatusbarStatus() || (!isEditing && hasStatusbarButtons())"
        data-anchor="choy.form.statusbar"
        data-testid="form-statusbar"
        class="flex min-h-control-sm min-w-0 w-full flex-nowrap items-center justify-between gap-3"
        role="region"
        :aria-label="_t('Status bar')"
      >
        <div
          v-if="!isEditing && hasStatusbarButtons()"
          data-anchor="choy.form.statusbar-buttons"
          data-testid="form-statusbar-buttons"
          class="flex min-w-0 flex-nowrap items-center gap-2 overflow-x-auto"
          role="toolbar"
          :aria-label="_t('Document actions')"
        >
          <slot
            name="statusbar-buttons"
            :form-data="exposedFormData"
            :record-id="effectiveRecordId"
            :view-mode="viewMode"
            :loading="loading"
          />
          <slot
            name="contextual-actions"
            :form-data="exposedFormData"
            :record-id="effectiveRecordId"
            :view-mode="viewMode"
            :loading="loading"
          />
          <slot
            v-if="resolvedShowActions"
            name="user-actions"
            :form-data="exposedFormData"
            :record-id="effectiveRecordId"
            :view-mode="viewMode"
            :loading="loading"
          />
        </div>
        <div
          v-if="hasStatusbarStatus()"
          data-anchor="choy.form.statusbar-status"
          data-testid="form-statusbar-status"
          class="ml-auto flex min-w-0 shrink-0 items-center justify-end overflow-x-auto"
        >
          <slot name="statusbar-status" />
          <slot name="statusbar" />
        </div>
      </div>
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
      <slot v-if="!isSettingsPresentation" name="button-box" />
      <ChoyFormPager
        v-if="!isSettingsPresentation && formRecordPager.visible.value"
        :current="formRecordPager.current.value"
        :total="formRecordPager.total.value"
        :can-prev="formRecordPager.canPrev.value"
        :can-next="formRecordPager.canNext.value"
        :busy="formRecordPager.busy.value || sessionUnreachable"
        @prev="formRecordPager.goPrev()"
        @next="formRecordPager.goNext()"
        @goto="formRecordPager.goTo($event)"
      />
      <slot name="header-right" />
    </template>

    <form
      ref="formRef"
      :class="isSettingsPresentation ? 'choy-settings-view__sheet flex min-h-0 min-w-0 flex-1 flex-col gap-8' : undefined"
      :inert="(isSettingsPresentation ? overlayBusy : submitting) || undefined"
      @submit.self.prevent="isSettingsPresentation ? onSettingsSaveGated() : onNativeSubmitGated()"
    >
      <slot :form-data="exposedFormData" :view-mode="viewMode" :loading="loading" />
    </form>
  </ChoyViewBase>
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
import type { ModelStore } from '@/web/web/stores/modelStore';
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
import { useFormRecordPager } from '@/web/web/composables/useFormRecordPager';
import { useFormViewSubmit } from '@/web/web/composables/useFormViewSubmit';
import { nextLocalToken } from '@/web/web/components/view/localToken';
import { createTranslate } from '@/web/web/i18n';
import { cn } from '@/web/web/lib/utils';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import ChoyActionButton from '@/web/web/components/button/ChoyActionButton.vue';
import ChoyActionMenuItem from '@/web/web/components/button/ChoyActionMenuItem.vue';
import { DropdownMenuSeparator } from '@/web/web/components/vendor/ui/dropdown-menu';
import ChoyViewBase from '@/web/web/components/view/ChoyViewBase.vue';
import { isConnectivityUnreachable } from '@/web/web/stores/connectivityStore';
import { runUnlessSessionUnreachable } from './formOfflineSaveGate';
import ChoyViewOverflowRefreshItem from '@/web/web/components/view/ChoyViewOverflowRefreshItem.vue';
import ChoyFormPager from '@/web/web/components/view/ChoyFormPager.vue';
import ChoyViewSwitcher, { type ChoyViewSwitcherOption } from '@/web/web/components/view/ChoyViewSwitcher.vue';
import { Plus, Pencil, Trash2, Check, X, Copy, RotateCcw } from 'lucide-vue-next';
import { useViewTitleOverflow } from '@/web/web/components/view/useViewTitleOverflow';
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
    store?: ModelStore<T>;
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
     * Settings / wizard draft bag when fields bind via form-root without a UiModelStore.
     * Ignored for record presentation.
     */
    draft?: Record<string, unknown>;
    /**
     * Optional UiModelStore for settings form-root (curated keys as field names).
     * When set, takes precedence over `draft` for getField/setField.
     */
    localStore?: UiModelStore | null;
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
  /**
   * Optional platform extras after Edit (display). Prefer domain CTAs in #statusbar-buttons.
   * Hidden while create/edit (Save/Cancel own the platform tray).
   */
  'system-actions'(props: {
    formData: Partial<ClientModel<T>>;
    recordId?: string;
    viewMode: ViewMode;
    loading: boolean;
  }): any;
  /**
   * Leading secondary (rare on Form). Stays in the left secondary tray via ViewBase.
   */
  'secondary-actions'(props: {
    formData: Partial<ClientModel<T>>;
    recordId?: string;
    viewMode: ViewMode;
    loading: boolean;
  }): any;
  /**
   * Domain workflow actions on the Form statusbar (left; display only).
   */
  'statusbar-buttons'(props: {
    formData: Partial<ClientModel<T>>;
    recordId?: string;
    viewMode: ViewMode;
    loading: boolean;
  }): any;
  /**
   * Status stage ribbon on the Form statusbar (right; stays in edit/create).
   */
  'statusbar-status'(): any;
  /**
   * @deprecated Prefer #statusbar-buttons (still rendered on statusbar left).
   */
  'contextual-actions'(props: {
    formData: Partial<ClientModel<T>>;
    recordId?: string;
    viewMode: ViewMode;
    loading: boolean;
  }): any;
  /**
   * @deprecated Alias of #statusbar-buttons / #contextual-actions.
   */
  'user-actions'(props: {
    formData: Partial<ClientModel<T>>;
    recordId?: string;
    viewMode: ViewMode;
    loading: boolean;
  }): any;
  /**
   * @deprecated Prefer #statusbar-status.
   */
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
  /** Soft-fail overlay Cancel — embedded/dialog hosts typically close. */
  (e: 'surface-cancel'): void;
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
/** Display load failure for BusyHost; create/edit drafts are frames, not load errors. */
const formSurfaceError = computed(() =>
  viewMode.value === 'create' ? null : controller.vm.error ?? null,
);
const formHasFrame = computed(
  () => !!(controller.vm.draft || controller.vm.original),
);
const formHasSettled = computed(() => !!controller.vm.hasAppliedOnce);
useProvideViewScope({
  kind: computed(() => (isSettingsPresentation.value ? 'Settings' : 'Form')),
  mode: viewMode,
});

/** Mutable bag used when settings presentation passes a plain `draft` object. */
const settingsDraftBag = reactive<Record<string, unknown>>({});
watch(
  // Reference only: setField writes back into props.draft, so deep:true would loop.
  // Hosts that reload/reset baseline must replace the draft object (new reference),
  // not mutate the same object in place.
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
 * Settings chrome: form-root binds UiModelStore or draft bag so curated Settings
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
/**
 * Page Form: gap above the sheet matches sheet gutter (section-gap / gap-2).
 * Paper shell + in-paper pad come from ViewBase sheetSurface / sheetPad.
 * Embedded stays flat (no double-frame).
 */
const formViewBaseClass = computed(() =>
  !isEmbedded.value ? 'choy-form-view gap-2' : 'choy-form-view',
);
const formSheetHostClass = computed(() =>
  cn('min-h-0 flex-1 overflow-y-auto', !isEmbedded.value && 'choy-form-view__sheet'),
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
  // Same-component VIEW_SCOPE inject cannot see Form's own provide — pass mode explicitly.
  viewMode,
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
  fieldsMetadata: () => (store as { metadata?: { fields?: Record<string, { isReadonly?: boolean }> } }).metadata?.fields,
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

const formRecordPager = useFormRecordPager({
  model: () => String((store as { modelName?: string }).modelName || '').trim() || undefined,
  recordId: () => effectiveRecordId.value,
  isCreate: () => viewMode.value === 'create',
  router,
  store: store as any,
  confirmDiscardIfDirty,
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
});

function onSaveExecute() {
  return runUnlessSessionUnreachable(() => handleSubmit());
}
function onNativeSubmitGated() {
  runUnlessSessionUnreachable(() => onNativeSubmit());
}
function onSettingsSaveGated() {
  return runUnlessSessionUnreachable(() => onSettingsSave());
}


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
/** Pager host offline gate (D10); ActionButton gates Save/Edit itself. */
const sessionUnreachable = computed(() => isConnectivityUnreachable());
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
      // Ignore failed navigations so an abort does not clear leaving early.
      // Listener errors are handled by Vue (not rethrown here); the 1s fallback
      // still clears afterEach/timer if no navigation commits.
      removeAfter = router.afterEach((_to, _from, failure) => {
        if (failure) return;
        finish();
      });
      fallbackTimer = setTimeout(finish, 1000);
      emit('view-select', value);
    });
  });
}

const isEditing = computed(() => viewMode.value === 'edit' || viewMode.value === 'create');

/**
 * Slot presence is read during render (not via computed) so production builds
 * re-evaluate when a parent adds/removes conditional named slots.
 */
function hasStatusbarButtons(): boolean {
  return (
    !!slots['statusbar-buttons'] ||
    !!slots['contextual-actions'] ||
    (resolvedShowActions.value && !!slots['user-actions'])
  );
}
function hasStatusbarStatus(): boolean {
  return !!slots['statusbar-status'] || !!slots.statusbar;
}
/** Reset lives in ⋮ during create/edit so the left bar stays [Save][Cancel][title][⋮]. */
const showResetInOverflow = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    resolvedShowHeader.value &&
    isEditing.value,
);
const showOverflowNew = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    viewMode.value === 'display' &&
    !!resolvedCreateAction.value &&
    canCreate.value,
);
const showOverflowCopy = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    viewMode.value === 'display' &&
    !!effectiveRecordId.value &&
    canCopy.value,
);
const showOverflowDelete = computed(
  () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    viewMode.value === 'display' &&
    !!effectiveRecordId.value &&
    canDelete.value,
);
const showFormLifecycleInOverflow = computed(
  () => showOverflowNew.value || showOverflowCopy.value || showOverflowDelete.value,
);
const { showRefreshInOverflow, showTitleOverflow } = useViewTitleOverflow({
  pageTitleActionsIo,
  showRefresh: () =>
    !isSettingsPresentation.value &&
    resolvedShowActions.value &&
    resolvedShowHeader.value &&
    canRefresh.value &&
    viewMode.value === 'display' &&
    !!effectiveRecordId.value,
  showReset: showResetInOverflow,
  hasOverflowSlot: () => !!slots['overflow-actions'] || showFormLifecycleInOverflow.value,
  enabled: () => !isSettingsPresentation.value,
});

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
      isDirty: isFormDirty,
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

/** Reload display draft from the server without the overflow-refresh success toast. */
async function reloadDisplay(): Promise<void> {
  const id = effectiveRecordId.value;
  if (!id) return;
  // Do not replace an in-progress edit/create draft (e.g. Activate completed while editing).
  if (viewMode.value === 'edit' || viewMode.value === 'create') return;
  await controller.beginDisplay(id);
}

async function onFormSurfaceRetry(): Promise<void> {
  const id = effectiveRecordId.value;
  if (!id || viewMode.value === 'create') return;
  // Soft-stale Retry after a failed save must not wipe the draft silently.
  if (!(await confirmDiscardIfDirty())) return;
  try {
    await controller.beginDisplay(id);
  } catch {
    // beginDisplay sets vm.error; Host keeps surface failure UI.
  }
  // Only re-enter edit when reload failed (successful beginDisplay already owns mode/draft).
  if (
    (props.viewMode as any) === 'edit' &&
    controller.vm.error != null &&
    controller.vm.original
  ) {
    controller.beginEdit();
  }
}

function onFormSurfaceCancel(): void {
  // Keep draft/original; clear the failure overlay.
  controller.vm.error = null;
  if (isEmbedded.value) {
    emit('surface-cancel');
  }
}

defineExpose({
  submit: handleSubmit,
  validate,
  /** Runs dirty confirmation, then cancelHandler / route back / display reload. */
  cancel: handleCancel,
  refresh: handleRefresh,
  /** Silent beginDisplay — for post-action sheet sync (e.g. Activate). */
  reloadDisplay,
  /** BusyHost surface Retry — reloads display after optional dirty discard. */
  retrySurface: onFormSurfaceRetry,
  /** BusyHost soft-fail Cancel — clears error; embedded also emits surface-cancel. */
  cancelSurface: onFormSurfaceCancel,
  reset: handleReset,
  copy: handleCopy,
  getFormData: () => toRaw(exposedFormData.value) as any,
  /** Whether the draft differs from the create/edit baseline. */
  isDirty: isFormDirty,
  /** Effective record id (prop, else route peek). */
  getRecordId: () => effectiveRecordId.value,
  getViewMode: () => viewMode.value,
  isLoading: () => loading.value,
});
</script>

