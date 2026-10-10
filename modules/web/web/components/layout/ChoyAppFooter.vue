<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <p
    class="m-0 text-center text-xs leading-snug text-muted-foreground"
    :class="attrsClass"
    data-testid="choy-app-footer"
  >
    {{ line }}
  </p>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import { shellPoweredByLine } from './choyWebShellChrome';

defineOptions({ name: 'ChoyAppFooter', inheritAttrs: false });

const attrs = useAttrs();
const attrsClass = computed(() => attrs.class);

const line = computed(() =>
  shellPoweredByLine(String((import.meta as ImportMeta).env?.CHOYSUM_APP_VERSION || '').trim() || 'dev'),
);
</script>
