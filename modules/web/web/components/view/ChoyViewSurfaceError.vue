<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-testid="choy-view-surface-error"
    class="choy-view-surface-error flex min-h-0 w-full flex-1 flex-col items-center justify-center"
  >
    <ChoyEmpty class="w-full !gap-3 !p-4 md:!p-6" :title="title" :description="detail">
      <template #actions>
        <ChoyButton type="button" data-testid="choy-view-surface-error-retry" @click="emit('retry')">
          {{ retryLabel }}
        </ChoyButton>
      </template>
    </ChoyEmpty>
  </div>
</template>

<script setup lang="ts">
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ChoyViewSurfaceError' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewSurfaceError' });

defineProps<{
  title: string;
  detail: string;
}>();

const emit = defineEmits<{
  retry: [];
}>();

const retryLabel = _t('Retry');
</script>
