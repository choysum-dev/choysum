<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest: viewport-locked column, no nav rail. -->
  <div
    class="choy-shell choy-shell--guest flex h-full min-h-0 w-full flex-col overflow-hidden bg-background text-foreground"
    data-testid="choy-shell"
    data-shell-mode="guest"
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
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyShellHeader from './ChoyShellHeader.vue'
import { useLayoutStore } from '../../stores/layoutStore'
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath'
import { shellMenuTriggerLabel } from './choyWebShellChrome'

/**
 * Guest shell: header + main canvas + footer, without the app nav rail.
 */
const props = withDefaults(
  defineProps<{
    showHeader?: boolean
    showFooter?: boolean
  }>(),
  {
    showHeader: true,
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
