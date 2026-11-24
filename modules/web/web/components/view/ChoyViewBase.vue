<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Mid-thickness view chrome host (pair with ChoyFieldBase naming; not a fat behavior base).

  New view type checklist:
  1. registerViewKind({ id, fieldPresentation, defaultMode })
  2. page meta.viewKind / viewPaths (+ optional ChoyViewSwitcher)
  3. wrap with ChoyViewBase; fill #system-actions / #center / default body
  4. useProvideViewScope({ kind })
  5. wire busy / error / hasFrame / hasSettled / @retry / @cancel
  6. collection: Search first-frame + #search or embedded Search
  7. export from kit + chrome/mount unit test

  Left chrome contract (List / Kanban / Form):
  [primary / #system-actions] [page title][⋮] [#secondary-actions…]
-->
<template>
  <ChoyViewContainer :class="containerClass" v-bind="containerBind">
    <template v-if="showHeaderRegion" #header>
      <div :data-anchor="chromeAnchor || undefined">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="systemActionsLabel"
          :user-actions-label="userActionsLabel"
        >
          <template v-if="showSystemTray" #system-actions>
            <slot name="system-actions" />
            <slot name="primary" />

            <h1
              v-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              :class="pageTitleClass || defaultPageTitleClass"
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
              <template v-if="slots['overflow-actions']" #overflow>
                <slot name="overflow-actions" />
              </template>
            </ChoyPageTitleActions>

            <slot name="secondary-actions" />
          </template>

          <template v-if="slots['user-actions']" #user-actions>
            <slot name="user-actions" />
          </template>

          <template v-if="slots.center" #center>
            <slot name="center" />
          </template>

          <template v-if="slots['header-right']" #header-right>
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <div v-if="slots['header-extra']" data-slot="view-header-extra" class="min-w-0 shrink-0">
      <slot name="header-extra" />
    </div>

    <ChoyViewBusyHost
      :busy="busy"
      :mode="mode"
      :class="busyHostClass"
      :busy-label="busyLabel"
      :error="error"
      :has-frame="hasFrame"
      :has-settled="hasSettled"
      :error-fallback-title="errorFallbackTitle"
      v-bind="busyHostAttrs"
      @retry="emit('retry')"
      @cancel="emit('cancel')"
    >
      <slot />
    </ChoyViewBusyHost>

    <template v-if="slots.footer" #footer>
      <slot name="footer" />
    </template>
  </ChoyViewContainer>
</template>

<script setup lang="ts">
import { computed, useAttrs, useSlots } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import ChoyPageTitleActions from '@/web/web/components/layout/ChoyPageTitleActions.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import { createTranslate } from '@/web/web/i18n';
import type { ViewBusyMode } from './viewBusy';

defineOptions({ name: 'ChoyViewBase', inheritAttrs: false });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewBase' });

export type ChoyViewBaseTitleActionsIo = {
  actionImport?: boolean;
  actionExport?: boolean;
  actionImportUploadHint?: string;
  actionImportColumnMapping?: Record<string, string>;
  actionListRef?: unknown;
  actionCompanyId?: string;
};

const props = withDefaults(
  defineProps<{
    /** When false, header ActionBar is omitted. */
    showHeader?: boolean;
    /** Force system tray even without title/overflow (domain primary buttons). */
    showActions?: boolean;
    class?: ClassValue;
    /** data-anchor on the chrome wrapper under #header. */
    chromeAnchor?: string;
    systemActionsLabel?: string;
    userActionsLabel?: string;
    pageTitle?: string | null;
    pageTitleId?: string | null;
    pageTitleClass?: ClassValue;
    pageTitleActionsIo?: ChoyViewBaseTitleActionsIo | null;
    /** When true, mount ChoyPageTitleActions (IO ⋮ / overflow). */
    showTitleOverflow?: boolean;
    store?: { storeId?: string; fullModelName?: string; state?: { result?: { total?: number } } };
    busy?: boolean;
    mode?: ViewBusyMode;
    busyLabel?: string;
    busyHostClass?: ClassValue;
    /** Extra attrs on ChoyViewBusyHost (e.g. data-anchor / data-region). */
    busyHostAttrs?: Record<string, unknown>;
    error?: unknown;
    hasFrame?: boolean;
    hasSettled?: boolean;
    errorFallbackTitle?: string;
  }>(),
  {
    showHeader: true,
    showActions: false,
    showTitleOverflow: false,
    busy: false,
    mode: 'block',
    hasFrame: false,
    hasSettled: false,
    busyHostAttrs: () => ({}),
  },
);

const emit = defineEmits<{
  retry: [];
  cancel: [];
}>();

const slots = useSlots();
const attrs = useAttrs();

const defaultPageTitleClass =
  'min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]';

const systemActionsLabel = computed(() => props.systemActionsLabel || _t('System actions'));
const userActionsLabel = computed(() => props.userActionsLabel || _t('User actions'));

const containerClass = computed(() => cn('gap-5', props.class));

/** Pass-through attrs onto the container (data-anchor, data-testid, data-region, …). */
const containerBind = computed(() => {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key === 'class') continue;
    out[key] = value;
  }
  return out;
});

const showSystemTray = computed(() => {
  return (
    props.showActions ||
    !!props.pageTitle ||
    !!props.showTitleOverflow ||
    !!slots['system-actions'] ||
    !!slots.primary ||
    !!slots['secondary-actions']
  );
});

const showHeaderRegion = computed(() => {
  if (!props.showHeader) return false;
  return (
    showSystemTray.value ||
    !!slots['user-actions'] ||
    !!slots.center ||
    !!slots['header-right']
  );
});
</script>
