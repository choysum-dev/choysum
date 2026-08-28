<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dropdown
    v-if="visibleItems.length"
    trigger="click"
    placement="bottom-end"
    @command="onCommand"
  >
    <el-button
      text
      class="o-page-io-menu__trigger"
      :aria-label="menuAriaLabel"
      data-test="page-io-menu-trigger"
    >
      <el-icon :size="18"><Setting /></el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="item in visibleItems"
          :key="item.key"
          :command="item.key"
          :disabled="item.disabled"
          :data-test="`page-io-menu-${item.key}`"
        >
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Setting } from '@element-plus/icons-vue';
import { createTranslate } from '@/web/web/i18n';
import type { PageIoMenuItem } from '@/web/web/composables/recordIoTypes';

defineOptions({ name: 'OPageIoMenu' });

const props = defineProps<{
  items: PageIoMenuItem[];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/page/OPageIoMenu' });
const menuAriaLabel = _t('Import and export');

const visibleItems = computed(() => props.items.filter(item => !item.hidden));

function onCommand(key: string) {
  const item = props.items.find(entry => entry.key === key);
  item?.onClick();
}
</script>

<style scoped>
.o-page-io-menu__trigger {
  padding: 4px 8px;
}
</style>
