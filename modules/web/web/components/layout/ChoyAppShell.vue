<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- App shell: full-bleed header, nav rail, viewport-locked footer. -->
  <SidebarProvider
    class="choy-shell flex h-full min-h-0 w-full flex-col overflow-hidden"
    data-testid="choy-shell"
    data-shell-mode="app"
  >
    <ChoyShellHeader
      v-if="effectiveShowHeader"
      :show-sidebar-chrome="showSidebar"
      :menu-trigger-label="menuTriggerLabel"
      :go-home="onBrandClick"
    >
      <template #header-actions>
        <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
          <slot name="header-actions" />
        </div>
      </template>
    </ChoyShellHeader>
    <div class="choy-shell__body relative flex min-h-0 flex-1 overflow-hidden">
      <template v-if="showSidebar">
        <ChoySidebarBridge />
        <Sidebar collapsible="icon" side="left" class="overflow-hidden">
          <div
            class="flex h-full min-h-0 w-full flex-row"
            data-testid="choy-shell-nav-split"
          >
            <ChoyAppRail />
            <div
              class="bg-sidebar flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
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
          <SidebarRail />
        </Sidebar>
      </template>
      <SidebarInset class="min-h-0 min-w-0 overflow-hidden">
        <div
          class="choy-shell__main-inner flex min-h-0 flex-1 flex-col overflow-y-auto bg-muted/30"
          data-testid="choy-shell-canvas"
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
      </SidebarInset>
    </div>
    <footer
      v-if="showFooter"
      class="shrink-0 px-4 py-4"
      data-testid="choy-shell-footer"
    >
      <ChoyAppFooter />
      <slot name="footer" />
    </footer>
  </SidebarProvider>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyShellHeader from './ChoyShellHeader.vue'
import ChoyAppRail from './ChoyAppRail.vue'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import {
  Sidebar,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/**
 * App shell: header + nav rail + main canvas + footer for authenticated routes.
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

let onBrandClick = () => {}
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key

try {
  const router = useRouter()
  if (router) {
    onBrandClick = () => {
      void router.push(resolveRuntimeDefaultLandPath())
    }
  }
} catch {
  onBrandClick = () => {}
}

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
.choy-shell[data-shell-mode='app'] {
  --sidebar-width: calc(var(--sidebar-width-icon) + 13.5rem) !important;
}

/* Sit under the product header: kit default is viewport-fixed h-svh.
   Unscoped style: plain descendant selectors (no :deep). */
.choy-shell[data-shell-mode='app'] .choy-shell__body [data-slot='sidebar'] > .fixed {
  position: absolute;
  height: 100%;
  border-right-width: 0;
}

/* Kit inner shell is flex-col; keep L1 rail and L2/L3 pane side by side. */
.choy-shell[data-shell-mode='app']
  .choy-shell__body
  [data-slot='sidebar']
  > .fixed
  > [data-sidebar='sidebar'] {
  flex-direction: row;
}
</style>
