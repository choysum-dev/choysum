<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { DropdownMenuGroup } from 'reka-ui';
</script>

<template>
  <DropdownMenuGroup data-slot="dropdown-menu-group">
    <slot />
  </DropdownMenuGroup>
</template>
