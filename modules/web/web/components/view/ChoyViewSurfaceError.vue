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
        <div class="flex flex-wrap items-center justify-center gap-2">
          <ChoyButton
            v-if="showCancel"
            type="button"
            variant="outline"
            data-testid="choy-view-surface-error-cancel"
            @click="emit('cancel')"
          >
            {{ cancelLabel }}
          </ChoyButton>
          <ChoyButton
            type="button"
            data-testid="choy-view-surface-error-retry"
            :disabled="retryDisabled || undefined"
            @click="emit('retry')"
          >
            {{ retryLabel }}
          </ChoyButton>
        </div>
      </template>
    </ChoyEmpty>
  </div>
</template>

<script setup lang="ts">
import ChoyButton from '../button/ChoyButton.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ChoyViewSurfaceError' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewSurfaceError' });

withDefaults(
  defineProps<{
    title: string;
    detail: string;
    /** Soft-fail (has prior frame): show Cancel next to Retry. */
    showCancel?: boolean;
    /** When true, Retry is non-interactive (e.g. still offline). */
    retryDisabled?: boolean;
  }>(),
  { showCancel: false, retryDisabled: false },
);

const emit = defineEmits<{
  retry: [];
  cancel: [];
}>();

const retryLabel = _t('Retry');
const cancelLabel = _t('Cancel');
</script>
