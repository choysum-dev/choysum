<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Store-bound engine: host OFormView (controller / onchange / field provides). -->
  <OFormView v-if="useStoreEngine" v-bind="(storeBind as any)" v-on="(storeListeners as any)">
    <template v-if="$slots.breadcrumb" #breadcrumb="slotData">
      <slot name="breadcrumb" v-bind="slotData || {}" />
    </template>
    <template v-if="$slots['system-actions']" #system-actions="slotData">
      <slot name="system-actions" v-bind="slotData || {}" />
    </template>
    <template v-if="$slots['user-actions']" #user-actions="slotData">
      <slot name="user-actions" v-bind="slotData || {}" />
    </template>
    <template v-if="$slots.statusbar" #statusbar="slotData">
      <slot name="statusbar" v-bind="slotData || {}" />
    </template>
    <template v-if="$slots['button-box']" #button-box="slotData">
      <slot name="button-box" v-bind="slotData || {}" />
    </template>
    <template v-if="$slots['header-right']" #header-right="slotData">
      <slot name="header-right" v-bind="slotData || {}" />
    </template>
    <slot />
  </OFormView>

  <!-- Chrome skeleton for Gallery / Dogfood (no WebModelStore). -->
  <div
    v-else
    data-anchor="choy.form-view"
    :class="[
      'choy-form-view relative rounded-lg bg-background text-foreground',
      embedded ? 'border-0 shadow-none' : 'border border-border',
    ]"
  >
    <div :aria-busy="loading || undefined">
      <div
        v-if="showHeader && (title || $slots.breadcrumb || $slots['system-actions'] || $slots['user-actions'] || $slots['header-right'] || $slots.statusbar)"
        class="choy-form-view__header flex flex-wrap items-center gap-3 border-b border-border px-4 py-3"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <div
          v-if="$slots.breadcrumb"
          class="choy-form-view__breadcrumb w-full basis-full"
        >
          <slot name="breadcrumb" />
        </div>
        <h2 v-if="title" class="choy-form-view__title text-base font-semibold">{{ title }}</h2>
        <div v-if="$slots.statusbar" class="choy-form-view__statusbar min-w-0 flex-1">
          <slot name="statusbar" />
        </div>
        <div class="choy-form-view__header-actions ml-auto flex flex-wrap items-center gap-2">
          <slot name="system-actions" />
          <slot name="user-actions" />
          <slot name="header-right" />
        </div>
      </div>

      <div
        v-if="showActions && $slots['button-box']"
        class="choy-form-view__button-box flex flex-wrap gap-2 border-b border-border px-4 py-2"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot name="button-box" />
      </div>

      <div
        v-if="showMessages && $slots.messages"
        class="choy-form-view__messages px-4 pt-3"
        aria-live="polite"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot name="messages" />
      </div>

      <div
        class="choy-form-view__body p-4"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot />
      </div>
    </div>

    <div
      v-if="loading"
      class="choy-form-view__loading absolute inset-0 z-10 flex items-center justify-center bg-background/60"
      aria-hidden="true"
    >
      <span class="text-sm text-foreground/70">Loading…</span>
    </div>
    <div
      role="status"
      class="absolute h-px w-px overflow-hidden whitespace-nowrap opacity-0"
    >
      {{ loading ? 'Loading…' : '' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useOptionalPageStore } from '@/web/web/composables/usePageContext';
import { hasChoyStoreEngine } from '@/web/web/composables/choyStoreMode';
import OFormView from './OFormView.vue';

defineOptions({ name: 'ChoyFormView', inheritAttrs: false });

/**
 * Form view: store-bound mode hosts OFormView; otherwise Gallery/Dogfood chrome.
 */
const props = withDefaults(
  defineProps<{
    title?: string;
    showHeader?: boolean;
    showActions?: boolean;
    showMessages?: boolean;
    loading?: boolean;
    embedded?: boolean;
    store?: WebModelStore<any>;
    recordId?: string;
    initialValues?: Record<string, unknown>;
    viewMode?: 'display' | 'edit' | 'create';
    createAction?: unknown;
    actionIds?: Record<string, string | undefined>;
    hasAction?: (actionId: string | undefined) => boolean;
    onchangeSessionId?: string;
    onchangeDebounceMs?: number;
    onchangeImmediateFirst?: boolean;
    submitHandler?: unknown;
    resolveRecordIdFromRoute?: boolean;
  }>(),
  {
    title: '',
    showHeader: true,
    showActions: true,
    showMessages: true,
    loading: false,
    embedded: false,
  },
);

const attrs = useAttrs();
const pageStore = useOptionalPageStore();
const useStoreEngine = computed(() => hasChoyStoreEngine(props.store, pageStore.value));

const storeBind = computed(() => ({
  ...attrs,
  store: props.store ?? pageStore.value,
  recordId: props.recordId,
  initialValues: props.initialValues,
  viewMode: props.viewMode,
  embedded: props.embedded,
  showHeader: props.showHeader,
  showActions: props.showActions,
  showMessages: props.showMessages,
  createAction: props.createAction,
  actionIds: props.actionIds,
  hasAction: props.hasAction,
  onchangeSessionId: props.onchangeSessionId,
  onchangeDebounceMs: props.onchangeDebounceMs,
  onchangeImmediateFirst: props.onchangeImmediateFirst,
  submitHandler: props.submitHandler,
  resolveRecordIdFromRoute: props.resolveRecordIdFromRoute,
}));

const storeListeners = computed(() => {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith('on') && typeof value === 'function') {
      out[key] = value;
    }
  }
  return out;
});
</script>
