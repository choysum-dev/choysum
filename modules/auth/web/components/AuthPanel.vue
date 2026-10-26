<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="auth-panel mx-auto flex w-full max-w-sm flex-1 flex-col items-center justify-center gap-6 px-4 py-8"
    data-testid="auth-panel"
  >
    <div class="w-full">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * Auth canvas column: form/card slot, vertically centered in the shell main.
 */
defineOptions({ name: 'AuthPanel' });
</script>
