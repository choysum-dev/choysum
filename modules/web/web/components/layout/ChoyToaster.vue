<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script lang="ts" setup>
/**
 * Product toast host. Owns product icon + close-button chrome so vendor/ui/sonner
 * can stay as kit defaults. Mirrors the kit Toaster props surface (QuickJS-safe:
 * no vue-sonner type imports).
 */
import { CircleAlert, CircleCheck, Info, Loader2, TriangleAlert, X } from 'lucide-vue-next';
import { Toaster as Sonner } from 'vue-sonner';
import { cn, type ClassValue } from '../../lib/utils';

type ToastPosition =
  | 'top-left'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-right'
  | 'top-center'
  | 'bottom-center';

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    theme?: 'light' | 'dark' | 'system';
    position?: ToastPosition;
    richColors?: boolean;
    closeButton?: boolean;
    expand?: boolean;
    duration?: number;
    visibleToasts?: number;
    hotkey?: string[];
    invert?: boolean;
    toastOptions?: Record<string, unknown>;
    offset?: string | number;
    dir?: 'ltr' | 'rtl' | 'auto';
  }>(),
  {
    richColors: true,
    closeButton: true,
    position: 'bottom-right',
  },
);
</script>

<template>
  <Sonner
    :class="cn('toaster group choy-toaster', props.class)"
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
      <!-- Alert mark for error status; close-button already uses the X glyph. -->
      <CircleAlert class="size-4" />
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
.choy-toaster [data-sonner-toast][data-type='success'] {
  background: var(--choy-color-success-subtle) !important;
  border-color: var(--choy-color-success) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='success'] [data-icon] {
  color: var(--choy-color-success);
}
.choy-toaster [data-sonner-toast][data-type='info'] {
  background: var(--choy-color-info-subtle) !important;
  border-color: var(--choy-color-info) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='info'] [data-icon] {
  color: var(--choy-color-info);
}
.choy-toaster [data-sonner-toast][data-type='warning'] {
  background: var(--choy-color-warning-subtle) !important;
  border-color: var(--choy-color-warning) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='warning'] [data-icon] {
  color: var(--choy-color-warning);
}
.choy-toaster [data-sonner-toast][data-type='error'] {
  background: var(--choy-color-danger-subtle) !important;
  border-color: var(--destructive) !important;
  color: var(--choy-color-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='error'] [data-icon] {
  color: var(--destructive);
}

/*
 * Close-button: vue-sonner hover uses --gray2/--gray5 and washes out richColors
 * surfaces. Match the toast level tokens instead.
 */
.choy-toaster [data-sonner-toast][data-styled='true'] [data-close-button] {
  background: var(--popover) !important;
  border-color: var(--border) !important;
  color: var(--muted-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-styled='true']:hover [data-close-button]:hover,
.choy-toaster [data-sonner-toast][data-styled='true'] [data-close-button]:hover {
  background: var(--accent) !important;
  border-color: var(--border) !important;
  color: var(--foreground) !important;
}

.choy-toaster [data-sonner-toast][data-type='success'] [data-close-button] {
  background: var(--choy-color-success-subtle) !important;
  border-color: var(--choy-color-success) !important;
  color: var(--muted-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='success']:hover [data-close-button]:hover,
.choy-toaster [data-sonner-toast][data-type='success'] [data-close-button]:hover {
  background: color-mix(in oklab, var(--choy-color-success) 22%, var(--choy-color-success-subtle)) !important;
  border-color: var(--choy-color-success) !important;
  color: var(--foreground) !important;
}

.choy-toaster [data-sonner-toast][data-type='info'] [data-close-button] {
  background: var(--choy-color-info-subtle) !important;
  border-color: var(--choy-color-info) !important;
  color: var(--muted-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='info']:hover [data-close-button]:hover,
.choy-toaster [data-sonner-toast][data-type='info'] [data-close-button]:hover {
  background: color-mix(in oklab, var(--choy-color-info) 22%, var(--choy-color-info-subtle)) !important;
  border-color: var(--choy-color-info) !important;
  color: var(--foreground) !important;
}

.choy-toaster [data-sonner-toast][data-type='warning'] [data-close-button] {
  background: var(--choy-color-warning-subtle) !important;
  border-color: var(--choy-color-warning) !important;
  color: var(--muted-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='warning']:hover [data-close-button]:hover,
.choy-toaster [data-sonner-toast][data-type='warning'] [data-close-button]:hover {
  background: color-mix(in oklab, var(--choy-color-warning) 22%, var(--choy-color-warning-subtle)) !important;
  border-color: var(--choy-color-warning) !important;
  color: var(--foreground) !important;
}

.choy-toaster [data-sonner-toast][data-type='error'] [data-close-button] {
  background: var(--choy-color-danger-subtle) !important;
  border-color: var(--destructive) !important;
  color: var(--muted-foreground) !important;
}
.choy-toaster [data-sonner-toast][data-type='error']:hover [data-close-button]:hover,
.choy-toaster [data-sonner-toast][data-type='error'] [data-close-button]:hover {
  background: color-mix(in oklab, var(--choy-color-danger) 22%, var(--choy-color-danger-subtle)) !important;
  border-color: var(--destructive) !important;
  color: var(--foreground) !important;
}
</style>
