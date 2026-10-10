<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Guest mobile nav: Popover + hamburger, matching shadcn-vue SiteHeader MobileNav. -->
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <ChoyButton
        variant="ghost"
        type="button"
        :class="
          cn(
            'h-8 items-center justify-start gap-2.5 !px-0 hover:bg-transparent focus-visible:bg-transparent focus-visible:ring-0 active:bg-transparent',
            props.class,
          )
        "
        :aria-label="menuLabel"
        :aria-expanded="open"
        data-testid="choy-guest-mobile-nav"
      >
        <div class="relative flex h-8 w-4 items-center justify-center" aria-hidden="true">
          <div class="relative size-4">
            <span
              :class="
                cn(
                  'absolute left-0 block h-0.5 w-4 bg-foreground transition-all duration-100',
                  open ? 'top-[0.4rem] -rotate-45' : 'top-1',
                )
              "
            />
            <span
              :class="
                cn(
                  'absolute left-0 block h-0.5 w-4 bg-foreground transition-all duration-100',
                  open ? 'top-[0.4rem] rotate-45' : 'top-2.5',
                )
              "
            />
          </div>
        </div>
        <span class="flex h-8 items-center text-lg font-medium leading-none">
          {{ menuLabel }}
        </span>
      </ChoyButton>
    </PopoverTrigger>
    <PopoverContent
      class="choy-guest-mobile-nav__panel overflow-y-auto rounded-none border-none bg-background/90 p-0 shadow-none outline-none backdrop-blur duration-100"
      align="start"
      side="bottom"
      :align-offset="-16"
      :side-offset="14"
      :style="panelStyle"
    >
      <div class="flex flex-col gap-12 overflow-auto px-6 py-6">
        <div class="flex flex-col gap-4">
          <div class="text-sm font-medium text-muted-foreground">
            {{ menuLabel }}
          </div>
          <div class="flex flex-col gap-3">
            <a
              v-for="item in items"
              :key="item.id"
              :href="item.href"
              class="text-2xl font-medium text-foreground no-underline hover:opacity-80"
              :data-testid="`choy-guest-mobile-nav-${item.id}`"
              @click="onNavigate($event, item)"
            >
              {{ item.label }}
            </a>
          </div>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ChoyButton from './ChoyButton.vue'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '../vendor/ui/popover'
import { cn, type ClassValue } from '../../lib/utils'
import { createTranslate } from '../../i18n'
import { isUnmodifiedPrimaryClick } from './guestNavActivate'

export type ChoyGuestNavItem = {
  id: string
  label: string
  href: string
  onSelect: () => void
}

defineOptions({ name: 'ChoyGuestMobileNav' })

const props = defineProps<{
  items: ChoyGuestNavItem[]
  class?: ClassValue
}>()

const open = ref(false)
const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyGuestMobileNav' })
const menuLabel = computed(() => _t('Menu'))

/**
 * Fill the remaining viewport under the header (shadcn-vue MobileNav).
 * Inline size beats kit PopoverContent defaults (`w-72` / content-sized height);
 * `cn` here does not twMerge, so class-only overrides are unreliable.
 */
const panelStyle = {
  width: 'var(--reka-popper-available-width, 100vw)',
  height: 'var(--reka-popper-available-height, calc(100dvh - 3.5rem))',
  maxWidth: '100vw',
}

function onNavigate(e: MouseEvent, item: ChoyGuestNavItem) {
  if (!isUnmodifiedPrimaryClick(e)) return
  e.preventDefault()
  open.value = false
  item.onSelect()
}
</script>
