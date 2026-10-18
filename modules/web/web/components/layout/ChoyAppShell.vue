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
    <div class="choy-shell__body relative flex min-h-0 flex-1 overflow-hidden">
      <template v-if="showSidebar">
        <ChoySidebarBridge />
        <Sidebar collapsible="icon" side="left">
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
        Inset main: margin + radius + white surface. Header, canvas, and footer
        are in-flow siblings; vertical rhythm is gap on this element.
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
              <router-view v-slot="{ Component, route: viewRoute }">
                <KeepAlive>
                  <component
                    :is="Component"
                    v-if="Component && viewRoute.meta?.keepAlive"
                    class="min-h-0 flex-1"
                    :key="viewRoute.path"
                  />
                </KeepAlive>
                <component
                  :is="Component"
                  v-if="Component && !viewRoute.meta?.keepAlive"
                  class="min-h-0 flex-1"
                  :key="viewRoute.fullPath"
                />
              </router-view>
            </slot>
          </div>
        </div>
        <footer
          v-if="showFooter"
          class="choy-shell__footer flex shrink-0 items-center justify-center"
          :style="{ height: 'var(--choy-layout-chrome-height)', boxSizing: 'border-box' }"
          data-testid="choy-shell-footer"
        >
          <ChoyAppFooter />
          <slot name="footer" />
        </footer>
      </SidebarInset>
    </div>
  </SidebarProvider>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyAppHeader from './ChoyAppHeader.vue'
import ChoyAppRail from './ChoyAppRail.vue'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import {
  Sidebar,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/**
 * App shell: full-height sidebar (company header / L1+L2 / user footer)
 * plus Inset chrome (trigger, breadcrumb, search, notifications) and canvas.
 */
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

const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: layoutStore?.sidebarMode === 'collapsed',
    t: (key) => tLayout(key),
  }),
)
</script>

<style>
/* Fill the app host; override kit SidebarProvider min-h-svh so inset chrome can size the column. */
.choy-shell[data-shell-mode='app'] {
  --sidebar-width: calc(var(--sidebar-width-icon) + 13.5rem) !important;
  /* Chrome + L1 + header/footer; L2 is a lighter nested pane. Both neutral. */
  --choy-shell-chrome: oklch(0.965 0 0);
  --choy-shell-pane: oklch(0.985 0 0);
  /*
   * Workspace canvas behind List/Kanban/Form sheets — deeper than sidebar chrome
   * so white view-body sheets and ActionBar-on-gray read clearly.
   */
  /* Same as sidebar header/footer chrome so inset gutters match the rail poles. */
  --choy-shell-workspace: var(--choy-shell-chrome);
  /* Inset main gutters + radius (dashboard-01-like; tune without touching kit). */
  --choy-shell-inset-margin-y: 0.5rem;
  --choy-shell-inset-margin-x: 0.5rem;
  --choy-shell-inset-radius: var(--radius-xl, 0.75rem);
  --choy-shell-inset-bg: oklch(1 0 0);
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
  --choy-shell-inset-bg: var(--choy-color-card);
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
   * Same block inset as <main> so sidebar header/footer bands share a horizontal
   * seam with the content header/footer (dashboard-01 inset rhythm).
   */
  top: var(--choy-shell-inset-margin-y) !important;
  bottom: var(--choy-shell-inset-margin-y) !important;
  height: auto !important;
  box-sizing: border-box !important;
  /*
   * No right edge: inset <main> gutters already separate the rail from the
   * white surface (a hairline would sit in the chrome gap and look redundant).
   */
  border-right: 0 !important;
  box-shadow: none !important;
  border-radius: var(--choy-shell-inset-radius);
  overflow: hidden;
  transition-property: left, right, width;
  transition-duration: 200ms;
  transition-timing-function: linear;
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
 * Collapsed: hide labels / meta / chevrons only. Mark size + left inset stay
 * identical to expanded so width animation does not jump the glyphs.
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
  [data-testid='company-switch-chevrons'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-chevrons'] {
  display: none !important;
}

/*
 * Chrome (L1 + spanning header/footer) = 0.965; nested L2 pane = 0.985.
 * Collapsed: whole panel is chrome (L2 hidden).
 */
.choy-shell[data-shell-mode='app'] [data-sidebar='sidebar'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-app-rail'] {
  background-color: var(--choy-shell-chrome) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-nav-pane'] {
  background-color: var(--choy-shell-pane) !important;
}

/*
 * Outer shell body is workspace gray; inset <main> is the white surface
 * (margin + radius). Inner scroll/canvas/footer inherit — no second bg.
 */
.choy-shell[data-shell-mode='app'] .choy-shell__body {
  background-color: var(--choy-shell-workspace);
}

.choy-shell[data-shell-mode='app'] [data-slot='sidebar-inset'] {
  margin-block: var(--choy-shell-inset-margin-y) !important;
  margin-inline-end: var(--choy-shell-inset-margin-x) !important;
  margin-inline-start: var(--choy-shell-inset-margin-x) !important;
  border-radius: var(--choy-shell-inset-radius) !important;
  background-color: var(--choy-shell-inset-bg) !important;
  /* Match dashboard-01 inset main (shadow-sm). */
  box-shadow: var(--shadow-sm) !important;
  overflow: hidden !important;
  min-height: 0 !important;
  /* Header ↔ canvas ↔ footer. Inline gutter stays the wider content pad. */
  gap: var(--choy-layout-section-gap) !important;
  padding-inline: var(--choy-layout-content-padding) !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-canvas'] {
  background-color: transparent !important;
}

/*
 * Sidebar header/footer bands match inset header/footer via layout tokens
 * so left/right poles share one horizontal seam. Identity triggers fill the
 * band; inline pad 0.5rem keeps mark x stable in the icon rail (size-8 slot).
 */
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-slot='sidebar-footer'] {
  box-sizing: border-box;
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: flex-start !important;
  gap: 0 !important;
  height: var(--choy-layout-chrome-height) !important;
  min-height: var(--choy-layout-chrome-height) !important;
  max-height: var(--choy-layout-chrome-height) !important;
  padding-block: 0.25rem !important;
  padding-inline: 0.5rem !important;
  border: 0 !important;
  background-color: var(--choy-shell-chrome) !important;
  background-image: none !important;
  box-shadow: none !important;
}

.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-header'],
.choy-shell[data-shell-mode='app'] [data-testid='choy-shell-sidebar-footer'] {
  height: 100%;
  min-height: 0;
}

.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'],
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

/* Chrome rows need an explicit hover fill; kit accent alone was ≈ chrome in light. */
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='company-switch-trigger'][data-state='open'],
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:hover,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger']:active,
.choy-shell[data-shell-mode='app'] [data-testid='auth-user-menu-trigger'][data-state='open'] {
  background-color: var(--choy-shell-chrome-hover) !important;
}

/* Defeat kit icon-mode size-control shrink — fill the shared chrome band. */
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='company-switch-trigger'],
.choy-shell[data-shell-mode='app']
  [data-slot='sidebar'][data-collapsible='icon']
  [data-testid='auth-user-menu-trigger'] {
  width: 100% !important;
  height: 100% !important;
  min-height: 0 !important;
  min-width: 0 !important;
  padding-block: 0 !important;
  padding-inline: 0 !important;
  margin-inline: 0 !important;
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
