<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <p
    class="m-0 inline-flex items-center justify-center gap-1.5 text-center text-xs leading-snug text-muted-foreground"
    :class="attrsClass"
    data-testid="choy-app-footer"
  >
    <img
      :src="logoUrl"
      alt=""
      class="size-3.5 shrink-0 opacity-80"
      width="14"
      height="14"
      data-testid="choy-app-footer-logo"
    />
    <span>{{ line }}</span>
  </p>
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import logoUrl from '../../assets/logo-32.png';
import { shellPoweredByLine } from './choyWebShellChrome';

defineOptions({ name: 'ChoyAppFooter', inheritAttrs: false });

const attrs = useAttrs();
const attrsClass = computed(() => attrs.class);

const line = computed(() =>
  shellPoweredByLine(String((import.meta as ImportMeta).env?.CHOYSUM_APP_VERSION || '').trim() || 'dev'),
);
</script>
