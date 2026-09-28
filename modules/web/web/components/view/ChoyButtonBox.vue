<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div v-if="hasContent()" class="choy-button-box flex flex-wrap items-stretch gap-2">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { useSlots } from 'vue';
import { slotHasContent } from '@/web/web/components/view/ostatinfo_helpers';

defineOptions({ name: 'ChoyButtonBox' });

const slots = useSlots();

function hasContent() {
  return slotHasContent(slots.default?.() as unknown[] | undefined);
}
</script>
