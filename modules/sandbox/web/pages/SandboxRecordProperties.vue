<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage>
    <SandboxRecordPropertiesDefinitionView />
  </ChoyPage>
</template>

<script setup lang="ts">
import { ChoyPage } from '@/web';
import SandboxRecordPropertiesDefinitionView from '../views/SandboxRecordPropertiesDefinitionView.vue';

defineOptions({ name: 'SandboxRecordPropertiesPage' });
</script>
