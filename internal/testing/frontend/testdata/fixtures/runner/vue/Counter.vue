<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->
<script setup lang="ts">
const FE_QJS_MARKER = 7;

function feQjsLabel(n: number): string {
  return `fe-qjs-${n}`;
}

const label = feQjsLabel(FE_QJS_MARKER);
void label;
</script>

<template>
  <div class="fe-qjs-counter" data-marker="7">{{ label }}</div>
</template>
