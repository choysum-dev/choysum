<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Evaluate slot body during render (not a cached computed) so late-added children remount the shell. -->
  <div v-if="hasContent()" class="choy-button-box flex flex-wrap items-stretch gap-2">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { useSlots } from 'vue';
import { slotHasContent } from '@/web/web/components/view/statInfoHelpers';

defineOptions({ name: 'ChoyButtonBox' });

const slots = useSlots();

/** Empty default slot → do not render the flex shell. */
function hasContent() {
  const raw = slots.default?.();
  if (raw == null) return false;
  const nodes = Array.isArray(raw) ? raw : [raw];
  return slotHasContent(nodes);
}
</script>
