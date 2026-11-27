<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- App shell: full-height sidebar + Inset header/canvas + shared footer. -->
  <SidebarProvider
    class="choy-shell flex h-full min-h-0 w-full flex-col overflow-hidden"
    data-testid="choy-shell"
    data-shell-mode="app"
  >
    <div
      class="choy-shell__body relative flex min-h-0 flex-1 overflow-hidden"
      :inert="navigationHandoffBusy || undefined"
    >
      <template v-if="showSidebar">
        <ChoySidebarBridge />
        <Sidebar collapsible="icon" :side="sidebarSide">
          <!-- Height/padding locked in product CSS so kit p-2 cannot shrink the band. -->
          <SidebarHeader class="choy-shell__sidebar-header">
            <div
              data-anchor="choy.shell.sidebar-header"
              class="flex w-full min-w-0 items-center"
              data-testid="choy-shell-sidebar-header"
            >
              <slot name="sidebar-header" />
            </div>
          </SidebarHeader>
          <div
            class="flex min-h-0 w-full flex-1 flex-row overflow-hidden"
            data-testid="choy-shell-nav-split"
          >
            <ChoyAppRail />
            <div
              class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
              data-testid="choy-shell-nav-pane"
            >
              <div
                class="flex min-h-0 flex-1 flex-col overflow-auto"
                data-testid="choy-shell-nav-pane-scroll"
              >
                <slot name="aside">
                  <ChoySidebarNav />
                </slot>
              </div>
            </div>
          </div>
          <SidebarFooter class="choy-shell__sidebar-footer">
            <div
              data-anchor="choy.shell.sidebar-footer"
              class="flex w-full min-w-0 items-center"
              data-testid="choy-shell-sidebar-footer"
            >
              <slot name="sidebar-footer" />
            </div>
          </SidebarFooter>
          <SidebarRail />
        </Sidebar>
      </template>
      <!--
        Inset main: muted canvas flush to shell body (no margin/radius/shadow).
        Header, canvas, and footer are in-flow siblings; gap sets vertical rhythm.
        data-testid=choy-shell-scroll marks the ancestor adaptive height measures.
      -->
      <SidebarInset
        class="choy-shell__inset flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
        data-testid="choy-shell-scroll"
      >
        <ChoyAppHeader
          v-if="effectiveShowHeader"
          :show-sidebar-chrome="showSidebar"
          :menu-trigger-label="menuTriggerLabel"
        >
          <template #header-actions>
            <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
              <slot name="header-actions" />
            </div>
          </template>
        </ChoyAppHeader>
        <div
          class="choy-shell__canvas flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto"
          data-testid="choy-shell-canvas"
        >
          <div
            class="choy-shell__content flex min-h-0 w-full min-w-0 flex-1 flex-col"
            data-anchor="choy.shell.content"
            data-testid="choy-shell-content"
          >
            <slot>
              <ChoyViewLoadError v-if="hasViewLoadError" />
              <router-view v-slot="{ Component, route: viewRoute }">
                <!-- Remount KeepAlive when company or leaf menu changes. -->
                <div
                  v-if="!hasViewLoadError"
                  :key="keepAliveScopeKey"
                  class="flex min-h-0 min-w-0 w-full flex-1 flex-col"
                >
                  <KeepAlive :max="5">
                    <component
                      :is="Component"
                      v-if="Component && routeShouldKeepAlive(viewRoute.meta)"
                      class="min-h-0 flex-1"
                      :key="viewRoute.path"
                    />
                  </KeepAlive>
                  <component
                    :is="Component"
                    v-if="Component && !routeShouldKeepAlive(viewRoute.meta)"
                    class="min-h-0 flex-1"
                    :key="viewRoute.fullPath"
                  />
                </div>
              </router-view>
            </slot>
          </div>
        </div>
        <footer
          v-if="showFooter"
          class="choy-shell__footer flex shrink-0 items-center justify-center"
          :style="{ height: 'var(--choy-layout-footer-height)', boxSizing: 'border-box' }"
          data-testid="choy-shell-footer"
        >
          <ChoyAppFooter />
          <slot name="footer" />
        </footer>
      </SidebarInset>
    </div>
    <!-- Blocks clicks until first surface owns the nav ticket (Setup/Login → app land). -->
    <ChoyNavigationHandoff />
  </SidebarProvider>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useNavigationHandoffBusy } from '@/core/web/client/loadingRegistry'
import { useI18n } from 'vue-i18n'
import { routeShouldKeepAlive } from '../../composables/useChoyViewRouteSwitch'
import { useKeepAliveScopeKey } from '../../composables/useKeepAliveScopeKey'
import { viewLoadError as viewLoadErrorRef } from '../../router/viewLoadError'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyAppHeader from './ChoyAppHeader.vue'
import ChoyAppRail from './ChoyAppRail.vue'
import ChoyNavigationHandoff from './ChoyNavigationHandoff.vue'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import ChoyViewLoadError from './ChoyViewLoadError.vue'
import {
  Sidebar,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { useI18nStore } from '../../stores/i18nStore'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/** Local binding so the template unwraps the imported computed. */
const hasViewLoadError = computed(() => viewLoadErrorRef.value != null)

/**
 * App shell: full-height sidebar (company header / L1+L2 / user footer)
 * plus Inset chrome (trigger, breadcrumb, search, notifications) and canvas.
 */
const navigationHandoffBusy = useNavigationHandoffBusy()

const props = withDefaults(
  defineProps<{
    showHeader?: boolean
    showSidebar?: boolean
    showFooter?: boolean
  }>(),
  {
    showHeader: true,
    showSidebar: true,
    showFooter: true,
  },
)

let layoutStore: ReturnType<typeof useLayoutStore> | null = null
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key

try {
  layoutStore = useLayoutStore()
} catch {
  layoutStore = null
}

try {
  const i18n = useI18n({ useScope: 'global' })
  tLayout = (key, values) => String(i18n.t(key, values as any))
} catch {
  tLayout = (key) => key
}

const effectiveShowHeader = computed(() => props.showHeader)

/** Remount KeepAlive when company scope or leaf menu (activeMenu) changes. */
const keepAliveScopeKey = useKeepAliveScopeKey()

const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: layoutStore?.sidebarMode === 'collapsed',
    t: (key) => tLayout(key),
  }),
)

/** Vendor Sidebar uses physical left|right; pin to inline-start via textDirection. */
const sidebarSide = computed<'left' | 'right'>(() => {
  try {
    return useI18nStore().activeLanguage?.textDirection === 'rtl' ? 'right' : 'left'
  } catch {
    return 'left'
  }
})
</script>

<style>
/* Fill the app host; override kit SidebarProvider min-h-svh so inset chrome can size the column. */
.choy-shell[data-shell-mode='app'] {
  --sidebar-width: calc(var(--sidebar-width-icon) + 13.5rem) !important;
  /* Chrome + L1 + header/footer; L2 is a lighter nested pane. Both neutral. */
  --choy-shell-chrome: oklch(0.965 0 0);
  --choy-shell-pane: oklch(0.985 0 0);
  /*
   * Workspace behind the inset; inset uses pane. Floating paper is view-body
   * (sheetSurface).
   */
  /* Same as sidebar header/footer chrome so inset gutters match the rail poles. */
  --choy-shell-workspace: var(--choy-shell-chrome);
  /* Shared with the sidebar rail panel; inset main itself is flush (no margin/radius). */
  --choy-shell-inset-radius: var(--radius-xl, 0.75rem);
  /* Same as sidebar nav-pane (lighter than chrome poles); view-body is paper. */
  --choy-shell-inset-bg: var(--choy-shell-pane);
  /*
   * Inset chrome = control + chrome-padding (header top pad only; bottom is 0 —
   * header↔canvas uses section-gap). Sidebar logo top matches menu-trigger.
   */
  /* max(2rem, control): compact control can be <2rem; marks stay size-8. */
  --choy-layout-chrome-height: calc(
    max(2rem, var(--choy-control-height)) + var(--choy-layout-chrome-padding)
  );
  --choy-layout-header-height: var(--choy-layout-chrome-height);
  /* Same band as [data-slot=sidebar-footer] (also chrome-height). */
  --choy-layout-footer-height: var(--choy-layout-chrome-height);
  --choy-layout-sidebar-header-pad-top: var(--choy-layout-chrome-padding);
  --choy-layout-sidebar-header-height: calc(
    max(2rem, var(--choy-control-height)) + var(--choy-layout-sidebar-header-pad-top)
  );
  /* Identity row hover/open on chrome — keep in sync with --choy-color-sidebar-accent. */
  --choy-shell-chrome-hover: var(--choy-color-sidebar-accent);
  min-height: 0 !important;
  height: 100%;
  max-height: 100%;
}

html.dark .choy-shell[data-shell-mode='app'],
.dark .choy-shell[data-shell-mode='app'] {
  --choy-shell-chrome: oklch(0.18 0 0);
  --choy-shell-pane: oklch(0.205 0 0);
  --choy-shell-workspace: var(--choy-shell-chrome);
  --choy-shell-inset-bg: var(--choy-shell-pane);
  --choy-shell-chrome-hover: var(--choy-color-sidebar-accent);
}

/*
 * Kit desktop sidebar uses position:fixed + h-svh and would paint over a shell-level
 * footer. Pin the panel to .choy-shell__body instead; powered-by lives in the inset.
 *
 * Collapse motion matches sidebar-07 (width / left / right over 200ms linear).
 * Product override — tailwind-go emits transition-[width] without `width` in
 * transition-property, so the kit utility classes alone do not animate.
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > div {
  transition-property: width;
  transition-duration: 200ms;
  transition-timing-function: linear;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed {
  position: absolute !important;
  /*
   * Flush to the shell body so company logo / user avatar measure equal air
   * from the rail's top-left and bottom-left corners. Inset <main> keeps its
   * flush inset (paper radius/margin live on view-body).
   */
  top: 0 !important;
  bottom: 0 !important;
  height: auto !important;
  box-sizing: border-box !important;
  background-color: var(--choy-shell-chrome) !important;
  /*
   * Kill kit content-edge hairlines on both sides. Desktop Sidebar uses
   * physical group-data-[side=left]:border-r / [side=right]:border-l; a single
   * border-inline-end:0 does not reliably override the opposite physical
   * longhand when side flips under rtl. Inset <main> gutters already separate
   * the rail from the canvas — a hairline in the chrome gap looks redundant.
   */
  border-left: 0 !important;
  border-right: 0 !important;
  box-shadow: none !important;
  border-radius: var(--choy-shell-inset-radius);
  /*
   * Keep overflow visible so SidebarRail's outside hit target is not clipped.
   * Rounding still applies via the inner sidebar surface background.
   */
  overflow: visible;
  transition-property: left, right, width;
  transition-duration: 200ms;
  transition-timing-function: linear;
}

/* Pin the absolute panel to the declared side (kit left-0/right-0 can lose to dir). */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='left'] > .fixed {
  left: 0 !important;
  right: auto !important;
}
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='right'] > .fixed {
  left: auto !important;
  right: 0 !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed > [data-sidebar='sidebar'] {
  position: relative;
  border-radius: inherit;
  /* Clip chrome content to the radius without eating the rail sibling hit box…
   * Rail is a child of this node, so clip only non-rail bands below. */
  overflow: visible;
}

/*
 * SidebarRail: product wants the outer edge (LTR → leftmost, RTL → rightmost),
 * not kit’s content-facing overhang. Keep the full w-4 strip inside the panel so
 * inset cannot steal hover/click; clear Tailwind v4 `translate` (not `transform`).
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='left'] [data-sidebar='rail'] {
  right: auto !important;
  left: 0 !important;
  translate: none !important;
  transform: none !important;
}
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='right'] [data-sidebar='rail'] {
  left: auto !important;
  right: 0 !important;
  translate: none !important;
  transform: none !important;
}
/* Hairline on the outer edge of the hit strip (kit centers via after:left-1/2). */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='left'] [data-sidebar='rail']::after {
  inset-inline: auto !important;
  inset-block: 0 !important;
  left: 0 !important;
  right: auto !important;
  translate: none !important;
  transform: none !important;
}
.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-side='right'] [data-sidebar='rail']::after {
  inset-inline: auto !important;
  inset-block: 0 !important;
  left: auto !important;
  right: 0 !important;
  translate: none !important;
  transform: none !important;
}

.choy-shell[data-shell-mode='app']
  [data-slot='sidebar']
  > .fixed
  > [data-sidebar='sidebar']
  > :not([data-sidebar='rail']) {
  /* Header / nav / footer stay inside the rounded pane. */
  max-width: 100%;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'] > .fixed::after {
  content: none !important;
}

/*
 * Collapsed = L1 only.
 * Hide the L2 pane and force the kit gap + fixed panel to icon width.
 * Tailwind group-data width utilities do not always win against the expanded
 * --sidebar-width override, so product CSS owns the collapsed width.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='choy-shell-nav-pane'] {
  display: none !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar'][data-collapsible='icon'] > div {
  width: var(--sidebar-width-icon) !important;
}

/*
 * Collapsed: hide labels / meta / company chevrons only. Mark size + left inset
 * stay identical to expanded so width animation does not jump the glyphs.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-label'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-meta'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-label'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-meta'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-chevrons'] {
  display: none !important;
}

/*
 * Chrome (L1 + spanning header/footer) = 0.965; nested L2 pane = 0.985.
 * Collapsed: whole panel is chrome (L2 hidden).
 * Column gap = section-gap so header/footer breathe against nav-split
 * (same 0.5rem rhythm as inset header↔canvas and nav-pane padding-block).
 */
.choy-shell[data-shell-mode='app'] [data-sidebar='sidebar'] {
  background-color: var(--choy-shell-chrome) !important;
  gap: var(--choy-layout-section-gap) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-app-rail'] {
  background-color: var(--choy-shell-chrome) !important;
}

/*
 * The pane is the rounded menu surface. Block pad = air to the scroller;
 * end margin reveals sidebar chrome so the pane reads as a nested card
 * against main (same pane fill). Radius stays md on this narrow column.
 */
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-nav-pane'] {
  background-color: var(--choy-shell-pane) !important;
  box-sizing: border-box;
  padding-block: var(--choy-layout-section-gap) !important;
  margin-inline-end: var(--choy-layout-section-gap) !important;
  border-radius: var(--radius-md, 0.5rem);
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-nav-pane-scroll'] {
  background-color: transparent !important;
}

/*
 * Shell body = chrome; inset main = pane (same as sidebar nav-pane), flush,
 * no radius/shadow — floating paper is view-body. Action Bar sits on pane.
 * Kill kit SidebarInset peer inset m-2 / rounded-xl / shadow-sm.
 */
.choy-shell[data-shell-mode='app'] .choy-shell__body {
  background-color: var(--choy-shell-workspace);
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar-inset'] {
  margin: 0 !important;
  border-radius: 0 !important;
  background-color: var(--choy-shell-inset-bg) !important;
  box-shadow: none !important;
  overflow: hidden !important;
  min-height: 0 !important;
  /* Header ↔ canvas ↔ footer. Inline gutters live on children (see below). */
  gap: var(--choy-layout-section-gap) !important;
  padding-inline: 0 !important;
}

/*
 * Header: chrome-padding on top + inline, no bottom (section-gap separates
 * canvas). Footer mirrors with bottom pad. Canvas has no inline pad — the
 * floating paper (view-body) owns horizontal gutters.
 */
.choy-shell[data-shell-mode='app'] .choy-app-header {
  padding: var(--choy-layout-chrome-padding) var(--choy-layout-chrome-padding) 0 !important;
  box-sizing: border-box;
}

.choy-shell[data-shell-mode='app'] .choy-shell__footer {
  padding: 0 var(--choy-layout-chrome-padding) var(--choy-layout-chrome-padding) !important;
  box-sizing: border-box;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-canvas'] {
  background-color: transparent !important;
  padding-inline: 0 !important;
  box-sizing: border-box;
}

/*
 * Sidebar header/footer: start chrome-padding; end margin matches nav-pane so
 * hover pills share the pane's trailing edge (chrome gutter shows past it).
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-footer'] {
  box-sizing: border-box;
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: flex-start !important;
  gap: 0 !important;
  padding-inline-start: var(--choy-layout-chrome-padding) !important;
  padding-inline-end: 0 !important;
  margin-inline-end: var(--choy-layout-section-gap) !important;
  border: 0 !important;
  background-color: var(--choy-shell-chrome) !important;
  background-image: none !important;
  box-shadow: none !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar-header'] {
  height: var(--choy-layout-sidebar-header-height) !important;
  min-height: var(--choy-layout-sidebar-header-height) !important;
  max-height: var(--choy-layout-sidebar-header-height) !important;
  padding-block-start: var(--choy-layout-sidebar-header-pad-top) !important;
  padding-block-end: 0 !important;
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar-footer'] {
  height: var(--choy-layout-chrome-height) !important;
  min-height: var(--choy-layout-chrome-height) !important;
  max-height: var(--choy-layout-chrome-height) !important;
  padding-block-start: 0 !important;
  padding-block-end: var(--choy-layout-chrome-padding) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-footer'] {
  height: 100%;
  min-height: 0;
  width: 100%;
}

/* Tip/popover hosts must stretch so size-lg h-12 does not leave empty band air. */
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'] > *,
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-footer'] > *,
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'] [data-slot='tooltip-trigger'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-footer'] [data-slot='tooltip-trigger'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'] [data-slot='popover-trigger'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'] [data-slot='popover-anchor'] {
  height: 100%;
  min-height: 0;
}

/* Footer identity row: trigger flexes; connectivity pill keeps a trailing slot. */
.choy-shell[data-shell-mode='app'] [data-testid='auth-sidebar-user-menu'] {
  height: 100%;
  min-height: 0;
}

.choy-shell[data-shell-mode='app'] [data-testid='auth-sidebar-user-menu'] > .auth-sidebar-user-menu__identity,
.choy-shell[data-shell-mode='app'] [data-testid='auth-sidebar-user-menu'] > .auth-sidebar-user-menu__identity > * {
  height: 100%;
  min-height: 0;
}

.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'] {
  justify-content: flex-start !important;
  align-items: center !important;
  width: 100% !important;
  height: 100% !important;
  min-height: 0 !important;
  min-width: 0 !important;
  padding-block: 0 !important;
  padding-inline: 0 !important;
  margin-inline: 0 !important;
  border-radius: var(--radius-md, 0.375rem);
  gap: 0.5rem !important;
  overflow: hidden !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'] {
  justify-content: flex-start !important;
  align-items: center !important;
  width: 100% !important;
  height: 100% !important;
  min-height: 0 !important;
  min-width: 0 !important;
  /* No inline pad: mark x stays at header inset in both expand and collapse. */
  padding-block: 0 !important;
  padding-inline: 0 !important;
  margin-inline: 0 !important;
  border-radius: var(--radius-md, 0.375rem);
  gap: 0.5rem !important;
  overflow: hidden !important;
}

/* Company + user identity share the same chrome fill hover/open. */
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'][data-state='open'],
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'][data-state='open'] {
  background-color: var(--choy-shell-chrome-hover) !important;
}

/*
 * Collapsed icon rail: keep marks start-aligned with the same chrome pad as
 * expanded. Centering a 2rem hit target while width still animates from the
 * wide panel slides the logo/avatar toward the middle, then back — L1 rail
 * tiles never do that because they stay pinned to the start edge.
 * With --sidebar-width-icon (3rem) + start pad + end margin (0.5rem each),
 * the remaining 2rem is exactly the size-8 mark, so final geometry stays centered.
 */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-trigger'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-trigger'] {
  width: 2rem !important;
  height: 2rem !important;
  min-width: 2rem !important;
  min-height: 2rem !important;
  padding: 0 !important;
  margin: 0 !important;
  justify-content: flex-start !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='company-switch-logo-wrap'],
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-logo'] {
  width: 2rem !important;
  height: 2rem !important;
  min-width: 2rem !important;
  min-height: 2rem !important;
}

.choy-shell[data-shell-mode='app']
  [data-testid='auth-user-menu-trigger']
  [data-slot='avatar'],
.choy-shell[data-shell-mode='app']
  [data-testid='auth-user-menu-heading']
  [data-slot='avatar'] {
  width: 2rem !important;
  height: 2rem !important;
  min-width: 2rem !important;
  min-height: 2rem !important;
  border-radius: 9999px !important;
}
</style>
