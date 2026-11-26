<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!--
  Mid-thickness view chrome host (pair with ChoyFieldBase naming; not a fat behavior base).

  New view type checklist:
  1. registerViewKind({ id, fieldPresentation, defaultMode })
  2. page meta.viewKind / viewPaths (+ optional ChoyViewSwitcher)
  3. wrap with ChoyViewBase; fill #system-actions / #secondary-actions / #center / default body
  4. useProvideViewScope({ kind })
  5. wire busy / error / hasFrame / hasSettled / @retry / @cancel
  6. collection: Search first-frame + #search or embedded Search
  7. export from kit + chrome/mount unit test

  Left chrome contract (List / Kanban / Form / Chart):
  [identity: max-width title menu] [主 → 次 → 领域] | [找] | [看]
  Title is the overflow/IO menu trigger (click); no standalone ⋯.
-->
<template>
  <ChoyViewContainer :class="containerClass" :sheet-surface="sheetSurface" v-bind="attrs">
    <!--
      Action Bar + optional header-extra (Form statusbar / Chart controls) share
      view-header; vertical rhythm is view-header gap (not padding on extra).
      Paper shell / in-paper density: sheetSurface + sheetPad (see choy-view-sheet.md).
    -->
    <template v-if="showChromeHeader()" #header>
      <div v-if="showHeaderRegion()" :data-anchor="chromeAnchor || undefined">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="systemActionsLabel"
          :user-actions-label="userActionsLabel"
        >
          <template v-if="pageTitle || showTitleOverflow" #identity>
            <div
              v-if="showTitleOverflow"
              data-slot="view-action-bar-overflow"
              class="flex min-w-0 max-w-full items-center"
            >
              <ChoyPageTitleActions
                :has-io-menu="!!pageTitleActionsIo"
                :action-import="pageTitleActionsIo?.actionImport ?? false"
                :action-export="pageTitleActionsIo?.actionExport ?? false"
                :action-import-upload-hint="pageTitleActionsIo?.actionImportUploadHint"
                :action-import-column-mapping="pageTitleActionsIo?.actionImportColumnMapping"
                :action-list-ref="(pageTitleActionsIo?.actionListRef as any)"
                :action-company-id="pageTitleActionsIo?.actionCompanyId"
                :store="store"
                :menu-title="pageTitle || _t('Actions')"
                :menu-title-id="pageTitleId"
                :menu-title-class="resolvedPageTitleClass"
              >
                <template v-if="slots['overflow-actions']" #overflow>
                  <slot name="overflow-actions" />
                </template>
              </ChoyPageTitleActions>
            </div>
            <h1
              v-else-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              :title="pageTitle"
              :class="resolvedPageTitleClass"
            >
              {{ pageTitle }}
            </h1>
          </template>

          <template v-if="showSystemTray()" #system-actions>
            <slot name="system-actions" />
            <slot name="primary" />
          </template>

          <!--
            Leading secondary tray (ActionBar #user-actions slot name is legacy).
            Host #contextual-actions go to #center (List/Kanban/Chart) or Form
            statusbar — do not park them in this leading tray.
          -->
          <template v-if="showLeadingSecondary()" #user-actions>
            <slot name="secondary-actions" />
          </template>

          <!-- ActionBar thirds pads an empty center when this slot is omitted. -->
          <template v-if="slots.center" #center>
            <slot name="center" />
          </template>

          <template v-if="slots['header-right']" #header-right>
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>

      <div
        v-if="slots['header-extra']"
        data-slot="view-header-extra"
        class="min-w-0 shrink-0"
      >
        <slot name="header-extra" />
      </div>
    </template>

    <ChoyViewBusyHost
      v-bind="resolvedBusyHostAttrs"
      :busy="busy"
      :mode="mode"
      :class="resolvedBusyHostClass"
      :busy-label="busyLabel"
      :error="error"
      :has-frame="hasFrame"
      :has-settled="hasSettled"
      :error-fallback-title="errorFallbackTitle"
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

/** In-paper density on view-body-host (see choy-view-sheet.md). */
export type ChoyViewSheetPad = 'form' | 'flush' | 'none';

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
    /** When true, mount ChoyPageTitleActions (IO ⋮ / overflow) at the actions cluster end. */
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
    /**
     * Floating paper on view-body. Off for embedded / dialog hosts.
     * Default true (page views).
     */
    sheetSurface?: boolean;
    /**
     * In-paper density on view-body-host.
     * form = four-side pad; flush|none = no pad (tables / boards / embedded).
     */
    sheetPad?: ChoyViewSheetPad;
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
    sheetSurface: true,
    sheetPad: 'flush',
  },
);

const emit = defineEmits<{
  retry: [];
  cancel: [];
}>();

const slots = useSlots();
const attrs = useAttrs();

/**
 * Title track: text-base, max-width, single-line ellipsis so long labels never
 * grow the action bar past adjacent control height.
 */
const defaultPageTitleClass =
  'min-w-0 max-w-[10rem] truncate text-base font-semibold leading-none text-foreground sm:max-w-[14rem]';

/** Default title chrome + optional BEM / extra classes from the product view. */
const resolvedPageTitleClass = computed(() => cn(defaultPageTitleClass, props.pageTitleClass));

const systemActionsLabel = computed(() => props.systemActionsLabel || _t('System actions'));
const userActionsLabel = computed(() => props.userActionsLabel || _t('User actions'));

const containerClass = computed(() => cn('gap-5', props.class));

const sheetSurface = computed(() => props.sheetSurface !== false);
const sheetPad = computed<ChoyViewSheetPad>(() => props.sheetPad || 'flush');

/** form preset only — flush/none leave host padding to the product view. */
const sheetPadClass = computed(() =>
  sheetPad.value === 'form' ? 'p-[var(--choy-view-sheet-pad)]' : undefined,
);

const resolvedBusyHostClass = computed(() => cn(sheetPadClass.value, props.busyHostClass));

const resolvedBusyHostAttrs = computed(() => ({
  ...(props.busyHostAttrs || {}),
  'data-sheet-pad': sheetPad.value,
}));

/**
 * Slot presence is read during render (not via computed) so Vue 3.5 production
 * builds re-evaluate when a parent adds/removes a conditional named slot.
 */
function showSystemTray(): boolean {
  return props.showActions || !!slots['system-actions'] || !!slots.primary;
}

/** Leading secondary tray (#secondary-actions only), separated from platform. */
function showLeadingSecondary(): boolean {
  return !!slots['secondary-actions'];
}

function showHeaderRegion(): boolean {
  if (!props.showHeader) return false;
  return (
    !!props.pageTitle ||
    !!props.showTitleOverflow ||
    showSystemTray() ||
    showLeadingSecondary() ||
    !!slots.center ||
    !!slots['header-right']
  );
}

/** Action Bar and/or header-extra (statusbar / Chart controls) as one chrome block. */
function showChromeHeader(): boolean {
  return showHeaderRegion() || !!slots['header-extra'];
}
</script>
