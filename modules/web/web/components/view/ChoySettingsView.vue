<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    presentation="settings"
    :store="resolvedStore"
    :local-store="localStore || null"
    :draft="draft"
    :show-header="showHeader"
    :show-actions="showHeader"
    :show-messages="false"
    :resolve-record-id-from-route="false"
    :register-action-target="false"
    view-mode="edit"
    :dirty="dirty"
    :content-loading="contentLoading"
    :control-busy="loading"
    :page-title="pageTitle"
    :save-label="saveLabel"
    :discard-label="discardLabel"
    :active-view="activeView"
    :view-switcher-options="viewSwitcherOptions"
    :cancel-handler="onCancelHandler"
    @save="emit('save')"
    @view-select="emit('view-select', $event)"
  >
    <slot />
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { createLocalFormStore, type LocalFormStore } from '@/web/web/stores/localFormStore';
import ChoyFormView from './ChoyFormView.vue';
import type { ChoyViewSwitcherOption } from './ChoyViewSwitcher.vue';

defineOptions({ name: 'ChoySettingsView' });

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
    /**
     * True while saving (control). Prefer `loading` alone for save lock;
     * use `contentLoading` for first-load surface busy without implying Saving….
     */
    loading?: boolean;
    /** First-load / refresh of the settings sheet (surface skeleton/block). */
    contentLoading?: boolean;
    dirty?: boolean;
    saveLabel?: string;
    discardLabel?: string;
    activeView?: string;
    viewSwitcherOptions?: ChoyViewSwitcherOption[];
  }>(),
  {
    showHeader: true,
    loading: false,
    contentLoading: false,
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

/**
 * Form requires a store; draft-only mounts use a chrome placeholder while
 * form-root binds the draft bag inside presentation=settings.
 */
const chromeStore = createLocalFormStore({
  fields: [],
  storeId: 'choy.settings.chrome',
});

const resolvedStore = computed(() => props.localStore || chromeStore);

function onCancelHandler() {
  emit('discard');
}
</script>
