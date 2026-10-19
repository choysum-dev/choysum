<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { AlertDialogRoot } from "reka-ui"

const open = defineModel<boolean>("open", { default: false })
</script>

<template>
  <AlertDialogRoot v-model:open="open" data-slot="alert-dialog">
    <slot />
  </AlertDialogRoot>
</template>
