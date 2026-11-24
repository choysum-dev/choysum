<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <DropdownMenuItem :data-testid="testId" :disabled="disabled" @select="onSelect">
    <RefreshCw class="size-4" aria-hidden="true" />
    {{ label || _t('Refresh') }}
  </DropdownMenuItem>
</template>

<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next';
import DropdownMenuItem from '@/web/web/components/vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ChoyViewOverflowRefreshItem' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewOverflowRefreshItem' });

withDefaults(
  defineProps<{
    testId?: string;
    disabled?: boolean;
    label?: string;
  }>(),
  {
    testId: 'view-overflow-refresh',
    disabled: false,
  },
);

const emit = defineEmits<{
  select: [];
}>();

function onSelect(): void {
  emit('select');
}
</script>
