<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->
<script setup lang="ts">
import { ref } from 'vue';

const count = ref(0);
const marker = 42;

function bump() {
  count.value += 1;
}
</script>

<template>
  <button class="bump" type="button" @click="bump">{{ count }}</button>
  <span class="marker">{{ marker }}</span>
</template>
