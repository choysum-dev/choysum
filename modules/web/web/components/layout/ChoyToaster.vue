<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Product toast host over the L2 sonner kit. Kit keeps upstream defaults;
    product error icon + close-button hover tokens live here.
  -->
  <Toaster class="choy-toaster" rich-colors close-button v-bind="attrs">
    <template #error-icon>
      <!-- Alert mark for error status; close-button already uses the X glyph. -->
      <CircleAlert class="size-4" />
    </template>
  </Toaster>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue';
import { CircleAlert } from 'lucide-vue-next';
import Toaster from '../vendor/ui/sonner/Sonner.vue';

defineOptions({ inheritAttrs: false });

const attrs = useAttrs();
</script>

<style>
/*
 * Close-button: vue-sonner hover uses --gray2/--gray5 and washes out richColors
 * surfaces. Match the toast level tokens instead. Kit class is `choy-sonner`;
 * this host adds `choy-toaster` via class.
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
