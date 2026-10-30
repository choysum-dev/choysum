<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyBreadcrumb
    v-if="items.length"
    :items="items"
    class="flex min-h-control min-w-0 flex-1 items-center [&_[data-slot=breadcrumb-list]]:leading-none"
    data-testid="choy-shell-breadcrumb"
  />
</template>

<script setup lang="ts">
import { computed, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ChoyBreadcrumb, { type ChoyBreadcrumbItem } from '../view/ChoyBreadcrumb.vue'
import { translateTerm } from '../../i18n'
import { useBreadcrumbStore } from '../../stores/breadcrumbStore'
import { isMenuNavigatedCrumb } from '../../stores/breadcrumbStore/breadcrumb_helpers'
import { useMenu } from '../../composables/useMenu'

/**
 * Product-shell header trail from breadcrumbStore (app + folders + leaf + pages).
 * App / pathless folder crumbs call navigateTo(menuId); menu/page crumbs use RouterLink.
 */
defineOptions({ name: 'ChoyShellBreadcrumb' })

function buildBreadcrumbItems(): ComputedRef<ChoyBreadcrumbItem[]> {
  try {
    const store = useBreadcrumbStore()
    let composer: { t: (...args: any[]) => unknown } | null = null
    try {
      composer = useI18n({ useScope: 'global' }) as any
    } catch {
      composer = null
    }

    let menuApi: ReturnType<typeof useMenu> | null = null
    try {
      menuApi = useMenu()
    } catch {
      menuApi = null
    }

    function onMenuSelect(menuId: string | undefined) {
      if (!menuId || !menuApi) return
      void menuApi.navigateTo(menuId)
    }

    return computed(() => {
      const stack = store.breadcrumbStack || []
      return stack.map((entry, index) => {
        const label = composer
          ? String(translateTerm(composer, entry.titleText, entry.title) || entry.title || '')
          : String(entry.title || '')
        const isLast = index === stack.length - 1
        if (isLast || !entry.clickable) {
          return { label }
        }
        if (isMenuNavigatedCrumb(entry)) {
          return {
            label,
            onSelect: () => onMenuSelect(entry.menuId),
          }
        }
        return {
          label,
          to: entry.path || undefined,
        }
      })
    })
  } catch {
    return computed(() => [])
  }
}

const items = buildBreadcrumbItems()
</script>
