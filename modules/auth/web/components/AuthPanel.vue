<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="auth-panel flex min-h-0 w-full flex-1 flex-col overflow-y-auto"
    data-testid="auth-panel"
  >
    <div class="mx-auto my-auto flex w-full max-w-sm flex-col items-center gap-6 px-4 py-8">
      <div class="w-full">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * Auth canvas column: form/card slot, vertically centered in the shell main.
 * Scrolls on its own when the card is taller than the viewport (the fill-height
 * page body clips overflow); `my-auto` keeps the top reachable while centering.
 */
defineOptions({ name: 'AuthPanel' });
</script>
