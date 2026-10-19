<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" setup>
import { CircleCheck, Info, Loader2, OctagonX, TriangleAlert, X } from 'lucide-vue-next'
import { Toaster as Sonner } from "vue-sonner"
import { cn, type ClassValue } from "../../../../lib/utils"

// Local props — avoid `defineProps<ToasterProps>()` (vue-sonner type import breaks QuickJS FE SFC).
type ToastPosition =
  | 'top-left'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-right'
  | 'top-center'
  | 'bottom-center'
const props = defineProps<{
  class?: ClassValue
  theme?: 'light' | 'dark' | 'system'
  position?: ToastPosition
  richColors?: boolean
  closeButton?: boolean
  expand?: boolean
  duration?: number
  visibleToasts?: number
  hotkey?: string[]
  invert?: boolean
  toastOptions?: Record<string, unknown>
  offset?: string | number
  dir?: 'ltr' | 'rtl' | 'auto'
}>()
</script>

<template>
  <!-- Cast: local props mirror ToasterProps but QuickJS cannot use vue-sonner type imports. -->
  <Sonner
    :class="cn('toaster group', props.class)"
    :style="{
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': 'var(--radius)',
    }"
    v-bind="props as Record<string, unknown>"
  >
    <template #success-icon>
      <CircleCheck class="size-4" />
    </template>
    <template #info-icon>
      <Info class="size-4" />
    </template>
    <template #warning-icon>
      <TriangleAlert class="size-4" />
    </template>
    <template #error-icon>
      <OctagonX class="size-4" />
    </template>
    <template #loading-icon>
      <div>
        <Loader2 class="size-4 animate-spin" />
      </div>
    </template>
    <template #close-icon>
      <X class="size-4" />
    </template>
  </Sonner>
</template>
