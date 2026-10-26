<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Thin wrapper: same UI as varchar, always strip char fixed-length trailing pads. -->
  <ChoyVarcharField v-bind="$attrs" :trim-trailing-spaces="true" />
</template>

<script setup lang="ts">
import ChoyVarcharField from './ChoyVarcharField.vue';

defineOptions({ name: 'ChoyCharField', inheritAttrs: false });
</script>
