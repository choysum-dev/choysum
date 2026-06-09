<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-breadcrumb">
    <ElBreadcrumb separator="/">
      <ElBreadcrumbItem
        v-for="(crumb, index) in breadcrumbs"
        :key="crumb.path"
        :to="crumb.clickable ? crumb.path : undefined"
        @click="navigateTo(crumb, index)"
      >
        {{ crumb.displayTitle }}
      </ElBreadcrumbItem>
    </ElBreadcrumb>
  </div>
</template>

<script setup lang="ts">
import { ElBreadcrumb, ElBreadcrumbItem } from 'element-plus';
import { useBreadcrumb } from '../../composables/useBreadcrumb';

const { breadcrumbs, navigateTo } = useBreadcrumb();
</script>
