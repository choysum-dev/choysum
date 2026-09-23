<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
/**
 * Form view chrome skeleton. Slots and action-bar regions only; store,
 * validation, and record CRUD land in PR5.
 */
withDefaults(
  defineProps<{
    title?: string;
    showHeader?: boolean;
    showActions?: boolean;
    showMessages?: boolean;
    loading?: boolean;
    embedded?: boolean;
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
</script>

<template>
  <div
    data-anchor="choy.form-view"
    :class="[
      'choy-form-view relative rounded-lg bg-background text-foreground',
      embedded ? 'border-0 shadow-none' : 'border border-border',
    ]"
    :aria-busy="loading"
  >
    <div
      v-if="showHeader && (title || $slots['system-actions'] || $slots['user-actions'] || $slots['header-right'] || $slots.statusbar)"
      class="choy-form-view__header flex flex-wrap items-center gap-3 border-b border-border px-4 py-3"
      :aria-hidden="loading || undefined"
      :inert="loading || undefined"
    >
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

    <div v-if="showMessages && $slots.messages" class="choy-form-view__messages px-4 pt-3">
      <slot name="messages" />
    </div>

    <div
      class="choy-form-view__body p-4"
      :aria-hidden="loading || undefined"
      :inert="loading || undefined"
    >
      <slot />
    </div>

    <div
      v-if="loading"
      class="choy-form-view__loading absolute inset-0 z-10 flex items-center justify-center bg-background/60"
      role="status"
    >
      <span class="text-sm text-foreground/70">Loading…</span>
    </div>
  </div>
</template>
