<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest top bar: SiteHeader density; desktop NavigationMenu + mobile Popover nav. -->
  <header
    class="choy-guest-header sticky top-0 z-50 w-full shrink-0 bg-background/95 backdrop-blur"
    :style="headerStyle"
    data-testid="choy-shell-header"
  >
    <div class="mx-auto flex h-full w-full max-w-screen-2xl items-center gap-2">
        <ChoyGuestMobileNav :items="navItems" class="flex lg:hidden" />
        <NavigationMenu
          :viewport="false"
          class="hidden !max-w-none flex-none justify-start lg:flex"
        >
          <NavigationMenuList class="!gap-0">
            <NavigationMenuItem v-for="item in navItems" :key="item.id">
              <NavigationMenuLink
                :class="navLinkClass"
                :href="item.href"
                :data-testid="`choy-guest-nav-${item.id}`"
                @click="onNavActivate($event, item)"
              >
                {{ item.label }}
              </NavigationMenuLink>
            </NavigationMenuItem>
          </NavigationMenuList>
        </NavigationMenu>
        <div class="ml-auto flex items-center gap-2 md:flex-1 md:justify-end">
          <ChoyShellHeaderTools />
          <Separator
            v-if="$slots['header-actions']"
            orientation="vertical"
            class="mx-1 hidden !h-4 lg:block"
            data-testid="choy-shell-header-sep"
          />
          <slot name="header-actions" />
        </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ChoyGuestMobileNav, { type ChoyGuestNavItem } from './ChoyGuestMobileNav.vue'
import ChoyShellHeaderTools from './ChoyShellHeaderTools.vue'
import { Separator } from '../vendor/ui/separator'
import {
  NavigationMenu,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
} from '../vendor/ui/navigation-menu'
import { cn } from '../../lib/utils'
import { createTranslate } from '../../i18n'
import { isUnmodifiedPrimaryClick } from './guestNavActivate'

defineOptions({ name: 'ChoyGuestHeader' })

const props = defineProps<{
  goHome: () => void
  /** Site-root href for Home anchors (modified clicks stay native). */
  homeHref: string
}>()

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyGuestHeader' })

/** Same chrome band height as AppShell poles (header / footer). */
const headerStyle = {
  height: 'var(--choy-layout-chrome-height)',
  paddingTop: '0',
  paddingBottom: '0',
  boxSizing: 'border-box' as const,
}

/** Ghost-button density matching shadcn-vue MainNav (`Button variant="ghost" size="sm"`). */
const navLinkClass = cn(
  'inline-flex h-8 w-max flex-row items-center justify-center gap-0 rounded-md px-2.5 py-0',
  'bg-transparent text-sm font-medium text-foreground/80 shadow-none',
  'hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground',
  'focus-visible:ring-0 focus-visible:outline-none',
  'data-active:bg-accent/50 data-active:text-foreground',
)

/**
 * Guest primary nav. Shared by desktop NavigationMenu and mobile Popover.
 * Add entries here when new guest destinations ship.
 */
const navItems = computed<ChoyGuestNavItem[]>(() => [
  {
    id: 'home',
    label: _t('Home'),
    href: props.homeHref,
    onSelect: () => props.goHome(),
  },
])

/** In-app SPA push for plain left-clicks; Ctrl/Cmd/etc. keep native anchor behavior. */
function onNavActivate(e: MouseEvent, item: ChoyGuestNavItem) {
  if (!isUnmodifiedPrimaryClick(e)) return
  e.preventDefault()
  item.onSelect()
}
</script>
