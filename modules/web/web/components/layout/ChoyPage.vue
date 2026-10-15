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
        // Title hosted in view chrome: keep side/bottom gutter, drop top pad so
        // ActionBar sits flush under the shell header (no empty title-band gap).
        props.padding ? (titleHostedByView ? 'choy-inset-pad-no-top' : 'choy-inset-pad') : '',
        widthClass,
        props.fillHeight ? 'flex min-h-0 flex-1 flex-col' : '',
        props.class,
      )
    "
  >
    <div
      :class="props.fillHeight ? 'flex min-h-0 flex-1 flex-col' : undefined"
      :role="title ? 'region' : undefined"
      :aria-busy="loading || undefined"
      :aria-labelledby="title && !$slots.header ? pageTitleId : undefined"
      :aria-label="title && $slots.header ? title : undefined"
    >
      <div
        v-if="
          $slots.header ||
          showPageTitle ||
          description ||
          showBreadcrumb ||
          $slots.breadcrumb ||
          showPageTitleActions
        "
        data-slot="page-header"
        class="choy-page__header mb-2 flex flex-col gap-1.5"
        :inert="loading || undefined"
      >
        <template v-if="$slots.header">
          <div
            v-if="showBreadcrumb || $slots.breadcrumb"
            data-slot="page-breadcrumb"
            class="choy-page__breadcrumb text-sm text-muted-foreground"
          >
            <slot name="breadcrumb" />
          </div>
          <div
            v-if="showPageTitleActions"
            class="choy-page__title-row flex items-start justify-between gap-3"
          >
            <div class="choy-page__header-slot min-w-0 flex-1">
              <slot name="header" />
            </div>
            <ChoyPageTitleActions
              :has-io-menu="pageTitleActionsHasIo"
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
          <div
            v-if="showBreadcrumb || $slots.breadcrumb"
            data-slot="page-breadcrumb"
            class="choy-page__breadcrumb text-sm text-muted-foreground"
          >
            <slot name="breadcrumb" />
          </div>
          <div
            v-if="showPageTitle || description || showPageTitleActions"
            class="choy-page__title-row flex min-h-control items-start justify-between gap-3"
          >
            <div v-if="showPageTitle || description" class="min-w-0 flex-1 space-y-1">
              <h1
                v-if="showPageTitle"
                :id="pageTitleId"
                data-slot="page-title"
                class="choy-page__title min-w-0 truncate text-lg font-semibold tracking-tight text-foreground"
              >
                {{ title }}
              </h1>
              <p
                v-if="description"
                data-slot="page-description"
                class="choy-page__description text-sm text-muted-foreground"
              >
                {{ description }}
              </p>
            </div>
            <ChoyPageTitleActions
              v-if="showPageTitleActions"
              :has-io-menu="pageTitleActionsHasIo"
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
        data-slot="page-toolbar"
        class="choy-page__toolbar mb-4"
        role="toolbar"
        :inert="loading || undefined"
      >
        <slot name="toolbar" />
      </div>

      <div
        data-slot="page-body"
        class="choy-page__body"
        :class="{
          'pb-4': !!$slots.footer,
          'flex min-h-0 flex-1 flex-col overflow-hidden': props.fillHeight,
        }"
        :inert="loading || undefined"
      >
        <slot />
      </div>

      <div
        v-if="$slots.footer"
        data-slot="page-footer"
        class="choy-page__footer mt-4 border-t border-border pt-4"
        :inert="loading || undefined"
      >
        <slot name="footer" />
      </div>
    </div>

    <div
      v-if="loading"
      data-slot="page-loading"
      class="choy-page__loading-mask absolute inset-0 z-10 flex items-center justify-center bg-background/60"
      aria-hidden="true"
    >
      <ChoySkeleton class="h-control w-32" />
    </div>
    <div role="status" class="sr-only">
      {{ loading ? loadingLabel : '' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useId, useSlots, watch } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import {
  providePageContext,
  useOptionalPageStore,
  type PageTitleActionsIo,
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
 * List/Kanban may claim the title (+ IO) into view chrome so the page header stays empty.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    title?: string;
    description?: string;
    showBreadcrumb?: boolean;
    padding?: boolean;
    width?: PageWidth;
    /** Stretch to the shell main and let nested flex children (auth cards) center. */
    fillHeight?: boolean;
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
    // ERP workspace pages fill the shell canvas; opt out for rare document pages.
    fillHeight: true,
    loading: false,
    actionImport: false,
    actionExport: false,
  },
);

const slots = useSlots();
const loadingLabel = computed(() => _t('Loading...'));

const pageTitleId = useId();
const hasIoMenu = computed(() => props.actionImport || props.actionExport);
const titleActionsIo = computed<PageTitleActionsIo | null>(() => {
  if (!hasIoMenu.value) return null;
  return {
    actionImport: !!props.actionImport,
    actionExport: !!props.actionExport,
    actionImportUploadHint: props.actionImportUploadHint,
    actionImportColumnMapping: props.actionImportColumnMapping,
    actionListRef: props.actionListRef,
    actionCompanyId: props.actionCompanyId,
  };
});

const parentPageStore = useOptionalPageStore();
const pageCtx = providePageContext({
  store: () => props.store ?? parentPageStore.value,
  title: () => props.title,
  titleId: pageTitleId,
  titleActionsIo,
});

const titleHostedByView = computed(() => pageCtx.titleHostedByView.value);
const showPageTitle = computed(() => !!props.title && !titleHostedByView.value);
// When a view hosts the title, IO moves into that chrome; keep page #title-actions
// visible (without duplicating the IO menu) so custom page actions do not disappear.
const showPageTitleActions = computed(() => {
  if (titleHostedByView.value) return !!slots['title-actions'];
  return hasIoMenu.value || !!slots['title-actions'];
});
const pageTitleActionsHasIo = computed(() => hasIoMenu.value && !titleHostedByView.value);

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
