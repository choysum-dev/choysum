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
    <!--
      Same column as app <main>: header / canvas / footer are siblings.
      Gap and inline gutter live on this frame, not on each band.
    -->
    <div
      class="choy-shell__main flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background"
      data-testid="choy-shell-scroll"
    >
      <ChoyGuestHeader v-if="effectiveShowHeader" :go-home="onBrandClick" :home-href="homeHref">
        <template #header-actions>
          <div data-anchor="choy.shell.header-actions" class="flex items-center gap-2">
            <slot name="header-actions" />
          </div>
        </template>
      </ChoyGuestHeader>
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { KeepAlive, computed } from 'vue'
import { useRouter } from 'vue-router'
import ChoyAppFooter from './ChoyAppFooter.vue'
import ChoyGuestHeader from './ChoyGuestHeader.vue'
import { createGuestHomeNavigate } from './guestHomeNavigate'

/** Guest "Home" is the site root, not the authenticated default land path. */
const GUEST_HOME_HREF = '/'

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

const homeHref = GUEST_HOME_HREF
const onBrandClick = createGuestHomeNavigate(
  () => GUEST_HOME_HREF,
  () => useRouter(),
)

const effectiveShowHeader = computed(() => props.showHeader)
</script>

<style>
/* Match app <main>: tighter section gap, wider shared inline gutter. */
.choy-shell[data-shell-mode='guest'] .choy-shell__main {
  gap: var(--choy-layout-section-gap);
  padding-inline: var(--choy-layout-content-padding);
}

/* Win over the shared .choy-shell__canvas { overflow: visible } rule. */
.choy-shell[data-shell-mode='guest'] .choy-shell__canvas {
  overflow-x: hidden;
  overflow-y: auto;
}
</style>
