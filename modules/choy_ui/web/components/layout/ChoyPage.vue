<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, useId } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import ChoyPageTitleActions from './ChoyPageTitleActions.vue';

type PageWidth = '' | 'narrow' | 'medium' | 'wide' | 'full';

/**
 * Page chrome inside the layout main area (title, toolbar, body, loading).
 * Slot visibility is read from `$slots` at render time (slots are not reactive).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    title?: string;
    showBreadcrumb?: boolean;
    padding?: boolean;
    width?: PageWidth;
    loading?: boolean;
    actionImport?: boolean;
    actionExport?: boolean;
  }>(),
  {
    title: '',
    showBreadcrumb: false,
    padding: true,
    width: '',
    loading: false,
    actionImport: false,
    actionExport: false,
  },
);

const pageTitleId = useId();

const hasIoMenu = computed(() => props.actionImport || props.actionExport);

const widthClass = computed(() => {
  switch (props.width) {
    case 'narrow':
      return 'max-w-2xl';
    case 'medium':
      return 'max-w-4xl';
    case 'wide':
      return 'max-w-6xl';
    case 'full':
      return 'max-w-none';
    default:
      return '';
  }
});

const emit = defineEmits<{
  import: [];
  export: [];
}>();
</script>

<template>
  <div
    data-anchor="choy.page"
    data-print="page"
    :class="
      cn(
        'choy-page relative mx-auto w-full text-foreground',
        props.padding ? 'p-4 md:p-6' : '',
        widthClass,
        props.class,
      )
    "
  >
    <div
      :role="title ? 'region' : undefined"
      :aria-busy="loading || undefined"
      :aria-labelledby="title && !$slots.header ? pageTitleId : undefined"
      :aria-label="title && $slots.header ? title : undefined"
    >
      <div
        v-if="
          $slots.header ||
          title ||
          showBreadcrumb ||
          $slots.breadcrumb ||
          hasIoMenu ||
          $slots['title-actions']
        "
        class="choy-page__header mb-4 flex flex-col gap-2"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <template v-if="$slots.header">
          <div
            v-if="hasIoMenu || $slots['title-actions']"
            class="choy-page__title-row flex items-start justify-between gap-3"
          >
            <div class="choy-page__header-slot min-w-0 flex-1">
              <slot name="header" />
            </div>
            <ChoyPageTitleActions
              :has-io-menu="hasIoMenu"
              :action-import="actionImport"
              :action-export="actionExport"
              @import="emit('import')"
              @export="emit('export')"
            >
              <slot name="title-actions" />
            </ChoyPageTitleActions>
          </div>
          <slot v-else name="header" />
        </template>
        <template v-else>
          <div v-if="showBreadcrumb || $slots.breadcrumb" class="choy-page__breadcrumb text-sm text-foreground/70">
            <slot name="breadcrumb" />
          </div>
          <div
            v-if="title || hasIoMenu || $slots['title-actions']"
            class="choy-page__title-row flex items-start justify-between gap-3"
          >
            <h1 v-if="title" :id="pageTitleId" class="choy-page__title text-xl font-semibold tracking-tight">
              {{ title }}
            </h1>
            <ChoyPageTitleActions
              :has-io-menu="hasIoMenu"
              :action-import="actionImport"
              :action-export="actionExport"
              @import="emit('import')"
              @export="emit('export')"
            >
              <slot name="title-actions" />
            </ChoyPageTitleActions>
          </div>
        </template>
      </div>

      <div
        v-if="$slots.toolbar"
        class="choy-page__toolbar mb-4"
        role="toolbar"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot name="toolbar" />
      </div>

      <div
        class="choy-page__body"
        :class="{ 'pb-4': !!$slots.footer }"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot />
      </div>

      <div
        v-if="$slots.footer"
        class="choy-page__footer mt-4 border-t border-border pt-4"
        :aria-hidden="loading || undefined"
        :inert="loading || undefined"
      >
        <slot name="footer" />
      </div>
    </div>

    <div
      v-if="loading"
      class="choy-page__loading-mask absolute inset-0 z-10 flex items-center justify-center bg-background/60"
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
