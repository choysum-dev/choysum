<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyBreadcrumb
    v-if="items.length"
    :items="items"
    class="min-w-0 flex-1"
    data-testid="choy-shell-breadcrumb"
  />
</template>

<script setup lang="ts">
import { computed, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ChoyBreadcrumb, { type ChoyBreadcrumbItem } from '../view/ChoyBreadcrumb.vue'
import { translateTerm } from '../../i18n'
import { useBreadcrumbStore } from '../../stores/breadcrumbStore'

/**
 * Product-shell header trail from breadcrumbStore (menu root + nested pages).
 */
defineOptions({ name: 'ChoyShellBreadcrumb' })

let items: ComputedRef<ChoyBreadcrumbItem[]> = computed(() => [])

try {
  const store = useBreadcrumbStore()
  let composer: { t: (...args: any[]) => unknown } | null = null
  try {
    composer = useI18n({ useScope: 'global' }) as any
  } catch {
    composer = null
  }
  items = computed(() => {
    const stack = store.breadcrumbStack || []
    return stack.map((entry, index) => {
      const label = composer
        ? String(translateTerm(composer, entry.titleText, entry.title) || entry.title || '')
        : String(entry.title || '')
      const isLast = index === stack.length - 1
      return {
        label,
        to: !isLast && entry.clickable && entry.path ? entry.path : undefined,
      }
    })
  })
} catch {
  items = computed(() => [])
}
</script>
