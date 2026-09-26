<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DialogRoot } from 'reka-ui';

const open = defineModel<boolean>('open', { default: false });
</script>

<template>
  <DialogRoot v-model:open="open" data-slot="dialog">
    <slot />
  </DialogRoot>
</template>
