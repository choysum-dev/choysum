<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div data-testid="choy-dialog-host" aria-live="polite">
    <div
      v-for="(entry, index) in stack"
      :key="entry.key"
      class="choy-dialog-host__entry"
      :data-dialog-key="entry.key"
      :data-dialog-mode="entry.mode"
      :style="{ '--choy-dialog-stack-index': String(index) }"
    >
      <component
        :is="entry.component"
        v-bind="entry.props"
        :model-value="true"
        @update:model-value="onOpenChange(entry.key, $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { closeDialog, useDialogStack } from '../../composables/dialogService';

/**
 * App-root host for openDialog({ component }).
 * Renders each stack entry; self-contained dialogs keep their own Teleport chrome.
 */
const stackRef = useDialogStack();
const stack = computed(() => stackRef.value);

function onOpenChange(key: string, open: unknown): void {
  if (open === false || open == null) closeDialog(key);
}
</script>
