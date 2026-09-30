<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    ref="rootEl"
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
          description ||
          showBreadcrumb ||
          $slots.breadcrumb ||
          hasIoMenu ||
          $slots['title-actions']
        "
        class="choy-page__header mb-3 flex flex-col gap-1 border-b border-border pb-2"
        :inert="loading || undefined"
      >
        <template v-if="$slots.header">
          <div
            v-if="showBreadcrumb || $slots.breadcrumb"
            class="choy-page__breadcrumb text-sm text-foreground/70"
          >
            <slot name="breadcrumb" />
          </div>
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
              :action-import-upload-hint="actionImportUploadHint"
              :action-import-column-mapping="actionImportColumnMapping"
              :action-list-ref="actionListRef"
              :action-company-id="actionCompanyId"
              :store="store"
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
            v-if="title || description || hasIoMenu || $slots['title-actions']"
            class="choy-page__title-row flex items-center justify-between gap-3"
            :style="{ minHeight: 'var(--choy-control-height)' }"
          >
            <div class="min-w-0 flex-1">
              <h1
                v-if="title"
                :id="pageTitleId"
                class="choy-page__title m-0 min-w-0 truncate text-base font-semibold tracking-tight text-foreground"
              >
                {{ title }}
              </h1>
              <p
                v-if="description"
                class="choy-page__description m-0 text-xs text-muted-foreground"
              >
                {{ description }}
              </p>
            </div>
            <ChoyPageTitleActions
              :has-io-menu="hasIoMenu"
              :action-import="actionImport"
              :action-export="actionExport"
              :action-import-upload-hint="actionImportUploadHint"
              :action-import-column-mapping="actionImportColumnMapping"
              :action-list-ref="actionListRef"
              :action-company-id="actionCompanyId"
              :store="store"
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
        :inert="loading || undefined"
      >
        <slot name="toolbar" />
      </div>

      <div
        class="choy-page__body"
        :class="{ 'pb-4': !!$slots.footer }"
        :inert="loading || undefined"
      >
        <slot />
      </div>

      <div
        v-if="$slots.footer"
        class="choy-page__footer mt-4 border-t border-border pt-4"
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
      <ChoySkeleton class="h-control w-32" />
    </div>
    <div
      role="status"
      class="absolute h-px w-px overflow-hidden whitespace-nowrap opacity-0"
    >
      {{ loading ? loadingLabel : '' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import {
  providePageContext,
  useOptionalPageStore
} from '../../composables/usePageContext';
import type { WebModelStore } from '../../stores/modelStore';
import { blurFocusedDescendant } from './choy_page_loading_focus';
import ChoyPageTitleActions from './ChoyPageTitleActions.vue';
import ChoySkeleton from './ChoySkeleton.vue';
import type { PageIoMenuListRef } from './ChoyPageIoMenu.vue';
import { createTranslate } from '../../i18n';

const { _t } = createTranslate('web', { scope: 'web/components/layout/Page' });

type PageWidth = '' | 'narrow' | 'medium' | 'wide' | 'full';

/**
 * Dense Admin Page chrome: title (+ optional description) + page actions + body.
 * View Save/New stay in View tray, not here. Loading uses inert (not aria-hidden).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    title?: string;
    description?: string;
    showBreadcrumb?: boolean;
    padding?: boolean;
    width?: PageWidth;
    loading?: boolean;
    actionImport?: boolean;
    actionExport?: boolean;
    /** Optional CSV upload hint forwarded to PageIoMenu. */
    actionImportUploadHint?: string;
    /** Optional import column mapping forwarded to PageIoMenu. */
    actionImportColumnMapping?: Record<string, string>;
    /** List/kanban view ref for export scope and import refresh. */
    actionListRef?: PageIoMenuListRef | null;
    /** Optional company override for import/export panels. */
    actionCompanyId?: string;
    /** Optional default screen store for ChoyFormView / ChoyListView / fields. */
    store?: WebModelStore<any>;
  }>(),
  {
    title: '',
    description: '',
    showBreadcrumb: false,
    padding: true,
    width: '',
    loading: false,
    actionImport: false,
    actionExport: false,
  },
);

const loadingLabel = computed(() => _t('Loading...'));

const parentPageStore = useOptionalPageStore();
providePageContext({ store: () => props.store ?? parentPageStore.value });

const pageTitleId = useId();
const rootEl = ref<HTMLElement | null>(null);

// Blur focused descendants before loading applies inert (e.g. login submit
// still focused while the page masks). Prefer inert over aria-hidden so a
// retained focus cannot trip Chromium's aria-hidden focus warning.
watch(
  () => props.loading,
  loading => {
    if (loading) blurFocusedDescendant(rootEl.value);
  },
  // flush:pre blurs before inert is applied; immediate covers an initial
  // loading=true before rootEl is bound (paired with onMounted below).
  { flush: 'pre', immediate: true },
);
// Cover mount-with-loading-true after rootEl is bound (immediate may run too early).
onMounted(() => {
  if (props.loading) blurFocusedDescendant(rootEl.value);
});

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
</script>
