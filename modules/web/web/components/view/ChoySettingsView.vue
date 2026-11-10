<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer class="gap-5" data-anchor="choy.settings-view" data-testid="choy-settings-view">
    <template v-if="showHeader" #header>
      <div data-anchor="choy.settings.view-chrome">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template #system-actions>
            <ChoyButton
              type="button"
              size="default"
              variant="default"
              data-testid="settings-save"
              :disabled="controlsLocked || !dirty"
              @click="emit('save')"
            >
              <Check class="size-4" aria-hidden="true" />
              {{ resolvedSaveLabel }}
            </ChoyButton>
            <ChoyButton
              type="button"
              size="default"
              variant="outline"
              data-testid="settings-discard"
              :disabled="controlsLocked || !dirty"
              @click="onDiscardClick"
            >
              <X class="size-4" aria-hidden="true" />
              {{ discardLabel }}
            </ChoyButton>
            <h1
              v-if="chromeTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              class="choy-settings-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]"
            >
              {{ chromeTitle }}
            </h1>
          </template>
          <!-- Keep #center registered so the thirds grid does not collapse. -->
          <template #center>
            <div class="min-w-0" />
          </template>
          <template v-if="viewSwitcherOptions.length > 1" #header-right>
            <ChoyViewSwitcher
              :active="activeView"
              :options="viewSwitcherOptions"
              :aria-label="_t('View switcher')"
              @select="onViewSelect"
            />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <ChoyViewBusyHost
      :busy="loading"
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto"
      data-anchor="choy.settings.sheet"
    >
      <div
        class="choy-settings-view__sheet flex min-h-0 min-w-0 flex-1 flex-col gap-8"
        :inert="loading || undefined"
      >
        <slot />
      </div>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, provide, reactive, watch } from 'vue';
import { useRouter } from 'vue-router';
import { Check, X } from 'lucide-vue-next';
import { createTranslate } from '@/web/web/i18n';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { useHostPageTitleInView } from '@/web/web/composables/usePageContext';
import { getDraftField, setDraftField } from '@/web/web/composables/listRowEdit';
import { confirmChoyAction, resolveConfirmChoy } from '@/web/web/composables/confirmChoyAction';
import { shouldSkipFormDirtyLeavePrompt } from '@/web/web/composables/formDirtyLeaveGuard';
import type { LocalFormStore } from '@/web/web/stores/localFormStore';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyViewActionBar from './ChoyViewActionBar.vue';
import ChoyViewBusyHost from './ChoyViewBusyHost.vue';
import ChoyViewContainer from './ChoyViewContainer.vue';
import ChoyViewSwitcher, { type ChoyViewSwitcherOption } from './ChoyViewSwitcher.vue';

defineOptions({ name: 'ChoySettingsView' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ChoySettingsView' });

const props = withDefaults(
  defineProps<{
    /** Reactive draft object; Choy*Field read/write via form-root. */
    draft?: Record<string, unknown>;
    /**
     * Optional local form store: when set without `draft`, form-root delegates
     * getField/setField to the store (curated settings keys as field names).
     */
    localStore?: LocalFormStore | null;
    showHeader?: boolean;
    /**
     * Optional title override for mounts without ChoyPage. Prefer Page `:title`
     * + useHostPageTitleInView so the page band does not duplicate chrome.
     */
    pageTitle?: string;
    loading?: boolean;
    dirty?: boolean;
    saveLabel?: string;
    discardLabel?: string;
    activeView?: string;
    viewSwitcherOptions?: ChoyViewSwitcherOption[];
  }>(),
  {
    showHeader: true,
    loading: false,
    dirty: false,
    activeView: 'settings',
    viewSwitcherOptions: () => [],
  },
);

const emit = defineEmits<{
  save: [];
  discard: [];
  'view-select': [value: string];
}>();

const { pageTitle: hostedTitle, pageTitleId } = useHostPageTitleInView(() => props.showHeader);
const chromeTitle = computed(() => {
  const override = String(props.pageTitle || '').trim();
  if (override) return override;
  return String(hostedTitle.value || '').trim();
});

const resolvedSaveLabel = computed(() => {
  if (props.saveLabel) return props.saveLabel;
  return props.loading ? _t('Saving...') : _t('Save');
});
const discardLabel = computed(() => props.discardLabel || _t('Discard'));
const controlsLocked = computed(() => props.loading === true);
const viewSwitcherOptions = computed(() => {
  const opts = props.viewSwitcherOptions || [];
  if (!controlsLocked.value) return opts;
  return opts.map((opt) => ({ ...opt, disabled: true }));
});

useProvideViewScope({ kind: 'Settings', mode: 'edit' });

/** Mutable bag used when the caller passes a plain draft object. */
const draftBag = reactive<Record<string, unknown>>({});

watch(
  () => props.draft,
  (next) => {
    if (!next || typeof next !== 'object') return;
    for (const key of Object.keys(draftBag)) {
      if (!(key in next)) delete draftBag[key];
    }
    Object.assign(draftBag, next);
  },
  { immediate: true, deep: true },
);

const formRoot = {
  get mode() {
    return 'edit' as const;
  },
  get draft() {
    if (props.localStore) return props.localStore.getValues();
    return draftBag;
  },
  getField(path: string) {
    if (props.localStore) return props.localStore.getField(path);
    return getDraftField(draftBag, path);
  },
  setField(path: string, value: unknown) {
    if (props.localStore) {
      props.localStore.setField(path, value);
      return;
    }
    setDraftField(draftBag, path, value);
    if (props.draft && typeof props.draft === 'object') {
      setDraftField(props.draft, path, value);
    }
  },
  onChange(_cb: (nextDraft: unknown) => void) {},
};

provide('form-root', formRoot);

let leaveConfirmPending = false;
let leaving = false;

async function confirmDiscardIfDirty(): Promise<boolean> {
  if (leaving || !props.dirty) return true;
  if (leaveConfirmPending) return false;
  leaveConfirmPending = true;
  try {
    await confirmChoyAction(_t('You have unsaved changes. Discard them?'), _t('Unsaved changes'), {
      confirmText: _t('Discard'),
      cancelText: _t('Cancel'),
      destructive: true,
    });
    return true;
  } catch {
    return false;
  } finally {
    leaveConfirmPending = false;
  }
}

function dismissLeaveConfirmIfPending() {
  if (!leaveConfirmPending) return;
  resolveConfirmChoy('dismiss');
  leaveConfirmPending = false;
}

const router = useRouter();
const removeLeaveGuard = router?.beforeEach?.((to, from) => {
  if (leaving) return true;
  // Auth exits (login/logout/register) must proceed even while Save/load runs.
  if (shouldSkipFormDirtyLeavePrompt(to, from)) {
    dismissLeaveConfirmIfPending();
    return true;
  }
  // Do not leave or switch views while Save/load is in flight.
  if (props.loading) return false;
  if (!props.dirty) return true;
  return confirmDiscardIfDirty();
});

onBeforeUnmount(() => {
  removeLeaveGuard?.();
  dismissLeaveConfirmIfPending();
});

async function onDiscardClick() {
  if (controlsLocked.value || !props.dirty) return;
  if (!(await confirmDiscardIfDirty())) return;
  emit('discard');
}

async function onViewSelect(value: string) {
  if (controlsLocked.value) return;
  if (!(await confirmDiscardIfDirty())) return;
  leaving = true;
  try {
    emit('view-select', value);
    // Keep leave-guard skipped until the sibling route push commits (or a
    // short fallback if afterEach never runs, e.g. stub / no-op navigation).
    await new Promise<void>((resolve) => {
      let settled = false;
      let removeAfter: (() => void) | undefined;
      const finish = () => {
        if (settled) return;
        settled = true;
        removeAfter?.();
        resolve();
      };
      removeAfter = router?.afterEach?.(() => finish());
      setTimeout(finish, 0);
    });
  } finally {
    leaving = false;
  }
}
</script>

<style scoped>
/*
 * Settings rows often carry longer CJK labels than dense Form columns.
 * Raise the horizontal label track only under this sheet; global Form tokens stay.
 * basis must clear ~max inside max-w-md hosts (20% of 28rem still under 6.5rem).
 */
.choy-settings-view__sheet {
  --choy-field-label-max: 12rem;
  --choy-field-label-basis: 55%;
}
</style>
