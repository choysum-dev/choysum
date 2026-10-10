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
    <ChoyGuestHeader v-if="effectiveShowHeader" :go-home="onBrandClick" :home-href="homeHref">
      <template #header-actions>
        <div
          data-anchor="choy.shell.header-actions"
          class="flex items-center gap-1.5"
        >
          <slot name="header-actions" />
        </div>
      </template>
    </ChoyGuestHeader>
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
      class="shrink-0 bg-background"
      data-testid="choy-shell-footer"
    >
      <div class="w-full px-6">
        <div class="mx-auto flex h-16 max-w-screen-2xl items-center justify-center">
          <ChoyAppFooter
            class="w-full px-1 text-center text-xs leading-loose text-muted-foreground sm:text-sm"
          />
        </div>
        <slot name="footer" />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useRouter } from 'vue-router'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyGuestHeader from './ChoyGuestHeader.vue'
import { createGuestHomeNavigate } from './guestHomeNavigate'
import { resolveRuntimeDefaultLandPath } from '../../router/resolveRuntimeDefaultLandPath'

/**
 * Guest shell: marketing-style header + main canvas + shared footer, without the app nav rail.
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

const homeHref = computed(() => resolveRuntimeDefaultLandPath())
const onBrandClick = createGuestHomeNavigate(
  () => homeHref.value,
  () => useRouter(),
)

const effectiveShowHeader = computed(() => props.showHeader)
</script>
