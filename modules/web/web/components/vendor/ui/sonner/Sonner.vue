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
const props = withDefaults(
  defineProps<{
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
  }>(),
  {
    richColors: true,
    closeButton: true,
    position: 'bottom-right',
  },
)
</script>

<template>
  <!-- Cast: local props mirror ToasterProps but QuickJS cannot use vue-sonner type imports. -->
  <Sonner
    :class="cn('toaster group choy-sonner', props.class)"
    :style="{
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': 'var(--radius)',
      '--success-bg': 'var(--choy-color-success-subtle)',
      '--success-border': 'var(--choy-color-success)',
      '--success-text': 'var(--choy-color-foreground)',
      '--info-bg': 'var(--choy-color-info-subtle)',
      '--info-border': 'var(--choy-color-info)',
      '--info-text': 'var(--choy-color-foreground)',
      '--warning-bg': 'var(--choy-color-warning-subtle)',
      '--warning-border': 'var(--choy-color-warning)',
      '--warning-text': 'var(--choy-color-foreground)',
      '--error-bg': 'var(--choy-color-danger-subtle)',
      '--error-border': 'var(--destructive)',
      '--error-text': 'var(--choy-color-foreground)',
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

<style>
/* Level chrome for richColors toasts — bind sonner data-type to product tokens. */
.choy-sonner [data-sonner-toast][data-type='success'] {
  background: var(--choy-color-success-subtle) !important;
  border-color: var(--choy-color-success) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-sonner [data-sonner-toast][data-type='success'] [data-icon] {
  color: var(--choy-color-success);
}
.choy-sonner [data-sonner-toast][data-type='info'] {
  background: var(--choy-color-info-subtle) !important;
  border-color: var(--choy-color-info) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-sonner [data-sonner-toast][data-type='info'] [data-icon] {
  color: var(--choy-color-info);
}
.choy-sonner [data-sonner-toast][data-type='warning'] {
  background: var(--choy-color-warning-subtle) !important;
  border-color: var(--choy-color-warning) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-sonner [data-sonner-toast][data-type='warning'] [data-icon] {
  color: var(--choy-color-warning);
}
.choy-sonner [data-sonner-toast][data-type='error'] {
  background: var(--choy-color-danger-subtle) !important;
  border-color: var(--destructive) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-sonner [data-sonner-toast][data-type='error'] [data-icon] {
  color: var(--destructive);
}
</style>
