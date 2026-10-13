<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuRoot } from 'reka-ui';

const open = defineModel<boolean>('open', { default: false });

withDefaults(
  defineProps<{
    /** When false, no dismiss overlay — needed for hover-open menus that share the pointer with the trigger. */
    modal?: boolean;
  }>(),
  { modal: true },
);
</script>

<template>
  <DropdownMenuRoot v-model:open="open" :modal="modal" data-slot="dropdown-menu">
    <slot />
  </DropdownMenuRoot>
</template>
