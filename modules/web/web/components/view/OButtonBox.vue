<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Evaluate slot body during render (not a cached computed) so late-added children remount the shell. -->
  <div v-if="hasContent()" class="o-button-box">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { useSlots } from 'vue';
import { slotHasContent } from '@/web/web/components/view/ostatinfo_helpers';

defineOptions({ name: 'OButtonBox' });

const slots = useSlots();

/** Empty default slot → do not render the flex shell (D3 / D9). */
function hasContent() {
  return slotHasContent(slots.default?.() as unknown[] | undefined);
}
</script>

<style scoped>
.o-button-box {
  display: flex;
  flex-wrap: wrap;
  align-items: stretch;
  gap: 8px;
}
</style>
