<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest / auth: viewport-locked column, no sidebar. -->
  <div
    v-if="isAuthPage || !sidebarAllowed"
    class="choy-shell flex h-full min-h-0 w-full flex-col overflow-hidden bg-background text-foreground"
    :class="isAuthPage ? 'choy-shell--auth' : undefined"
    data-testid="choy-shell"
    data-shell-mode="canvas"
  >
    <ChoyShellHeader
      v-if="effectiveShowHeader"
      :show-sidebar-chrome="false"
      :menu-trigger-label="menuTriggerLabel"
      :go-home="onBrandClick"
    >
      <template #header-actions>
        <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
          <slot name="header-actions" />
        </div>
      </template>
    </ChoyShellHeader>
    <div
      class="choy-shell__main-inner flex min-h-0 flex-1 flex-col overflow-y-auto bg-background"
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
    <footer
      v-if="showFooter"
      class="shrink-0 px-4 py-4"
      data-testid="choy-shell-footer"
    >
      <ChoyAppFooter />
      <slot name="footer" />
    </footer>
  </div>

  <!-- Product shell: full-bleed header, nav rail, viewport-locked footer. -->
  <SidebarProvider
    v-else
    class="choy-shell flex h-full min-h-0 w-full flex-col overflow-hidden"
    data-testid="choy-shell"
    data-shell-mode="sidebar"
  >
    <ChoyShellHeader
      v-if="effectiveShowHeader"
      :show-sidebar-chrome="true"
      :menu-trigger-label="menuTriggerLabel"
      :go-home="onBrandClick"
    >
      <template #header-actions>
        <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
          <slot name="header-actions" />
        </div>
      </template>
    </ChoyShellHeader>
    <div class="choy-shell__body flex min-h-0 flex-1 overflow-hidden">
      <ChoySidebarBridge />
      <Sidebar collapsible="icon" side="left">
        <SidebarContent>
          <slot name="aside">
            <ChoySidebarNav />
          </slot>
        </SidebarContent>
        <SidebarRail />
      </Sidebar>
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
import { KeepAlive, computed, type ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyShellHeader from './ChoyShellHeader.vue'
import ChoySidebarNav from './ChoySidebarNav.vue'
import ChoySidebarBridge from './ChoySidebarBridge.vue'
import {
  Sidebar,
  SidebarContent,
  SidebarInset,
  SidebarProvider,
  SidebarRail,
} from '../vendor/ui/sidebar/index'
import { useLayoutStore } from '../../stores/layoutStore'
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/**
 * Product shell: header + optional nav rail + viewport-locked footer.
 * Auth pages keep the top bar and footer, and hide the rail.
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

let isAuthPage: ComputedRef<boolean> = computed(() => false)
let onBrandClick = () => {}
let layoutStore: ReturnType<typeof useLayoutStore> | null = null
let tLayout: (key: string, values?: Record<string, unknown>) => string = (key) => key

try {
  const route = useRoute()
  if (route) {
    isAuthPage = computed(() => !!route.meta?.isAuthPage)
  }
} catch {
  // Unit mounts may omit vue-router.
}

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
const sidebarAllowed = computed(() => props.showSidebar && !isAuthPage.value)

const menuTriggerLabel = computed(() =>
  shellMenuTriggerLabel({
    isMobile: !!layoutStore?.isMobile,
    railCollapsed: layoutStore?.sidebarMode === 'collapsed',
    t: (key) => tLayout(key),
  }),
)
</script>
