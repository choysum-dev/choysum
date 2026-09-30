<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.action-tray"
    data-testid="choy-action-tray"
    role="toolbar"
    :class="
      cn(
        'choy-action-tray flex flex-wrap items-center gap-2',
        'min-h-[var(--choy-control-height)]',
        props.class,
      )
    "
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Dense Admin Action tray: Button row at control height.
 * Callers must keep at most one solid primary Button in the tray.
 */
const props = defineProps<{
  class?: ClassValue;
}>();
</script>
